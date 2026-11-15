package refresher

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type RefreshStatus int

const (
	RefreshStatusSuccess RefreshStatus = iota
	RefreshStatusPending
	RefreshStatusError
)

const (
	refreshIcon = "↻"
	hoursPerDay = 24
)

type Widget struct {
	theme         core.Theme
	interval      time.Duration
	status        RefreshStatus
	spinner       components.Spinner
	now           func() time.Time
	lastUpdatedAt time.Time
	nextRefreshAt time.Time
}

func NewWidget(
	theme core.Theme, status RefreshStatus,
	now func() time.Time,
	interval time.Duration,
) *Widget {
	s := components.NewSpinner(theme, "").WithSpinner(spinner.Line)
	s.Stop() // Don't show initially

	return &Widget{
		theme:    theme,
		now:      now,
		interval: interval,
		status:   status,
		spinner:  s,
	}
}

func (w *Widget) Init() tea.Cmd {
	return nil
}

func (w *Widget) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case spinner.TickMsg:
		return w, w.spinner.Update(m)

	case MsgRefreshRequested:
		w.status = RefreshStatusPending
		return w, nil

	case MsgRefreshComplete:
		w.status = RefreshStatusSuccess
		w.lastUpdatedAt = m.t
		w.spinner.Stop()

	case MsgRefreshFailed:
		w.status = RefreshStatusError
		w.spinner.Stop()
	}
	return w, nil
}

func (w *Widget) View() tea.View {
	interval := "Off"
	if w.interval > 0 {
		interval = "Every " + w.interval.String()
	}

	style := w.theme.SuccessStyle
	switch w.status {
	case RefreshStatusPending:
		style = w.theme.FocusedText
	case RefreshStatusError:
		style = w.theme.ErrorStyle
	}

	age := "—"
	if !w.lastUpdatedAt.IsZero() {
		age = formatDuration(max(0, w.now().Sub(w.lastUpdatedAt))) + " ago"
	}

	hotkey := w.theme.RenderHelp(core.Shortcuts{core.RefreshSettings()}, core.HelpKeysOnly)
	content := style.Render(refreshIcon) + " " + w.theme.RefreshTextStyle.Render(age) +
		w.theme.BlurredText.Render(" • ") + w.theme.RefreshTextStyle.Render(interval) + " " + hotkey
	return tea.NewView(content)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < hoursPerDay*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/hoursPerDay))
}

func countdown(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return formatDuration(((d + time.Second - 1) / time.Second) * time.Second)
}
