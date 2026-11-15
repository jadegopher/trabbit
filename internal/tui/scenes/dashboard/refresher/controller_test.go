package refresher

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestAgeRemainsVisibleWhileDeadlineResets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	c := NewController(core.DefaultTheme(), func() time.Time { return now }, 5*time.Second)
	c.Update(Result{})
	last := c.widget.lastUpdatedAt
	require.Equal(t, now.Add(5*time.Second), c.widget.nextRefreshAt)
	now = now.Add(2 * time.Second)
	c.ResetDeadline()
	require.Equal(t, last, c.widget.lastUpdatedAt)
	require.Equal(t, now.Add(5*time.Second), c.widget.nextRefreshAt)
	now = now.Add(3 * time.Second)
	require.Contains(t, ansi.Strip(c.View().Content), "↻ 5s ago")
	require.NotContains(t, c.View().Content, "Refresh in")
	c.SetVisible(false)
	now = now.Add(time.Second)
	c.SetVisible(true)
	require.Contains(t, ansi.Strip(c.View().Content), "↻ 6s ago")
	c.SetInterval(0)
	require.True(t, c.widget.nextRefreshAt.IsZero())
	require.Contains(t, c.View().Content, "Off")
}

func TestCountdownNeverShowsNegativeTime(t *testing.T) {
	t.Parallel()

	require.Equal(t, "0s", countdown(-2*time.Second))
	require.Equal(t, "5s", countdown(4500*time.Millisecond))
	require.Equal(t, "1s", countdown(time.Millisecond))
}

func TestLoadingKeepsAgeAndSchedulesNoAnimation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	c := NewController(core.DefaultTheme(), func() time.Time { return now }, 5*time.Second)
	c.Update(Result{})
	last := c.widget.lastUpdatedAt
	now = now.Add(5 * time.Second)
	c.Update(MsgRefreshRequested{generation: c.timer.generation})
	cmd := c.Update(pendingIndicator{generation: c.generation})
	require.Nil(t, cmd)
	require.Contains(t, ansi.Strip(c.View().Content), "↻ 5s ago")
	require.Contains(t, c.View().Content, refreshIcon)
	require.False(t, c.widget.spinner.IsActive())
	c.Update(Result{Err: errors.New("offline")})
	require.Equal(t, last, c.widget.lastUpdatedAt)
}

func TestSelectionResetsDeadlineWithoutChangingSuccessTime(t *testing.T) {
	t.Parallel()

	c := NewController(core.DefaultTheme(), time.Now, time.Second)
	old := c.Init()().(MsgRefreshRequested)
	last := c.widget.lastUpdatedAt
	c.ResetDeadline()
	require.False(t, c.Accepts(old))
	require.Equal(t, last, c.widget.lastUpdatedAt)
}

func TestPageCompletionInvalidatesOldPollingDeadline(t *testing.T) {
	t.Parallel()

	c := NewController(core.DefaultTheme(), time.Now, time.Second)
	old := c.Init()().(MsgRefreshRequested)
	c.Update(Result{})
	require.False(t, c.Accepts(old))
	current := MsgRefreshRequested{generation: c.timer.generation}
	delay := c.Update(current)
	require.NotNil(t, delay)
	require.False(t, c.Accepts(current), "a deadline is consumed only once")
	c.Update(Result{})
	require.Nil(t, c.Update(pendingIndicator{generation: c.generation - 1}), "fast completion cancels delayed indicator")
	require.False(t, c.widget.spinner.IsActive())
}

func TestControllerRejectsTicksAfterPauseAndIntervalChange(t *testing.T) {
	t.Parallel()

	c := NewController(core.DefaultTheme(), time.Now, time.Second)
	tick := c.Init()().(MsgRefreshRequested)
	require.True(t, c.Accepts(tick))
	c.SetVisible(false)
	require.False(t, c.Accepts(tick))
	require.Nil(t, c.Update(tick))
	c.SetVisible(true)
	require.False(t, c.Accepts(tick))
	c.SetInterval(0)
	require.False(t, c.Accepts(tick))
	c.Update(Result{Err: errors.New("offline")})
	require.Equal(t, RefreshStatusError, c.widget.status)
	c.Update(Result{})
	require.Equal(t, RefreshStatusSuccess, c.widget.status)
}
