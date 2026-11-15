package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

type FormField struct {
	// Section starts a visual group; headings never become focusable fields.
	Section string
	Label   string
	Item    FormItem
}

// FormAction adds a shortcut handled by the form before input reaches a field.
type FormAction struct {
	Binding key.Binding
	Run     func() tea.Cmd
}

type FormModal struct {
	theme          core.Theme
	title          string
	fields         []FormField
	cursor         int
	width          int
	preferredWidth int

	shortcuts formShortcuts

	OnSubmit       func() (tea.Msg, error)
	Actions        []FormAction
	CloseOnSubmit  bool
	WrapNavigation bool
	ConfirmCancel  bool
	busy           bool
	err            error
	loader         Spinner
	ExtraView      func() string

	labelWidth int

	scrollOffset int
	screenWidth  int
	screenHeight int
}

type formShortcuts struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Submit key.Binding
	Cancel key.Binding
}

func (s formShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{s.Up, s.Down, s.Submit}
}

func (s formShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{s.Up, s.Down, s.Toggle},
		{s.Submit},
	}
}

func NewFormModal(
	theme core.Theme,
	title string,
	fields []FormField,
) *FormModal {
	labelWidth := 0
	for _, f := range fields {
		labelWidth = max(labelWidth, lipgloss.Width(f.Label))
	}
	return &FormModal{
		theme:          theme,
		title:          title,
		fields:         fields,
		width:          core.DefaultModalWidth,
		preferredWidth: core.DefaultModalWidth,
		WrapNavigation: true,
		shortcuts: formShortcuts{
			Up:     core.NavigateUp(),
			Down:   core.NavigateDown(),
			Toggle: core.Toggle(),
			Submit: core.Submit(),
			Cancel: core.Cancel(),
		},
		labelWidth: labelWidth,
		loader:     NewSpinner(theme, "Working..."),
	}
}

func (m *FormModal) Init() tea.Cmd {
	return m.updateFocus()
}

func (m *FormModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(spinner.TickMsg); ok {
		return m, m.loader.Update(msg)
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		if !m.busy {
			v := m.renderContent()
			m.theme.HandleScroll(wheel, &m.scrollOffset, lipgloss.Height(v.Content)-m.bodyHeight(v))
		}
		return m, nil
	}
	defer m.ensureCursorVisible()
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.screenWidth = size.Width
		m.screenHeight = size.Height
		m.resizeWidth()
		return m, nil
	}
	switch v := msg.(type) {
	case core.DialogFeedback:
		m.busy = false
		m.err = v.Err
		m.loader.Stop()
		return m, nil
	case core.VisibilityMsg:
		if !v.Visible {
			m.loader.Stop()
			return m, nil
		}
		if m.busy {
			return m, m.loader.Start()
		}
	}
	if len(m.fields) == 0 {
		return m, nil
	}
	if m.busy {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && key.Matches(keyMsg, m.shortcuts.Cancel) {
			return m, m.requestCancel()
		}
		return m, m.loader.Update(msg)
	}
	switch msgTyped := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth = msgTyped.Width
		m.screenHeight = msgTyped.Height

	case tea.KeyPressMsg:
		for _, action := range m.Actions {
			if key.Matches(msgTyped, action.Binding) && action.Run != nil {
				return m, action.Run()
			}
		}

		switch {
		case key.Matches(msgTyped, m.shortcuts.Cancel):
			return m, m.requestCancel()

		case key.Matches(msgTyped, m.shortcuts.Submit):
			if m.OnSubmit == nil {
				return m, nil
			}
			value, err := m.OnSubmit()
			m.err = err
			if err != nil {
				return m, nil
			}
			m.busy = !m.CloseOnSubmit
			var tick tea.Cmd
			if m.busy {
				tick = m.loader.Start()
			}
			return m, tea.Batch(core.Cmd(core.DialogResult{Dialog: m, Value: value, Close: m.CloseOnSubmit}), tick)

		case key.Matches(msgTyped, m.shortcuts.Up):
			return m, m.handleUp()

		case key.Matches(msgTyped, m.shortcuts.Down):
			return m, m.handleDown()

		case key.Matches(msgTyped, m.shortcuts.Toggle):
			if ok, cmd := m.fields[m.cursor].Item.OnToggle(msgTyped); ok {
				return m, cmd
			}
		}
	}

	var cmds []tea.Cmd
	for i, f := range m.fields {
		if i == m.cursor {
			cmds = append(cmds, f.Item.Update(msg))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *FormModal) handleUp() tea.Cmd {
	if ok, cmd := m.fields[m.cursor].Item.OnUp(); ok {
		return cmd
	}

	if m.cursor > 0 {
		m.cursor--
	} else if m.WrapNavigation {
		m.cursor = len(m.fields) - 1
	}
	if rows, ok := m.fields[m.cursor].Item.(*KeyValues); ok {
		rows.FocusBoundary(true)
	}

	return m.updateFocus()
}

func (m *FormModal) handleDown() tea.Cmd {
	if ok, cmd := m.fields[m.cursor].Item.OnDown(); ok {
		return cmd
	}

	if m.cursor < len(m.fields)-1 {
		m.cursor++
	} else if m.WrapNavigation {
		m.cursor = 0
	}
	if rows, ok := m.fields[m.cursor].Item.(*KeyValues); ok {
		rows.FocusBoundary(false)
	}

	return m.updateFocus()
}

func (m *FormModal) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i, f := range m.fields {
		if i == m.cursor {
			cmds = append(cmds, f.Item.Focus())
		} else {
			f.Item.Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *FormModal) renderContent() tea.View {
	mainLayout := core.NewLayout(m.width, 0, core.Vertical)
	mainLayout.AddFixedView("header", 0, tea.NewView(m.renderHeader()))

	for i, f := range m.fields {
		if f.Section != "" {
			mainLayout.AddFixedView(fmt.Sprintf("section%d", i), 0, tea.NewView(m.renderSection(i)))
		}
		rowLayout := core.NewLayout(m.width, 0, core.Horizontal)
		labelStyle := m.theme.LabelStyle.Width(m.labelWidth)
		if i == m.cursor {
			labelStyle = labelStyle.Foreground(m.theme.Primary)
		}

		if f.Label != "" {
			rowLayout.AddFixedView("label", m.labelWidth, tea.NewView(labelStyle.Render(f.Label)))
		}

		view := f.Item.View(i == m.cursor)
		valueView := tea.NewView(view)

		if i == m.cursor {
			valueView.Cursor = f.Item.Cursor()
		}

		rowLayout.AddRelativeView("value", 1, valueView)

		mainLayout.AddFixedView(fmt.Sprintf("field%d", i), 0, rowLayout.Render())
	}

	if m.ExtraView != nil {
		mainLayout.AddFixedView("extra", 0, tea.NewView(m.ExtraView()))
	}

	if m.err != nil {
		mainLayout.AddFixedView("error", 0, tea.NewView(m.theme.ErrorStyle.Render(m.err.Error())))
	}
	if m.busy {
		mainLayout.AddFixedView("busy", 0, tea.NewView(m.loader.View()))
	}

	return mainLayout.Render()
}

func (m *FormModal) renderHeader() string {
	return m.theme.LabelStyle.Foreground(m.theme.Primary).Render(m.title) + "\n"
}

func (m *FormModal) renderSection(fieldIndex int) string {
	heading := m.theme.HeaderStyle.Render(m.fields[fieldIndex].Section)
	if fieldIndex > 0 {
		heading = "\n" + heading
	}
	return heading
}

func (m *FormModal) requestCancel() tea.Cmd {
	if m.ConfirmCancel {
		return core.Cmd(
			core.ConfirmDialog{
				Owner:   m,
				Message: "Discard changes to this form?",
				Accept:  core.DialogResult{Dialog: m, Close: true},
			},
		)
	}
	return core.Cmd(core.DialogResult{Dialog: m, Close: true})
}

func (m *FormModal) footer() string {
	style := m.theme.ModalStyle(m.width, m.theme.Secondary)
	return ansi.Wrap(m.theme.RenderHelp(m), m.width-style.GetHorizontalFrameSize(), " ")
}

func (m *FormModal) ShortHelp() []key.Binding {
	bindings := m.shortcuts.ShortHelp()
	if len(m.fields) > 0 {
		if _, ok := m.fields[m.cursor].Item.(*KeyValues); ok {
			bindings = append(bindings, core.AddKeyValue(), core.DeleteKeyValue())
		}
	}
	for _, action := range m.Actions {
		bindings = append(bindings, action.Binding)
	}
	if len(m.fields) > 0 {
		switch m.fields[m.cursor].Item.(type) {
		case *Toggle, *Select:
			bindings = append(bindings, m.shortcuts.Toggle)
		}
	}
	return bindings
}

func (m *FormModal) FullHelp() [][]key.Binding {
	groups := m.shortcuts.FullHelp()
	if len(m.fields) > 0 {
		if _, ok := m.fields[m.cursor].Item.(*KeyValues); ok {
			groups = append(groups, []key.Binding{core.AddKeyValue(), core.DeleteKeyValue()})
		}
	}
	for _, action := range m.Actions {
		groups = append(groups, []key.Binding{action.Binding})
	}
	return groups
}

func (m *FormModal) bodyHeight(v tea.View) int {
	style := m.theme.ModalStyle(m.width, m.theme.Secondary)
	available := m.screenHeight - style.GetVerticalFrameSize() - lipgloss.Height(m.footer()) - 1
	if m.screenHeight <= 0 {
		return lipgloss.Height(v.Content)
	}
	return min(lipgloss.Height(v.Content), max(1, available))
}
func (m *FormModal) View() tea.View {
	v := m.renderContent()
	height := m.bodyHeight(v)
	lines := strings.Split(v.Content, "\n")
	offset := min(m.scrollOffset, max(0, len(lines)-height))
	visible := lines[offset:min(len(lines), offset+height)]
	body := strings.Join(visible, "\n")
	style := m.theme.ModalStyle(m.width, m.theme.Secondary)
	innerWidth := m.width - style.GetHorizontalFrameSize()
	clipped := strings.Split(body, "\n")
	for i, line := range clipped {
		clipped[i] = ansi.Truncate(line, innerWidth, "")
	}
	v.Content = strings.Join(clipped, "\n") + "\n\n" + m.footer()
	if v.Cursor != nil {
		c := *v.Cursor
		c.Y -= offset
		if m.busy || c.Y < 0 || c.Y >= height || c.X >= innerWidth {
			v.Cursor = nil
		} else {
			v.Cursor = &c
		}
	}
	result := m.theme.PlaceInModal(v, m.width, m.theme.Secondary)

	result.Content = m.theme.ApplyScrollbar(
		result.Content,
		style.GetPaddingTop()+style.GetBorderTopSize(),
		height,
		len(lines),
		offset,
		style.GetBorderTopForeground(),
	)

	return result
}
func (m *FormModal) ensureCursorVisible() {
	v := m.renderContent()
	height := m.bodyHeight(v)
	target := 0
	if v.Cursor != nil {
		target = v.Cursor.Y
	} else {
		target = lipgloss.Height(m.renderHeader())
		for i, field := range m.fields {
			if field.Section != "" {
				target += lipgloss.Height(m.renderSection(i))
			}
			if i == m.cursor {
				break
			}
			target += lipgloss.Height(field.Item.View(false))
		}
	}
	if m.cursor == 0 && target < height {
		m.scrollOffset = 0
	}
	if target < m.scrollOffset {
		m.scrollOffset = target
	}
	if target >= m.scrollOffset+height {
		m.scrollOffset = target - height + 1
	}
	m.scrollOffset = max(0, min(m.scrollOffset, lipgloss.Height(v.Content)-height))
}

func (m *FormModal) SetWidth(width int) {
	m.preferredWidth = max(20, width)
	m.resizeWidth()
}

func (m *FormModal) resizeWidth() {
	m.width = m.preferredWidth
	if m.screenWidth > 0 {
		m.width = min(m.width, max(20, m.screenWidth-2))
	}
	contentWidth := m.width - m.theme.ModalStyle(m.width, m.theme.Secondary).GetHorizontalFrameSize()
	for _, field := range m.fields {
		if rows, ok := field.Item.(*KeyValues); ok {
			width := contentWidth
			if field.Label != "" {
				width -= m.labelWidth
			}
			rows.SetWidth(width)
		}
	}
}
