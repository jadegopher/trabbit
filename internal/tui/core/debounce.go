package core

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Debouncer delays work until triggers settle. Its zero value is ready to use.
// Call Trigger, Cancel and IsValid from Update; commands only emit messages.
type Debouncer struct {
	gen int
}

// NewDebouncer creates a new Debouncer.
func NewDebouncer() *Debouncer {
	return &Debouncer{}
}

// Trigger supersedes earlier timers. The callback must capture its inputs,
// and Update must check IsValid before acting on the returned generation.
func (d *Debouncer) Trigger(delay time.Duration, callback func(gen int) tea.Msg) tea.Cmd {
	d.gen++
	g := d.gen
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return callback(g)
	})
}

// Cancel invalidates pending messages. Existing tea.Tick commands still finish.
func (d *Debouncer) Cancel() {
	d.gen++
}

// IsValid checks if the provided generation matches the current generation.
func (d *Debouncer) IsValid(gen int) bool {
	return d.gen == gen
}
