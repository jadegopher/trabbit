package components

import (
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSpinnerStopsAndRejectsOldTicks(t *testing.T) {
	s := NewSpinner(core.DefaultTheme(), "Loading")
	start := s.Start()
	require.NotNil(t, start)
	old := start()
	require.Nil(t, s.Start(), "duplicate start must not create a second timer")
	require.NotNil(t, s.Update(old))
	s.Stop()
	require.Nil(t, s.Update(old))
	next := s.Start()
	require.NotNil(t, next)
	require.Nil(t, s.Update(old), "old generation must not restart after new start")
	require.NotNil(t, s.Update(next()))
}
func TestLoaderStops(t *testing.T) {
	l := NewLoader("Loading", core.DefaultTheme())
	start := l.Init()
	require.NotNil(t, start)
	msg := start()
	l.Stop()
	_, cmd := l.Update(msg)
	require.Nil(t, cmd)
}
