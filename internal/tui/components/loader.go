package components

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"time"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// Spinner is a light wrapper around bubble's spinner.Model.
// Only active, visible spinners schedule ticks.
type Spinner struct {
	spinner spinner.Model
	active  bool
	Message string
}

func NewSpinner(theme core.Theme, msg string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Spinner.FPS = 100 * time.Millisecond
	s.Style = theme.FocusedText
	return Spinner{
		spinner: s,
		active:  false,
		Message: msg,
	}
}

func (s *Spinner) Start() tea.Cmd {
	if s.active {
		return nil
	}
	s.active = true
	return s.spinner.Tick
}

func (s *Spinner) Stop() {
	s.active = false
	fresh := spinner.New()
	fresh.Spinner = s.spinner.Spinner
	fresh.Style = s.spinner.Style
	s.spinner = fresh
}

func (s *Spinner) IsActive() bool {
	return s.active
}

func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	if m, ok := msg.(spinner.TickMsg); ok && s.active && m.ID == s.spinner.ID() {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(m)
		return cmd
	}
	return nil
}

func (s *Spinner) View() string {
	if !s.active {
		return ""
	}
	return s.spinner.View() + " " + s.Message
}

func (s *Spinner) ViewCentered(width int) string {
	if !s.active {
		return ""
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, s.View())
}

func (s Spinner) WithSpinner(sp spinner.Spinner) Spinner {
	s.spinner.Spinner = sp
	s.spinner.Spinner.FPS = 100 * time.Millisecond
	return s
}

// Loader is a tea.Model wrapper for Spinner, supporting modal and inline views.
type Loader struct {
	spinner Spinner
	theme   core.Theme
	style   lipgloss.Style

	width  int
	height int
}

type LoaderOption func(*Loader)

func NewLoader(msg string, theme core.Theme, opts ...LoaderOption) *Loader {
	l := &Loader{
		spinner: NewSpinner(theme, msg),
		theme:   theme,
		style:   theme.ModalStyle(0, theme.Secondary).Align(lipgloss.Center, lipgloss.Center),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func (l *Loader) SetSize(w, h int) {
	l.width = w
	l.height = h
}

func (l *Loader) SetMessage(msg string) {
	l.spinner.Message = msg
}

func (l *Loader) Init() tea.Cmd {
	return l.spinner.Start()
}

func (l *Loader) Start() tea.Cmd {
	return l.spinner.Start()
}

func (l *Loader) Stop() {
	l.spinner.Stop()
}

func (l *Loader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case core.VisibilityMsg:
		if !m.Visible {
			l.Stop()
			return l, nil
		}
		return l, l.Start()
	case tea.KeyPressMsg:
		if m.String() == core.KeyEsc {
			return l, core.Cmd(core.DialogResult{Dialog: l, Close: true})
		}
		return l, nil
	}
	return l, l.spinner.Update(msg)
}

func (l *Loader) View() tea.View {
	content := l.spinner.View()

	style := l.style
	if l.width > 0 {
		style = style.Width(l.width)
	}
	if l.height > 0 {
		style = style.Height(l.height)
	}

	return tea.NewView(style.Render(content))
}
