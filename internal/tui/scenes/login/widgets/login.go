package widgets

import (
	"fmt"

	"github.com/jadegopher/trabbit/internal/app"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	defaultURL  = "http://localhost:15672"
	defaultUser = "guest"
	defaultPass = "guest"
	rabbitSign  = " _   _\n" +
		"| | | |\n" +
		"| | | |___\n" +
		"|         \\\n" +
		"|       X |\n" +
		" \\________/"
	trabbitBanner = " _             _     _     _ _\n" +
		"| |           | |   | |   (_) |\n" +
		"| |_ _ __ __ _| |__ | |__  _| |_\n" +
		"| __| '__/ _` | '_ \\| '_ \\| | __|\n" +
		"| |_| | | (_| | |_) | |_) | | |_\n" +
		" \\__|_|  \\__,_|_.__/|_.__/|_|\\__|"
)

const (
	loginURLField = iota
	loginUserField
	loginPasswordField
	loginFieldCount
)

type ActionLoginRequested struct {
	URL      string
	Username string
	Password string
}

type Login struct {
	inputs    []textinput.Model
	focused   int
	theme     core.Theme
	shortcuts loginShortcuts
}

type loginShortcuts struct {
	NavigateUp   key.Binding
	NavigateDown key.Binding
	Submit       key.Binding
}

func (b loginShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{b.NavigateUp, b.NavigateDown, b.Submit}
}

func (b loginShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{b.NavigateUp, b.NavigateDown, b.Submit},
	}
}

const LoginComponentName = "login"

func NewLogin(theme core.Theme) *Login {
	m := &Login{
		inputs: make([]textinput.Model, loginFieldCount),
		theme:  theme,

		shortcuts: loginShortcuts{
			NavigateUp:   core.NavigateUp(),
			NavigateDown: core.NavigateDown(),
			Submit:       core.Submit(),
		},
	}

	for i := range m.inputs {
		t := m.theme.NewTextInput()

		switch i {
		case loginURLField:
			t.Focus()
			t.SetValue(defaultURL)
		case loginUserField:
			t.SetValue(defaultUser)
		case loginPasswordField:
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
			t.SetValue(defaultPass)
		}

		m.inputs[i] = t
	}

	return m
}

func (l *Login) Init() tea.Cmd {
	return nil
}

// SetConnection fills the draft, keeping the password in its masked input.
func (l *Login) SetConnection(connection app.Connection) {
	l.inputs[loginURLField].SetValue(connection.URL)
	l.inputs[loginUserField].SetValue(connection.Username)
	l.inputs[loginPasswordField].SetValue(connection.Password)
}

func (l *Login) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, l.shortcuts.Submit):
			return l, core.Cmd(l.submit())

		case key.Matches(msg, l.shortcuts.NavigateUp):
			return l, l.switchFocus(-1)

		case key.Matches(msg, l.shortcuts.NavigateDown):
			return l, l.switchFocus(1)

		}
	}

	return l, l.updateInputs(msg)
}

func (l *Login) switchFocus(delta int) tea.Cmd {
	l.focused += delta

	if l.focused >= len(l.inputs) {
		l.focused = 0
	} else if l.focused < 0 {
		l.focused = len(l.inputs) - 1
	}

	cmds := make([]tea.Cmd, len(l.inputs))
	for i := 0; i < len(l.inputs); i++ {
		if i == l.focused {
			cmds[i] = l.inputs[i].Focus()
			continue
		}
		l.inputs[i].Blur()
	}

	return tea.Batch(cmds...)
}

func (l *Login) updateInputs(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		var cmd tea.Cmd
		l.inputs[l.focused], cmd = l.inputs[l.focused].Update(msg)
		return cmd
	}

	cmds := make([]tea.Cmd, len(l.inputs))
	for i := range l.inputs {
		l.inputs[i], cmds[i] = l.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (l *Login) submit() tea.Msg {
	u := l.inputs[loginURLField].Value()
	user := l.inputs[loginUserField].Value()
	pass := l.inputs[loginPasswordField].Value()

	err := app.ValidateConnection(app.Connection{URL: u, Username: user, Password: pass})
	if err != nil {
		return LoginValidationFailed{Err: err}
	}

	return ActionLoginRequested{
		URL:      u,
		Username: user,
		Password: pass,
	}
}

func (l *Login) View() tea.View {
	return (func() tea.View {
		labels := []string{"URL", "User", "Password"}
		maxLabelLen := 0
		for _, label := range labels {
			if len(label) > maxLabelLen {
				maxLabelLen = len(label)
			}
		}
		labelWidth := maxLabelLen + 2

		mainLayout := core.NewLayout(0, 0, core.Vertical)

		for i := range l.inputs {
			name := fmt.Sprintf("input%d", i)

			rowLayout := core.NewLayout(0, 0, core.Horizontal)
			labelStyle := l.theme.LabelStyle.Width(labelWidth)
			if i == l.focused {
				labelStyle = l.theme.LabelFocused.Width(labelWidth)
			}
			rowLayout.AddFixedView("label", labelWidth, tea.NewView(labelStyle.Render(labels[i]+": ")))

			inputView := tea.NewView(l.inputs[i].View())
			if i == l.focused {
				inputView.Cursor = l.inputs[i].Cursor()
			}
			rowLayout.AddRelativeView("input", 1, inputView)

			mainLayout.AddFixedView(name, 1, rowLayout.Render())
		}

		mainLayout.AddSpacing("spacing")
		mainLayout.AddFixedView("help", 0, tea.NewView(l.theme.RenderHelp(l.shortcuts)))

		return mainLayout.Render()
	})()
}

type LoginValidationFailed struct{ Err error }
