package components

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

type MsgSelectionModal struct {
	Index int
}

type SelectionModal struct {
	modalTemplate
	title     string
	options   []fmt.Stringer
	cursor    int
	OnSelect  func(int) tea.Msg
	shortcuts selectionShortcuts
}

type selectionShortcuts struct {
	Confirm key.Binding
	Cancel  key.Binding
	Up      key.Binding
	Down    key.Binding
}

func (s selectionShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{s.Up, s.Down, s.Confirm}
}

func (s selectionShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{s.Up, s.Down, s.Confirm, s.Cancel},
	}
}

func NewSelection(title string, options []fmt.Stringer, theme core.Theme) *SelectionModal {
	return &SelectionModal{
		modalTemplate: newTemplate(theme, 40, theme.Secondary),
		title:         title,
		options:       options,
		shortcuts: selectionShortcuts{
			Confirm: core.Submit(),
			Cancel:  core.Cancel(),
			Up:      core.NavigateUp(),
			Down:    core.NavigateDown(),
		},
	}
}

func (m *SelectionModal) Init() tea.Cmd { return nil }

func (m *SelectionModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth = msg.Width
		m.screenHeight = msg.Height
	case tea.KeyPressMsg:
		if len(m.options) == 0 {
			return m, nil
		}
		switch {
		case key.Matches(msg, m.shortcuts.Up):
			m.cursor = (m.cursor - 1 + len(m.options)) % len(m.options)
		case key.Matches(msg, m.shortcuts.Down):
			m.cursor = (m.cursor + 1) % len(m.options)
		case key.Matches(msg, m.shortcuts.Confirm):
			var value tea.Msg = MsgSelectionModal{Index: m.cursor}
			if m.OnSelect != nil {
				value = m.OnSelect(m.cursor)
			}
			return m, core.Cmd(core.DialogResult{Dialog: m, Value: value, Close: true})

		case key.Matches(msg, m.shortcuts.Cancel):
			return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
		}
	}
	return m, nil
}

func (m *SelectionModal) View() tea.View {
	header := m.theme.HeaderStyle.Render(m.title)
	content := header + "\n\n"

	for i, opt := range m.options {
		content += RenderSelector(m.theme, opt.String(), i == m.cursor) + "\n"
	}

	help := m.theme.RenderHelp(m.shortcuts)
	return tea.NewView(m.render(content, help))
}
