package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// ErrorModal is a specialized modal for displaying error messages.
type ErrorModal struct {
	modalTemplate
	err      error
	shortcut key.Binding
}

func (m *ErrorModal) ShortHelp() []key.Binding {
	return []key.Binding{m.shortcut}
}

func (m *ErrorModal) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.shortcut}}
}

func NewError(err error, theme core.Theme) *ErrorModal {
	width := defaultWidth

	return &ErrorModal{
		modalTemplate: newTemplate(theme, width, theme.Warning),
		err:           err,
		shortcut:      core.CloseAny(),
	}
}

func (m *ErrorModal) Init() tea.Cmd { return nil }

func (m *ErrorModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth = msg.Width
		m.screenHeight = msg.Height
	case tea.KeyPressMsg:
		return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
	}
	return m, nil
}

func (m *ErrorModal) View() tea.View {
	headerText := "Error"
	message := m.err.Error()

	header := m.theme.HeaderStyle.Foreground(m.theme.Warning).Render(headerText)
	content := header + "\n\n" + message

	help := m.theme.RenderHelp(m)
	return tea.NewView(m.render(content, help))
}
