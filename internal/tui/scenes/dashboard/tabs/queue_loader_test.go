package tabs

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
)

func TestQueueSubtabsShareDelayedLoader(t *testing.T) {
	t.Parallel()

	for active, name := range queueSubtabIDs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
			q.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
			q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
			q.subtabs.Active = active
			q.scheduleDetails()
			require.NotContains(t, q.detailsView().Content, "Loading", "debouncing must not show a separate loading message")
			q.fetchDetails()
			require.NotContains(t, q.detailsView().Content, "Loading", "the shared delay owns loading presentation")
			if active == queueRuntime {
				require.NotContains(t, q.detailsView().Content, "unavailable", "metrics are unknown until the first result")
			}
			require.False(t, q.detailsLoader.IsActive(), "fast requests must not flash the spinner")

			_, start := q.Update(delayedLoading(q.detailsID, true)())
			require.NotNil(t, start)
			tick := start()
			_, next := q.Update(tick)
			require.NotNil(t, next, "loader ticks must continue even while the queue table has focus")

			panel := ansi.Strip(q.detailsView().Content)
			require.Contains(t, panel, "Loading details...")
			_, layout := q.getLayout()
			require.Equal(t, layout.GetWidth("queue_details"), lipgloss.Width(panel))
			require.Equal(t, layout.GetHeight("queue_details"), lipgloss.Height(panel))
			require.Contains(t, ansi.Strip(q.View().Content), "orders", "the queue table stays visible")

			if active == queueBindings {
				q.Update(bindingsResult{id: q.detailsID, vhost: "/", name: "orders"})
			} else {
				q.Update(detailResult{id: q.detailsID, vhost: "/", name: "orders"})
			}
			require.False(t, q.detailsLoader.IsActive())
			require.NotContains(t, q.detailsView().Content, "Loading details...")
			_, next = q.Update(tick)
			require.Nil(t, next, "completed requests must stop scheduling loader ticks")
			_, next = q.Update(loadingDelay{id: q.detailsID, details: true})
			require.Nil(t, next, "a late timer must not revive a completed loader")
		})
	}
}

func TestQueueSubtabErrorsStopSharedLoader(t *testing.T) {
	t.Parallel()

	for active, name := range queueSubtabIDs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
			q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
			q.subtabs.Active = active
			q.fetchDetails()
			q.Update(loadingDelay{id: q.detailsID, details: true})
			require.True(t, q.detailsLoader.IsActive())

			err := errors.New("unavailable")
			var result tea.Msg = detailResult{id: q.detailsID, vhost: "/", name: "orders", err: err}
			if active == queueBindings {
				result = bindingsResult{id: q.detailsID, vhost: "/", name: "orders", err: err}
			}
			_, report := q.Update(result)
			require.ErrorIs(t, report().(RefreshFailed).Err, err)
			require.False(t, q.detailsLoader.IsActive())
		})
	}
}

func TestQueueSharedLoaderRejectsSupersededTicks(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	q.fetchDetails()
	oldID := q.detailsID
	_, start := q.Update(loadingDelay{id: oldID, details: true})
	oldTick := start()

	q.Update(tea.KeyPressMsg{Code: ']', Text: core.KeyCloseBracket})
	require.False(t, q.detailsLoader.IsActive(), "switching subtabs stops the previous animation")
	_, cmd := q.Update(loadingDelay{id: oldID, details: true})
	require.Nil(t, cmd)
	_, cmd = q.Update(oldTick)
	require.Nil(t, cmd)

	q.fetchDetails()
	_, start = q.Update(loadingDelay{id: q.detailsID, details: true})
	require.NotNil(t, start)
	_, cmd = q.Update(oldTick)
	require.Nil(t, cmd, "a previous animation cannot tick a replacement loader")
	q.Update(detailResult{id: oldID, vhost: "/", name: "orders"})
	require.True(t, q.detailsLoader.IsActive(), "stale results cannot stop the current loader")

	q.SetVisible(false)
	require.False(t, q.detailsLoader.IsActive(), "hidden scenes stop animating")
	_, cmd = q.Update(start())
	require.Nil(t, cmd)
}
