package tabs

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish"
	publishwidgets "github.com/jadegopher/trabbit/internal/tui/scenes/publish/widgets"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPublishOpensScreenWithoutChangingQueueSubtabs(t *testing.T) {
	t.Parallel()
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{Name: "orders", VHost: "/team"}}})
	_, cmd := q.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	require.NotNil(t, cmd, "publish opens even when the details panel is hidden")
	open := cmd().(core.OpenScreen)
	require.IsType(t, &publish.Scene{}, open.Model)
	open.Model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	open.Model.Init()
	t.Cleanup(open.Model.(*publish.Scene).Close)
	require.Contains(t, open.Model.View().Content, "orders")
	_, submit := open.Model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	request := submit().(publishwidgets.PublishRequest)
	require.Equal(t, "/team", request.Target.VHost)
	require.Equal(t, app.DefaultExchange, request.Target.Exchange)
	require.Equal(t, "orders", request.Options.RoutingKey)
	require.Equal(t, []string{"Overview", "Consumers", "Bindings", "Runtime metrics"}, q.subtabs.Labels)
	require.Equal(t, "orders", q.table.ActiveQueue().Name)
	for i, key := range []rune{'[', ']'} {
		q.Update(tea.KeyPressMsg{Code: key, Text: string(key)})
		require.Equal(t, []int{queueRuntime, queueOverview}[i], q.subtabs.Active)
	}
}
