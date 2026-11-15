package refresher

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

type pendingIndicator struct{ generation int }
type Result struct{ Err error }
type statusTick struct{ generation int }
type Controller struct {
	pending    bool
	timer      *Refresher
	widget     *Widget
	visible    bool
	generation int
}

func NewController(theme core.Theme, now func() time.Time, interval time.Duration) *Controller {
	return &Controller{
		timer:   NewRefresher(interval, now),
		widget:  NewWidget(theme, RefreshStatusPending, now, interval),
		visible: true,
	}
}
func (c *Controller) Init() tea.Cmd                      { return c.timer.Tick() }
func (c *Controller) Accepts(m MsgRefreshRequested) bool { return c.visible && c.timer.IsValid(m) }
func (c *Controller) SetVisible(visible bool) tea.Cmd {
	if c.visible == visible {
		return nil
	}
	c.visible = visible
	c.pending = false
	c.generation++
	c.timer.generation++
	c.widget.spinner.Stop()
	if visible {
		now := c.timer.now()
		elapsed := now.Sub(c.widget.lastUpdatedAt)
		if !c.widget.nextRefreshAt.IsZero() {
			elapsed = c.timer.interval - c.widget.nextRefreshAt.Sub(now)
		}
		if c.widget.lastUpdatedAt.IsZero() || (c.timer.interval > 0 && elapsed >= c.timer.interval) {
			c.timer.firstTick = true
			c.widget.nextRefreshAt = time.Time{}
			return c.timer.Tick()
		}
		if c.timer.interval == 0 {
			return c.statusTick()
		}
		return tea.Batch(c.statusTick(), c.scheduleNext(c.timer.interval-elapsed))
	}
	return nil
}
func (c *Controller) SetInterval(interval time.Duration) tea.Cmd {
	c.timer.SetInterval(interval)
	c.widget.interval = interval
	c.widget.nextRefreshAt = time.Time{}
	if c.visible {
		return c.timer.Tick()
	}
	return nil
}
func (c *Controller) statusTick() tea.Cmd {
	if !c.visible {
		return nil
	}
	gen := c.generation
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return statusTick{gen} })
}
func (c *Controller) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case MsgRefreshRequested:
		if !c.Accepts(m) {
			return nil
		}
		c.timer.generation++
		c.generation++
		c.pending = true
		c.widget.nextRefreshAt = c.timer.now()
		gen := c.generation
		return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return pendingIndicator{gen} })
	case pendingIndicator:
		if c.pending && c.visible && m.generation == c.generation {
			_, cmd := c.widget.Update(MsgRefreshRequested{})
			return cmd
		}
		return nil
	case Result:
		c.pending = false
		c.timer.generation++
		c.timer.firstTick = false
		c.generation++
		c.widget.spinner.Stop()
		if m.Err == nil {
			c.widget.lastUpdatedAt = c.timer.now()
			c.widget.status = RefreshStatusSuccess
		} else {
			c.widget.status = RefreshStatusError
		}
		if c.visible {
			return tea.Batch(c.scheduleNext(c.timer.interval), c.statusTick())
		}
		return nil
	case statusTick:
		if m.generation != c.generation {
			return nil
		}
		return c.statusTick()
	default:
		if c.visible {
			_, cmd := c.widget.Update(msg)
			return cmd
		}
	}
	return nil
}
func (c *Controller) View() tea.View          { return c.widget.View() }
func (c *Controller) Interval() time.Duration { return c.timer.Interval() }

// ResetDeadline postpones polling after user navigation without changing last-success status.
func (c *Controller) ResetDeadline() tea.Cmd {
	c.timer.generation++
	c.timer.firstTick = false
	if !c.visible {
		return nil
	}
	return c.scheduleNext(c.timer.interval)
}

func (c *Controller) scheduleNext(delay time.Duration) tea.Cmd {
	if c.timer.interval <= 0 {
		c.widget.nextRefreshAt = time.Time{}
		return nil
	}
	deadline := c.timer.now().Add(max(0, delay))
	c.widget.nextRefreshAt = deadline
	gen := c.timer.generation
	return tea.Tick(max(0, delay), func(t time.Time) tea.Msg { return MsgRefreshRequested{t: t, generation: gen} })
}
