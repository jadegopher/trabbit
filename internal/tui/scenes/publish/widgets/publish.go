package widgets

import (
	"fmt"
	"image"
	"time"

	"charm.land/bubbles/v2/key"
	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

// PublishTarget seeds a draft; submission captures the edited target independently of selection.
type PublishTarget struct{ VHost, Exchange, RoutingKey string }

// PublishRequest carries an immutable submission. Hosts own network commands and
// return their result through Complete; the form never depends on a particular tab.
type PublishRequest struct {
	Form    *PublishForm
	ID      uint64
	Target  PublishTarget
	Options app.PublishOptions
}

type PublishBack struct{ Form *PublishForm }
type PublishExit struct{ Form *PublishForm }

type publishNoticeExpired struct{ generation int }

const publishNoticeDuration = 3 * time.Second

// PublishEvent routes clipboard and cursor command results back to their editor.
type PublishEvent struct {
	Form  *PublishForm
	Value tea.Msg
}

func (p *PublishForm) scope(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		msg := cmd()
		switch value := msg.(type) {
		case tea.BatchMsg:
			commands := make(tea.BatchMsg, len(value))
			for i, child := range value {
				commands[i] = p.scope(child)
			}
			return commands
		case PublishRequest, PublishBack, PublishExit, PublishEvent, core.OpenDialog:
			return msg
		default:
			return PublishEvent{Form: p, Value: msg}
		}
	}
}

const (
	publishFields = iota
	publishPayload
)

type PublishForm struct {
	theme               core.Theme
	routing             *components.TextInput
	exchange            *components.TextInput
	vhost               *components.Select
	delivery, encoding  *components.Select
	properties, headers *components.KeyValues
	fields              *components.FormBody
	payload             *components.TextArea
	formatters          []PayloadFormatter
	width, height       int
	focus               int
	focused, busy       bool
	requestID           uint64
	status              string
	statusKind          core.NoticeKind
	noticeTimer         core.Debouncer
	err                 error
}

func NewPublishForm(theme core.Theme, target PublishTarget, formatters ...PayloadFormatter) *PublishForm {
	p := &PublishForm{theme: theme, formatters: formatters, focus: publishPayload}
	if len(p.formatters) == 0 {
		p.formatters = []PayloadFormatter{JSONFormatter{}}
	}
	p.vhost = components.NewSelect(theme, []string{target.VHost}, 0)
	p.exchange = components.NewTextInput(theme)
	if target.Exchange == "" {
		target.Exchange = app.DefaultExchange
	}
	p.exchange.SetValue(target.Exchange)
	p.exchange.Placeholder = "(default)"
	p.routing = components.NewTextInput(theme)
	p.routing.SetValue(target.RoutingKey)
	p.delivery = components.NewSelect(theme, []string{"1 Non persistent", "2 Persistent"}, 1)
	p.encoding = components.NewSelect(theme, []string{"string", "base64"}, 0)
	p.properties = components.NewKeyValues(theme, "Properties", "Key:", "Value:")
	p.headers = components.NewKeyValues(theme, "Headers", "Key:", "Value:")
	p.fields = components.NewFormBody(
		theme, []components.FormField{
			{Label: "Vhost", Item: p.vhost},
			{Label: "Exchange", Item: p.exchange},
			{Label: "Routing key", Item: p.routing},
			{Label: "Delivery mode", Item: p.delivery},
			{Label: "Encoding", Item: p.encoding},
			{Item: p.properties}, {Item: p.headers},
		},
	)
	p.fields.WrapNavigation = true
	p.payload = components.NewTextArea(theme)
	// Ctrl+B belongs to formatting in this form, so it cannot also move left.
	p.payload.KeyMap.CharacterBackward = core.WithoutShortcuts(p.payload.KeyMap.CharacterBackward, core.Beautify())
	p.payload.EnableRowNumbers()
	p.payload.Blur()
	// Viewport height must not become a limit on pasted logical lines.
	p.payload.MaxHeight = 0
	p.payload.CharLimit = 0
	return p
}

// SetVHosts refreshes choices without replacing the selected queue's preset or
// a choice the user made while the request was running.
func (p *PublishForm) SetVHosts(names []string, err error) {
	if err != nil {
		p.err = fmt.Errorf("load virtual hosts: %w", err)
		return
	}
	current := p.vhost.Value()
	options := []string{current}
	for _, name := range names {
		found := false
		for _, option := range options {
			if option == name {
				found = true
				break
			}
		}
		if !found {
			options = append(options, name)
		}
	}
	p.vhost.Options, p.vhost.Index = options, 0
}

func (p *PublishForm) Target() PublishTarget {
	exchange := p.exchange.Value()
	if exchange == "" {
		exchange = app.DefaultExchange
	}
	return PublishTarget{VHost: p.vhost.Value(), Exchange: exchange, RoutingKey: p.routing.Value()}
}

func (p *PublishForm) Busy() bool { return p.busy }

func (p *PublishForm) Pending(id uint64) bool { return p.busy && p.requestID == id }

func (p *PublishForm) Focus() tea.Cmd {
	p.focused = true
	return p.focusCurrent()
}

func (p *PublishForm) Blur() {
	p.noticeTimer.Cancel()
	p.status = ""
	p.focused = false
	p.fields.Blur()
	p.payload.Blur()
}

func (p *PublishForm) focusCurrent() tea.Cmd {
	p.fields.Blur()
	p.payload.Blur()
	if !p.focused || p.busy {
		return nil
	}
	if p.focus == publishFields {
		return p.scope(p.fields.Focus())
	}
	if p.focus == publishPayload {
		return p.scope(p.payload.Focus())
	}
	return nil
}

// SetSize receives the complete screen, including its header and help rows.
func (p *PublishForm) SetSize(width, height int) {
	p.width, p.height = max(0, width), max(0, height)
	layout := p.geometry()
	p.fields.SetSize(layout.fields.content.Dx(), layout.fields.content.Dy())
	p.payload.SetWidth(max(1, layout.editorText.Dx()))
	p.payload.SetHeight(max(1, layout.editor.Dy()))
}

func (p *PublishForm) move(back bool) tea.Cmd {
	_, cmd := p.fields.Move(back)
	return cmd
}

func (p *PublishForm) submit() tea.Cmd {
	p.noticeTimer.Cancel()
	p.status = ""
	props, err := p.properties.Map()
	if err != nil {
		p.err = err
		return nil
	}
	headers, err := p.headers.Map()
	if err != nil {
		p.err = err
		return nil
	}
	props["delivery_mode"] = p.delivery.Index + 1
	options := app.PublishOptions{
		RoutingKey: p.routing.Value(), Payload: p.payload.Value(), PayloadEncoding: p.encoding.Value(),
		Properties: app.PublishMessageProperties{Properties: props, Headers: headers},
	}
	if err := app.ValidatePublish(p.Target().VHost, options); err != nil {
		p.err = err
		return nil
	}
	p.requestID++
	p.busy, p.err, p.status = true, nil, "Publishing..."
	p.statusKind = core.NoticeInfo
	p.focusCurrent()
	return core.Cmd(PublishRequest{Form: p, ID: p.requestID, Target: p.Target(), Options: options})
}

func (p *PublishForm) beautify() {
	if p.encoding.Value() != "string" {
		p.err = fmt.Errorf("beautify is available for string payloads")
		return
	}
	for _, formatter := range p.formatters {
		formatted, err := formatter.Format(p.payload.Value())
		if err == nil {
			p.payload.SetValue(formatted)
			p.err = nil
			return
		}
	}
	p.err = fmt.Errorf("payload does not match a supported format; text unchanged")
}

// Complete rejects duplicate and stale results. Failed submissions retain all inputs.
func (p *PublishForm) Complete(id uint64, result app.PublishResult, err error) tea.Cmd {
	if !p.busy || id != p.requestID {
		return nil
	}
	p.busy, p.err = false, err
	p.status = "Published • routed"
	p.statusKind = core.NoticeSuccess
	if !result.Routed {
		p.status = "Published • NOT routed: no queue matched the routing key."
		p.statusKind = core.NoticeWarning
	}
	if err != nil {
		p.err = fmt.Errorf("Publish failed: %w", err)
		p.status, p.statusKind = p.err.Error(), core.NoticeError
	}
	timer := p.noticeTimer.Trigger(publishNoticeDuration, func(generation int) tea.Msg {
		return publishNoticeExpired{generation: generation}
	})
	return tea.Batch(p.focusCurrent(), p.scope(timer))
}

// Update keeps arrows in the owning column: fields use vertical navigation,
// while payload arrows and selection stay entirely inside the native editor.
func (p *PublishForm) Update(msg tea.Msg) (cmd tea.Cmd) {
	if event, ok := msg.(PublishEvent); ok {
		if event.Form != p {
			return nil
		}
		msg = event.Value
	}
	defer func() { cmd = p.scope(cmd) }()
	if expired, ok := msg.(publishNoticeExpired); ok {
		if p.noticeTimer.IsValid(expired.generation) {
			p.noticeTimer.Cancel()
			if p.err != nil && p.err.Error() == p.status {
				p.err = nil
			}
			p.status = ""
		}
		return nil
	}
	if !p.focused {
		return nil
	}
	layout := p.geometry()
	switch mouse := msg.(type) {
	case tea.MouseMotionMsg:
		if !p.busy && p.focus == publishPayload {
			local := image.Pt(mouse.X, mouse.Y).Sub(layout.editor.Min)
			p.payload.DragMouseSelection(local.X, local.Y)
		}
		return nil
	case tea.MouseReleaseMsg:
		if !p.busy && p.focus == publishPayload {
			local := image.Pt(mouse.X, mouse.Y).Sub(layout.editor.Min)
			p.payload.DragMouseSelection(local.X, local.Y)
			p.payload.EndMouseSelection()
		}
		return nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		if p.busy {
			return nil
		}
		point := image.Pt(wheel.X, wheel.Y)
		delta := publishWheelScrollRows
		if wheel.Button == tea.MouseWheelUp {
			delta = -delta
		} else if wheel.Button != tea.MouseWheelDown {
			return nil
		}

		switch {
		case point.In(layout.fields.content):
			p.fields.Scroll(delta)
		case point.In(layout.payload.content):
			p.payload.Scroll(delta)
		}
		return nil
	}
	if mouse, ok := msg.(tea.MouseClickMsg); ok {
		p.payload.EndMouseSelection()
		if p.busy || mouse.Button != tea.MouseLeft {
			return nil
		}
		point := image.Pt(mouse.X, mouse.Y)
		switch {
		case point.In(layout.payload.content):
			p.focus = publishPayload
			cmd := p.focusCurrent()
			if point.In(layout.editorText) {
				local := point.Sub(layout.editor.Min)
				p.payload.BeginMouseSelection(local.X, local.Y)
			}
			return cmd
		case point.In(layout.fields.content):
			p.focus = publishFields
			p.payload.Blur()
			return p.fields.Click(point.Y - layout.fields.content.Min.Y)
		}
		return nil
	}

	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, core.ShowHelp()):
			bindings := []core.Shortcut{core.Beautify()}
			bindings = append(bindings, p.payload.Shortcuts()...)
			return core.Cmd(core.OpenDialog{Model: components.NewShortcutsModal(p.theme, "Text editing", bindings)})
		case key.Matches(k, core.Exit()):
			return core.Cmd(PublishExit{Form: p})
		case key.Matches(k, core.SwitchPanel()):
			if p.focus == publishPayload {
				p.focus = publishFields
			} else {
				p.focus = publishPayload
			}
			return p.focusCurrent()
		case key.Matches(k, core.Publish()):
			if !p.busy {
				return p.submit()
			}
		case key.Matches(k, core.Beautify()):
			if !p.busy && p.focus == publishPayload {
				p.beautify()
			}
			return nil
		}
		if p.busy {
			return nil
		}
		if p.focus != publishPayload {
			switch k.String() {
			case core.KeyUp, core.KeyDown:
				return p.move(k.String() == core.KeyUp)
			case core.KeyPgUp:
				p.fields.ScrollPage(true)
				return nil
			case core.KeyPgDown:
				p.fields.ScrollPage(false)
				return nil
			}
		}
	}
	if p.busy {
		return nil
	}
	switch msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		p.err = nil
	}
	if p.focus == publishFields {
		return p.fields.Update(msg)
	}
	if p.focus == publishPayload {
		cmd := p.payload.Update(msg)
		if p.payload.Err != nil {
			p.err = p.payload.Err
		}
		return cmd
	}
	return nil
}

// View composes the header, independently scrolling panels and fixed help area.
func (p *PublishForm) View() tea.View {
	layout := p.geometry()
	if layout.fields.content.Empty() || layout.payload.content.Empty() {
		return core.CropView(tea.NewView(""), p.width, p.height, 0)
	}
	payload := tea.NewView(p.payload.ScrollView(p.theme))
	if p.payload.Focused() {
		payload.Cursor = p.payload.Cursor()
	}
	payloadView := core.NewLayout(layout.payload.content.Dx(), layout.payload.content.Dy(), core.Vertical).
		AddFixedView(
			"title",
			layout.payloadTitle.Dy(),
			core.CropView(
				tea.NewView(p.theme.HeaderStyle.Render("Payload")),
				layout.payloadTitle.Dx(),
				layout.payloadTitle.Dy(),
				0,
			),
		).
		AddFixedView(
			"editor",
			layout.editor.Dy(),
			core.CropView(payload, layout.editor.Dx(), layout.editor.Dy(), 0),
		).Render()
	panels := core.NewLayout(layout.panels.Dx(), layout.panels.Dy(), core.Horizontal).
		AddFixedView(
			"fields",
			layout.fields.bounds.Dx(),
			p.border(p.fields.View(), layout.fields, p.focused && p.focus == publishFields),
		).
		AddFixedView(
			"payload",
			layout.payload.bounds.Dx(),
			p.border(payloadView, layout.payload, p.focused && p.focus == publishPayload),
		).Render()

	header := p.theme.HeaderStyle.Render("Publish message")
	if p.err != nil {
		header = p.theme.RenderNotice(layout.header.Dx(), layout.header.Dy(), p.err.Error(), core.NoticeError)
	} else if p.status != "" {
		header = p.theme.RenderNotice(layout.header.Dx(), layout.header.Dy(), p.status, p.statusKind)
	}
	hints := ansi.Wrap(p.theme.RenderHelp(p), layout.hints.Dx(), " ")
	return core.NewLayout(p.width, p.height, core.Vertical).
		AddFixedView(
			"header",
			layout.header.Dy(),
			core.CropView(tea.NewView(header), layout.header.Dx(), layout.header.Dy(), 0),
		).
		AddFixedView("panels", layout.panels.Dy(), panels).
		AddFixedView(
			"hints",
			layout.hints.Dy(),
			core.CropView(tea.NewView(hints), layout.hints.Dx(), layout.hints.Dy(), 0),
		).Render()
}

func (p *PublishForm) border(view tea.View, panel publishPanelGeometry, focused bool) tea.View {
	view = core.CropView(view, panel.content.Dx(), panel.content.Dy(), 0)
	innerWidth := panel.content.Dx() + publishPanelLeadingPadding
	padded := core.NewLayout(innerWidth, panel.content.Dy(), core.Horizontal).
		AddFixedView(
			"padding",
			publishPanelLeadingPadding,
			core.CropView(tea.NewView(""), publishPanelLeadingPadding, panel.content.Dy(), 0),
		).
		AddRelativeView("content", 1, view).Render()
	offset := 0
	padded.Content = p.theme.RenderScrollContainer(innerWidth, panel.content.Dy(), focused, &offset, padded.Content, 0)
	if padded.Cursor != nil {
		cursor := *padded.Cursor
		// Layout already applied the leading padding; add only the outer border.
		cursor.X += publishPanelBorderCells
		cursor.Y += publishPanelBorderCells
		padded.Cursor = &cursor
	}
	return padded
}

func (p *PublishForm) ShortHelp() []key.Binding {
	keys := []key.Binding{core.Publish(), core.ShowHelp()}
	if p.focus == publishFields {
		if _, ok := p.fields.FocusedItem().(*components.KeyValues); ok {
			keys = append(keys, core.AddKeyValue(), core.DeleteKeyValue())
		}
	}
	return keys
}

func (p *PublishForm) FullHelp() [][]key.Binding { return [][]key.Binding{p.ShortHelp()} }
