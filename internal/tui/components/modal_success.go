package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// TODO: autoclose

// SuccessModal is a specialized modal for displaying success messages.
type SuccessModal struct {
	modalTemplate
	message  string
	shortcut key.Binding
}

func (m *SuccessModal) ShortHelp() []key.Binding {
	return []key.Binding{m.shortcut}
}

func (m *SuccessModal) FullHelp() [][]key.Binding {
	return [][]key.Binding{{m.shortcut}}
}

func NewSuccess(msg string, theme core.Theme) *SuccessModal {
	return &SuccessModal{
		modalTemplate: newTemplate(theme, defaultWidth, theme.Success),
		message:       msg,
		shortcut:      core.CloseAny(),
	}
}

func (m *SuccessModal) Init() tea.Cmd { return nil }

func (m *SuccessModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth = msg.Width
		m.screenHeight = msg.Height
	case tea.KeyPressMsg:
		return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
	}
	return m, nil
}

func (m *SuccessModal) View() tea.View {
	header := m.theme.LabelStyle.Render("Success")
	content := header + "\n\n" + m.message

	help := m.theme.RenderHelp(m)
	return tea.NewView(m.render(content, help))
}
