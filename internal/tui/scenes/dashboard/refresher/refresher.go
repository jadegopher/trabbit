package refresher

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Refresher encapsulates the logic for periodic data refreshing using tea.Tick.
// It uses a generation counter to ensure that only messages from the most recent
// timer are processed, effectively "cancelling" older timers that are still running.
type Refresher struct {
	interval   time.Duration
	generation int
	now        func() time.Time
	// firstTick tracks whether the next Tick() call is the first one for the current interval.
	// This allows an immediate initial refresh when the interval is set (even for >0 intervals),
	// preventing a long delay before the first update.
	firstTick bool
}

// NewRefresher creates a new Refresher with the given interval.
func NewRefresher(interval time.Duration, now func() time.Time) *Refresher {
	return &Refresher{
		interval:  interval,
		now:       now,
		firstTick: true,
	}
}

// Tick returns a tea.Cmd that will send a MsgRefreshRequested after the current interval.
func (r *Refresher) Tick() tea.Cmd {
	if r.interval == 0 && !r.firstTick {
		return nil
	}

	gen := r.generation

	if r.firstTick {
		r.firstTick = false
		return func() tea.Msg {
			return MsgRefreshRequested{
				t:          r.now(),
				generation: gen,
			}
		}
	}

	return tea.Tick(
		r.interval, func(t time.Time) tea.Msg {
			return MsgRefreshRequested{
				t:          t,
				generation: gen,
			}
		},
	)
}

// SetInterval updates the refresh interval and increments the generation.
// This effectively invalidates any pending ticks from the previous interval.
func (r *Refresher) SetInterval(interval time.Duration) {
	r.interval = interval
	r.generation++
	r.firstTick = true
}

func (r *Refresher) IsValid(msg MsgRefreshRequested) bool {
	return msg.generation == r.generation
}

// RefreshNow returns a MsgRefreshRequested immediately.
func (r *Refresher) RefreshNow() tea.Msg {
	r.generation++
	return MsgRefreshRequested{
		t:          r.now(),
		generation: r.generation,
	}
}

// Interval returns the current refresh interval.
func (r *Refresher) Interval() time.Duration {
	return r.interval
}
