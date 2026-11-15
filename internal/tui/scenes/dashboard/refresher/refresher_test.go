package refresher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRefresher(t *testing.T) {
	t.Parallel()

	now := time.Now

	t.Run(
		"initial_state", func(t *testing.T) {
			t.Parallel()
			interval := 5 * time.Second
			r := NewRefresher(interval, now)

			assert.Equal(t, interval, r.Interval())
			assert.Equal(t, 0, r.generation)
		},
	)

	t.Run(
		"set_interval_updates_and_increments_generation", func(t *testing.T) {
			t.Parallel()
			r := NewRefresher(5*time.Second, now)

			newInterval := 10 * time.Second
			r.SetInterval(newInterval)

			assert.Equal(t, newInterval, r.Interval())
			assert.Equal(t, 1, r.generation)
		},
	)

	t.Run(
		"tick_emits_once_for_zero_interval", func(t *testing.T) {
			t.Parallel()
			r := NewRefresher(0, now)

			// First tick should emit
			cmd := r.Tick()
			assert.NotNil(t, cmd)
			assert.Equal(t, 0, r.generation)
			assert.False(t, r.firstTick)

			// Second tick should be nil
			cmd = r.Tick()
			assert.Nil(t, cmd)
			assert.Equal(t, 0, r.generation)
		},
	)

	t.Run(
		"tick_does_not_increment_generation", func(t *testing.T) {
			t.Parallel()
			r := NewRefresher(5*time.Second, now)

			cmd := r.Tick()
			assert.NotNil(t, cmd)
			assert.Equal(t, 0, r.generation)

			_ = r.Tick()
			assert.Equal(t, 0, r.generation)
		},
	)

	t.Run(
		"is_valid_checks_generation", func(t *testing.T) {
			t.Parallel()
			r := NewRefresher(5*time.Second, now)

			// Initial generation is 0
			assert.True(t, r.IsValid(MsgRefreshRequested{generation: 0}))
			assert.False(t, r.IsValid(MsgRefreshRequested{generation: 1}))

			// RefreshNow increments generation
			r.RefreshNow() // gen = 1
			assert.True(t, r.IsValid(MsgRefreshRequested{generation: 1}))
			assert.False(t, r.IsValid(MsgRefreshRequested{generation: 0}))

			// Increment generation via SetInterval
			r.SetInterval(10 * time.Second) // gen = 2
			assert.True(t, r.IsValid(MsgRefreshRequested{generation: 2}))
			assert.False(t, r.IsValid(MsgRefreshRequested{generation: 1}))
		},
	)

	t.Run(
		"tick_command_execution", func(t *testing.T) {
			t.Parallel()
			// Use a very small interval to test the command execution
			interval := 1 * time.Millisecond
			r := NewRefresher(interval, now)
			cmd := r.Tick()

			msg := cmd()
			refreshMsg, ok := msg.(MsgRefreshRequested)

			assert.True(t, ok)
			assert.Equal(t, 0, refreshMsg.generation)
			assert.False(t, refreshMsg.t.IsZero())
		},
	)

}
