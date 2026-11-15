package tabs

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

type MockTab struct {
	name   string
	theme  core.Theme
	width  int
	height int
}

func NewMockTab(name string, theme core.Theme) *MockTab {
	return &MockTab{name: name, theme: theme}
}

func (m *MockTab) Init() tea.Cmd { return nil }

func (m *MockTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if mTyped, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = mTyped.Width
		m.height = mTyped.Height
	}
	return m, nil
}

func (m *MockTab) View() tea.View {
	return tea.NewView(m.theme.HeaderStyle.Render(m.name + " (Coming soon...)"))
}
