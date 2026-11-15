package refresher

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// MsgRefreshRequested is sent when it's time to refresh data showed (i.e. go and pull new data)
type MsgRefreshRequested struct {
	t time.Time
	// generation is used to track the "version" of the timer.
	// Since tea.Tick cannot be cancelled, we use this to ignore messages from old timers.
	generation int
}

func (m MsgRefreshRequested) Generation() int {
	return m.generation
}

// MsgRefreshComplete is sent when all i/o request completed successfully
type MsgRefreshComplete struct {
	t          time.Time
	generation int
	Msgs       []tea.Msg
}

// MsgRefreshFailed is sent when one or more i/o requests fail
type MsgRefreshFailed struct {
	t          time.Time
	generation int
	Err        error
}
