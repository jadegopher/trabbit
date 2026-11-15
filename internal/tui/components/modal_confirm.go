package components

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// ConfirmModel is a specialized modal for binary choices (Yes/No).
type ConfirmModel struct {
	modalTemplate
	message   string
	selected  bool
	shortcuts confirmShortcuts
	result    tea.Msg
}

type confirmShortcuts struct {
	Confirm  key.Binding
	Cancel   key.Binding
	Yes      key.Binding
	No       key.Binding
	Navigate key.Binding
}

func (b confirmShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{b.Navigate, b.Confirm}
}

func (b confirmShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{b.Navigate, b.Confirm, b.Cancel},
		{b.Yes, b.No},
	}
}

func NewConfirm(msg string, theme core.Theme) *ConfirmModel {
	return &ConfirmModel{
		modalTemplate: newTemplate(theme, 60, theme.Secondary),
		message:       msg,
		shortcuts: confirmShortcuts{
			Confirm:  core.Submit(),
			Cancel:   core.Cancel(),
			Yes:      core.Yes(),
			No:       core.No(),
			Navigate: core.NavigateChoice(),
		},
	}
}

func (m *ConfirmModel) WithResult(result tea.Msg) *ConfirmModel { m.result = result; return m }

func (m *ConfirmModel) Init() tea.Cmd { return nil }

func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth = msg.Width
		m.screenHeight = msg.Height
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.shortcuts.Yes):
			return m, m.confirm()

		case key.Matches(msg, m.shortcuts.No):
			return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})

		case key.Matches(msg, m.shortcuts.Navigate):
			m.selected = !m.selected

		case key.Matches(msg, m.shortcuts.Confirm):
			if m.selected {
				return m, m.confirm()
			}
			return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
		case key.Matches(msg, m.shortcuts.Cancel):
			return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
		}
	}
	return m, nil
}

func (m *ConfirmModel) confirm() tea.Cmd {
	return core.Cmd(core.DialogResult{Dialog: m, Value: m.result, Close: true})
}

func (m *ConfirmModel) View() tea.View {
	options := lipgloss.JoinHorizontal(
		lipgloss.Center,
		RenderSelector(m.theme, "Yes", m.selected),
		"  ",
		RenderSelector(m.theme, "No", !m.selected),
	)

	header := m.theme.HeaderStyle.Render("Confirm")
	content := header + "\n\n" + m.message + "\n\n" + options

	help := m.theme.RenderHelp(m.shortcuts)

	return tea.NewView(m.render(content, help))
}
