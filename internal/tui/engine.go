package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/client"
	"github.com/jadegopher/trabbit/internal/log"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard"
	"github.com/jadegopher/trabbit/internal/tui/scenes/login"
)

const (
	minWidth  = core.MinWidth
	minHeight = 15
)

type dialogReturn struct {
	model    tea.Model
	backdrop []string
}
type Engine struct {
	logger                log.Logger
	settings              *settings.Store
	confirmationReturn    *dialogReturn
	theme                 core.Theme
	scene                 tea.Model
	returnScene           tea.Model
	dialog                tea.Model
	connect               app.Connector
	session               uint64
	width, height         int
	backdrop              []string
	lastDialog, composite string
	quitPending           bool
}
type scopedMsg struct {
	session uint64
	dialog  tea.Model
	value   tea.Msg
}

type sceneMsg struct {
	owner tea.Model
	value tea.Msg
}

// Scene commands retain their owner across full-screen navigation. Background
// results can update the saved scene without being interpreted as active input.
func sceneScoped(cmd tea.Cmd, owner tea.Model) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			commands := make(tea.BatchMsg, len(batch))
			for i, child := range batch {
				commands[i] = sceneScoped(child, owner)
			}
			return commands
		}
		return sceneMsg{owner: owner, value: msg}
	}
}

func scoped(cmd tea.Cmd, session uint64, dialog tea.Model) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			cmds := make(tea.BatchMsg, len(batch))
			for i, c := range batch {
				cmds[i] = scoped(c, session, dialog)
			}
			return cmds
		}
		return scopedMsg{session, dialog, msg}
	}
}

func NewEngine(preferences *settings.Store) *Engine {
	return NewEngineWithConnector(client.Connect, preferences)
}

func NewEngineWithConnector(connect app.Connector, preferences *settings.Store) *Engine {
	return NewEngineWithConnection(connect, preferences, nil)
}

// NewEngineWithConnection starts authentication through the login scene so it
// retains request ownership and presents failures in the editable login form.
func NewEngineWithConnection(connect app.Connector, preferences *settings.Store, connection *app.Connection) *Engine {
	e := &Engine{theme: core.DefaultTheme(), connect: connect, session: 1, settings: preferences, logger: log.DefaultLogger}
	if connection == nil {
		e.scene = login.NewLoginScene(e.theme, connect)
	} else {
		e.scene = login.NewLoginSceneWithConnection(e.theme, connect, *connection)
	}
	return e
}

func (e *Engine) Init() tea.Cmd { return scoped(sceneScoped(e.scene.Init(), e.scene), e.session, nil) }

func (e *Engine) Close() {
	e.quitPending = false
	if scene, ok := e.scene.(interface{ Close() }); ok {
		scene.Close()
	}
	if saved, ok := e.returnScene.(interface{ Close() }); ok {
		saved.Close()
	}
}

func (e *Engine) sceneUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	e.scene, cmd = e.scene.Update(msg)
	return scoped(sceneScoped(cmd, e.scene), e.session, nil)
}

func (e *Engine) visible() bool { return e.width >= minWidth && e.height >= minHeight }

func (e *Engine) capture() {
	e.backdrop = frameLines(e.scene.View().Content, e.width, e.height)
	e.lastDialog = ""
	e.composite = ""
}

func (e *Engine) open(dialog tea.Model, after tea.Cmd) tea.Cmd {
	if dialog == nil {
		return nil
	}
	e.logger.Debug("Opening dialog", log.Component, "engine", "dialog", fmt.Sprintf("%T", dialog), "session", e.session)
	first := e.dialog == nil
	if e.dialog != nil {
		e.dialog.Update(core.VisibilityMsg{Visible: false})
	}
	e.dialog = dialog
	dialog.Update(tea.WindowSizeMsg{Width: e.width, Height: e.height})
	if first {
		e.capture()
	} else {
		e.lastDialog = ""
		e.composite = ""
	}
	return tea.Batch(
		e.sceneUpdate(core.VisibilityMsg{Visible: false}),
		scoped(dialog.Init(), e.session, dialog),
		scoped(after, e.session, nil),
	)
}

func (e *Engine) closeDialog() tea.Cmd {
	if e.dialog == nil {
		return nil
	}
	e.logger.Debug("Closing dialog", log.Component, "engine", "dialog", fmt.Sprintf("%T", e.dialog), "session", e.session)
	e.dialog.Update(core.VisibilityMsg{Visible: false})
	e.dialog = nil
	e.backdrop = nil
	e.lastDialog = ""
	e.composite = ""
	return e.sceneUpdate(core.VisibilityMsg{Visible: e.visible()})
}

func (e *Engine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var origin tea.Model
	if m, ok := msg.(scopedMsg); ok {
		if m.session != e.session || (m.dialog != nil && m.dialog != e.dialog) {
			e.logger.Debug("Ignoring stale command result", log.Component, "engine", "message", fmt.Sprintf("%T", m.value), "session", e.session, "origin_session", m.session)
			return e, nil
		}
		msg = m.value
		origin = m.dialog
	}
	if m, ok := msg.(sceneMsg); ok {
		if m.owner != e.scene {
			if m.owner == e.returnScene {
				_, cmd := e.returnScene.Update(m.value)
				return e, scoped(sceneScoped(cmd, e.returnScene), e.session, nil)
			}
			e.logger.Debug("Ignoring result from closed scene", log.Component, "engine", "scene", fmt.Sprintf("%T", m.owner), "message", fmt.Sprintf("%T", m.value))
			return e, nil
		}
		msg = m.value
	}
	switch m := msg.(type) {
	case core.OpenScreen:
		if m.Model == nil || e.returnScene != nil || e.dialog != nil {
			return e, nil
		}
		e.logger.Debug("Opening screen", log.Component, "engine", "from", fmt.Sprintf("%T", e.scene), "to", fmt.Sprintf("%T", m.Model), "session", e.session)
		hidden := e.sceneUpdate(core.VisibilityMsg{Visible: false})
		e.returnScene, e.scene = e.scene, m.Model
		size := e.sceneUpdate(tea.WindowSizeMsg{Width: e.width, Height: e.height})
		init := scoped(sceneScoped(e.scene.Init(), e.scene), e.session, nil)
		var visibility tea.Cmd
		if !e.visible() {
			visibility = e.sceneUpdate(core.VisibilityMsg{Visible: false})
		}
		return e, tea.Batch(hidden, size, init, visibility)
	case core.CloseScreen:
		if m.Model != e.scene || e.returnScene == nil {
			return e, nil
		}
		e.logger.Debug("Returning to screen", log.Component, "engine", "from", fmt.Sprintf("%T", e.scene), "to", fmt.Sprintf("%T", e.returnScene), "session", e.session)
		if scene, ok := e.scene.(interface{ Close() }); ok {
			scene.Close()
		}
		e.scene, e.returnScene = e.returnScene, nil
		size := e.sceneUpdate(tea.WindowSizeMsg{Width: e.width, Height: e.height})
		return e, tea.Batch(size, e.sceneUpdate(core.VisibilityMsg{Visible: e.visible()}))
	case tea.KeyPressMsg:
		if key.Matches(m, core.Quit()) {
			// Holding the key must not confirm when repeat events are available.
			if m.IsRepeat {
				return e, nil
			}
			if !e.quitPending {
				e.quitPending = true
				e.logger.Debug("Quit confirmation requested", log.Component, "engine")
				return e, nil
			}
			e.Close()
			return e, tea.Quit
		}
		e.quitPending = false
	case tea.PasteMsg, tea.MouseClickMsg, tea.MouseWheelMsg:
		e.quitPending = false
	case tea.WindowSizeMsg:
		wasVisible := e.visible()
		e.width = m.Width
		e.height = m.Height
		cmd := e.sceneUpdate(m)
		if e.dialog != nil {
			_, dc := e.dialog.Update(m)
			if e.visible() {
				if saved := e.confirmationReturn; saved != nil {
					saved.model.Update(m)
					saved.backdrop = frameLines(e.scene.View().Content, e.width, e.height)
					e.backdrop = frameLines(
						overlay(saved.backdrop, saved.model.View().Content, e.width, e.height),
						e.width,
						e.height,
					)
					e.lastDialog = ""
					e.composite = ""
				} else {
					e.capture()
				}
			}
			_, vc := e.dialog.Update(core.VisibilityMsg{Visible: e.visible()})
			return e, tea.Batch(cmd, scoped(dc, e.session, e.dialog), scoped(vc, e.session, e.dialog))
		}
		if wasVisible != e.visible() {
			cmd = tea.Batch(cmd, e.sceneUpdate(core.VisibilityMsg{Visible: e.visible()}))
		}
		return e, cmd
	case login.MsgLoginSuccess:
		e.Close()
		e.returnScene = nil
		e.session++
		e.logger.Info("Opening dashboard", log.Component, "engine", "session", e.session)
		e.dialog = nil
		e.scene = dashboard.NewDashboardScene(e.theme, m.Service, time.Now, e.settings)
		e.scene.Update(tea.WindowSizeMsg{Width: e.width, Height: e.height})
		return e, scoped(sceneScoped(e.scene.Init(), e.scene), e.session, nil)
	case core.ConfirmDialog:
		if e.dialog != m.Owner || e.confirmationReturn != nil {
			return e, nil
		}
		saved := &dialogReturn{model: e.dialog, backdrop: e.backdrop}
		frozen := frameLines(e.View().Content, e.width, e.height)
		dialog := components.NewConfirm(m.Message, e.theme).WithResult(m.Accept)
		cmd := e.open(dialog, nil)
		e.confirmationReturn = saved
		e.backdrop = frozen
		return e, cmd
	case core.OpenDialog:
		return e, e.open(m.Model, m.After)
	case core.ReplaceDialog:
		if e.dialog != m.Old {
			return e, nil
		}
		return e, e.open(m.Model, nil)
	case core.DialogFeedback:
		if saved := e.confirmationReturn; saved != nil && m.Dialog == saved.model {
			_, cmd := saved.model.Update(m)
			return e, scoped(cmd, e.session, saved.model)
		}
		if e.dialog != m.Dialog {
			return e, nil
		}
		_, cmd := e.dialog.Update(m)
		return e, scoped(cmd, e.session, e.dialog)
	case core.DialogResult:
		if saved := e.confirmationReturn; saved != nil && m.Dialog == saved.model && m.Close {
			e.confirmationReturn = nil
			return e, e.closeDialog()
		}
		if e.dialog != m.Dialog {
			return e, nil
		}
		if e.confirmationReturn != nil {
			saved := e.confirmationReturn
			e.confirmationReturn = nil
			e.dialog.Update(core.VisibilityMsg{Visible: false})
			e.dialog = saved.model
			e.backdrop = saved.backdrop
			e.lastDialog = ""
			e.composite = ""
			e.dialog.Update(tea.WindowSizeMsg{Width: e.width, Height: e.height})
			if m.Value != nil {
				return e, e.resultCommand(m.Value)
			}
			_, cmd := e.dialog.Update(core.VisibilityMsg{Visible: e.visible()})
			return e, scoped(cmd, e.session, e.dialog)
		}
		var closeCmd tea.Cmd
		if m.Close {
			closeCmd = e.closeDialog()
		}
		if _, ok := m.Value.(dashboard.ActionLogoutRequested); ok {
			e.logger.Info("Logging out", log.Component, "engine", "session", e.session)
			e.Close()
			e.returnScene = nil
			e.session++
			e.dialog = nil
			e.backdrop = nil
			e.scene = login.NewLoginScene(e.theme, e.connect)
			e.scene.Update(tea.WindowSizeMsg{Width: e.width, Height: e.height})
			return e, scoped(sceneScoped(e.scene.Init(), e.scene), e.session, nil)
		}
		if m.Value != nil {
			return e, tea.Batch(closeCmd, e.sceneUpdate(m))
		}
		return e, closeCmd
	}
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		if !e.visible() {
			return e, nil
		}
		if e.dialog != nil {
			var cmd tea.Cmd
			e.dialog, cmd = e.dialog.Update(msg)
			return e, scoped(cmd, e.session, e.dialog)
		}
	}
	if origin != nil {
		_, cmd := e.dialog.Update(msg)
		return e, scoped(cmd, e.session, e.dialog)
	}
	return e, e.sceneUpdate(msg)
}

func (e *Engine) View() tea.View {
	var view tea.View
	if !e.visible() {
		view = tea.NewView(
			e.theme.HeaderStyle.Render(
				fmt.Sprintf(
					"Terminal size too small\nRequired: %d x %d\nCurrent: %d x %d",
					minWidth,
					minHeight,
					e.width,
					e.height,
				),
			),
		)
	} else if e.dialog != nil {
		v := e.dialog.View()
		if e.composite == "" || v.Content != e.lastDialog {
			e.composite = overlay(e.backdrop, v.Content, e.width, e.height)
			e.lastDialog = v.Content
		}
		view = tea.NewView(e.composite)
		view.Cursor = placedCursor(v, e.width, e.height)
	} else {
		view = e.scene.View()
	}
	if e.quitPending {
		binding := core.ShortcutReference(core.Quit())
		message := fmt.Sprintf("Press %s again to quit", binding.Help().Key)
		view = e.theme.RenderTopNotice(view, e.width, e.height, message, core.NoticeWarning)
	}
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

func (e *Engine) resultCommand(value tea.Msg) tea.Cmd {
	return scoped(core.Cmd(value), e.session, e.dialog)
}
