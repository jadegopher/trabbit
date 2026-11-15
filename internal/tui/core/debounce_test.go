package core

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestDebouncerLatestTriggerWins(t *testing.T) {
	t.Parallel()

	var d Debouncer
	message := func(gen int) tea.Msg { return gen }
	first := d.Trigger(0, message)
	latest := d.Trigger(0, message)

	// Commands can finish out of order without changing which trigger is valid.
	latestGeneration := latest().(int)
	firstGeneration := first().(int)
	require.True(t, d.IsValid(latestGeneration))
	require.False(t, d.IsValid(firstGeneration))
}

func TestDebouncerCancelAndRestart(t *testing.T) {
	t.Parallel()

	d := NewDebouncer()
	message := func(gen int) tea.Msg { return gen }
	pending := d.Trigger(0, message)
	d.Cancel()
	cancelledGeneration := pending().(int)
	require.False(t, d.IsValid(cancelledGeneration))

	restartedGeneration := d.Trigger(0, message)().(int)
	require.True(t, d.IsValid(restartedGeneration))
	require.False(t, d.IsValid(cancelledGeneration))

	// Cancellation also rejects a timer message already waiting in the event loop.
	d.Cancel()
	require.False(t, d.IsValid(restartedGeneration))
}

func TestDebouncersKeepIndependentState(t *testing.T) {
	t.Parallel()

	first, second := NewDebouncer(), NewDebouncer()
	message := func(gen int) tea.Msg { return gen }
	firstGeneration := first.Trigger(0, message)().(int)
	secondGeneration := second.Trigger(0, message)().(int)

	first.Cancel()
	require.False(t, first.IsValid(firstGeneration))
	require.True(t, second.IsValid(secondGeneration))
}
