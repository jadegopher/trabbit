package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	shortcutsModalWidth            = 78
	shortcutsModalBodyRows         = 22
	shortcutsFooterGapRows         = 1
	shortcutsContentPaddingColumns = 1
	shortcutsWheelRows             = 3
)

// ShortcutsModal renders the owning control's bindings with shared help styles.
// Scrolling keeps the complete reference accessible in small terminals.
type ShortcutsModal struct {
	theme                 core.Theme
	title                 string
	bindings              []core.Shortcut
	width, height, offset int
}

func NewShortcutsModal(theme core.Theme, title string, bindings []core.Shortcut) *ShortcutsModal {
	m := &ShortcutsModal{theme: theme, title: title, bindings: bindings, width: shortcutsModalWidth}
	footerRows := len(strings.Split(m.footer(m.width-core.BorderOverhead-shortcutsContentPaddingColumns-core.ScrollbarColumns), "\n"))
	m.height = shortcutsModalBodyRows + core.BorderOverhead + shortcutsFooterGapRows + footerRows
	return m
}

func (m *ShortcutsModal) Init() tea.Cmd { return nil }

func (m *ShortcutsModal) body() (string, int, int) {
	width := max(1, min(shortcutsModalWidth, m.width)-core.BorderOverhead)
	textWidth := max(1, width-shortcutsContentPaddingColumns-core.ScrollbarColumns)
	lines := []string{ansi.Wrap(m.theme.HeaderStyle.Render(m.title), textWidth, " "), ""}
	for _, binding := range m.bindings {
		if binding.Enabled() {
			display := core.ShortcutReference(binding)
			lines = append(lines, ansi.Wrap(m.theme.RenderHelp(core.Shortcuts{display}), textWidth, " "))
		}
	}
	lines = append(lines, "", ansi.Wrap("Command keys and Ctrl+Shift+C depend on terminal support.", textWidth, " "))
	body := m.theme.PlainStyle.PaddingLeft(shortcutsContentPaddingColumns).Render(strings.Join(lines, "\n"))
	footerRows := len(strings.Split(m.footer(textWidth), "\n"))
	availableRows := m.height - core.BorderOverhead - shortcutsFooterGapRows - footerRows
	height := max(1, min(shortcutsModalBodyRows, availableRows, len(strings.Split(body, "\n"))))
	return body, width, height
}

func (m *ShortcutsModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, core.Esc(), core.ShowHelp()):
			return m, core.Cmd(core.DialogResult{Dialog: m, Close: true})
		case key.Matches(msg, core.NavigateUp()):
			m.offset--
		case key.Matches(msg, core.NavigateDown()):
			m.offset++
		case msg.String() == core.KeyPgUp:
			_, _, height := m.body()
			m.offset -= height
		case msg.String() == core.KeyPgDown:
			_, _, height := m.body()
			m.offset += height
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			m.offset -= shortcutsWheelRows
		} else if msg.Button == tea.MouseWheelDown {
			m.offset += shortcutsWheelRows
		}
	}
	body, _, height := m.body()
	m.offset = max(0, min(m.offset, len(strings.Split(body, "\n"))-height))
	return m, nil
}

func (m *ShortcutsModal) footer(width int) string {
	return ansi.Wrap(m.theme.RenderHelp(core.Shortcuts{core.NavigateUp(), core.NavigateDown(), core.Esc()}), width, " ")
}

func (m *ShortcutsModal) View() tea.View {
	body, width, height := m.body()
	textWidth := max(1, width-shortcutsContentPaddingColumns-core.ScrollbarColumns)
	footer := strings.Repeat("\n", shortcutsFooterGapRows) + m.footer(textWidth)
	footer = m.theme.PlainStyle.PaddingLeft(shortcutsContentPaddingColumns).Render(footer)
	// Body and fixed hints share the generic panel's border and scrollbar rules.
	offset := m.offset
	return tea.NewView(m.theme.RenderInsetScrollContainer(width, height, true, &offset, body, 0, footer))
}
