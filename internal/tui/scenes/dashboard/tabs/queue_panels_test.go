package tabs

import (
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/mocks"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
)

func TestBracketsFetchActiveSubtabAndPreserveQueueSelection(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().GetQueueDetailedStats(mock.Anything, "/", "orders", time.Minute).Return(domain.QueueDetailedStats{}, nil).Times(3)
	backend.On("ListQueueBindings", mock.Anything, "/", "orders").Return([]domain.QueueBinding{{VHost: "/", Queue: "orders"}}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 160, Height: 30})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}, {VHost: "/", Name: "other"}}})
	for _, expected := range []int{queueConsumers, queueBindings, queueRuntime, queueOverview} {
		_, cmd := q.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
		require.Equal(t, expected, q.subtabs.Active)
		require.Equal(t, focusDetails, q.focus)
		settle(q, cmd)
		require.Equal(t, "orders", q.table.ActiveQueue().Name)
	}
	q.Update(tea.KeyPressMsg{Code: '[', Text: "["})
	require.Equal(t, queueRuntime, q.subtabs.Active)
	require.Equal(t, focusDetails, q.focus)
	q.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, "orders", q.table.ActiveQueue().Name)
	q.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	require.Equal(t, focusTable, q.focus)
}

func TestSubtabSwitchCancelsOldRequestAndRejectsItsResult(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	q.fetchDetails()
	oldID := q.detailsID
	cancelled := false
	cancel := q.detailsCancel
	q.detailsCancel = func() { cancel(); cancelled = true }
	q.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	require.True(t, cancelled)
	_, cmd := q.Update(detailResult{id: oldID, vhost: "/", name: "orders", err: errors.New("old overview error")})
	require.Nil(t, cmd)
	q.SetVisible(false)
	_, cmd = q.Update(loadingDelay{id: q.detailsID, details: true})
	require.Nil(t, cmd)
	require.False(t, q.detailsFetching)
}

func TestConsumerPaginationCannotChangeQueuePagination(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	q.scheduleDetails()
	consumers := make([]domain.QueueConsumer, 120)
	for i := range consumers {
		consumers[i].Tag = fmt.Sprintf("consumer-%03d", i)
	}
	q.consumers.SetData(consumers)
	query := q.query
	_, cmd := q.Update(core.DialogResult{Value: components.ActionPageChange{Owner: q.consumers.PageOwner(), Page: 2}})
	require.Nil(t, cmd)
	require.Equal(t, query, q.query)
	require.Contains(t, q.consumers.View().Content, "consumer-100")
}

func TestUnbindConfirmsCapturedBindingAndNeverRunsTwice(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	binding := domain.QueueBinding{VHost: "/", Queue: "orders", Source: "events", RoutingKey: "order.*", PropertiesKey: "returned-key"}
	backend.On("DeleteQueueBinding", mock.Anything, binding).Return(nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}, {VHost: "/", Name: "other"}}})
	q.subtabs.Active, q.focus = queueBindings, focusDetails
	q.queueBindings.SetLoading("/", "orders")
	q.queueBindings.SetData([]domain.QueueBinding{{VHost: "/", Queue: "orders"}, binding})
	_, cmd := q.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	confirm := cmd().(core.OpenDialog).Model
	require.Contains(t, confirm.View().Content, "order.*")
	_, yes := confirm.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	confirmed := yes().(core.DialogResult)
	q.table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, operation := q.Update(confirmed)
	_, duplicate := q.Update(confirmed)
	require.Nil(t, duplicate)
	open := operation().(core.OpenDialog)
	result := open.After().(bindingOperationResult)
	_, finish := q.Update(result)
	require.IsType(t, core.DialogResult{}, finish())
	_, duplicate = q.Update(result)
	require.Nil(t, duplicate)
	require.False(t, q.detailsFetching, "completion cannot fetch bindings for a different selected queue")
}

func TestBindingFailurePreservesFormAndRequiresExplicitRetry(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	params := app.CreateBindingParams{VHost: "/", Queue: "orders", Source: "events"}
	backend.On("CreateQueueBinding", mock.Anything, params).Return(errors.New("forbidden")).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	dialog := widgets.NewModalAddBinding(core.DefaultTheme(), "/", "orders")
	_, cmd := q.Update(core.DialogResult{Dialog: dialog, Value: widgets.ActionCreateBinding{Params: params}})
	_, feedback := q.Update(cmd())
	message := feedback().(core.DialogFeedback)
	require.Same(t, dialog, message.Dialog)
	require.EqualError(t, message.Err, "bind queue //orders: forbidden")
	require.False(t, q.bindingBusy)
	require.False(t, q.detailsFetching)
}

func TestAddBindingAvailableAcrossSubtabsAndFocus(t *testing.T) {
	t.Parallel()
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	for _, active := range []int{queueOverview, queueConsumers, queueBindings, queueRuntime} {
		for _, focus := range []queueFocus{focusTable, focusDetails} {
			q.subtabs.Active, q.focus = active, focus
			_, cmd := q.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
			require.NotNil(t, cmd)
			require.IsType(t, core.OpenDialog{}, cmd())
			help := ansi.Strip(q.theme.RenderHelp(q.helpBindings()))
			require.Contains(t, help, "Add binding")
			require.NotContains(t, help, "Switch Focus")
			require.NotContains(t, help, "Scroll")
			require.NotContains(t, help, "Columns")
		}
	}
}

func TestBindingsPaginationRoutesIndependently(t *testing.T) {
	t.Parallel()
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 180, Height: 30})
	q.queueBindings.SetLoading("/", "orders")
	data := make([]domain.QueueBinding, 120)
	for i := range data {
		data[i] = domain.QueueBinding{Source: fmt.Sprintf("exchange-%03d", i), PropertiesKey: fmt.Sprint(i)}
	}
	q.queueBindings.SetData(data)
	q.consumers.SetData(make([]domain.QueueConsumer, 120))
	query := q.query
	_, cmd := q.Update(core.DialogResult{Value: components.ActionPageChange{Owner: q.queueBindings.PageOwner(), Page: 2}})
	require.Nil(t, cmd)
	require.Equal(t, query, q.query)
	require.Equal(t, "exchange-100", q.queueBindings.ActiveBinding().Source)
	require.Contains(t, ansi.Strip(q.consumers.View().Content), "1 of 2")
}

func TestSelectedSubtabSavesLatestChoiceAndRestores(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), settings.FileName)
	preferences, err := settings.Open(path)
	require.NoError(t, err)
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), preferences)
	_, first := q.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	require.Equal(t, queueConsumers, q.subtabs.Active)
	// Change again while the first save has not run. Only its completion starts
	// the next save, so execution order cannot restore the older choice.
	_, second := q.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	require.Nil(t, second)
	require.Equal(t, queueBindings, q.subtabs.Active)
	settle(q, first)
	reopened, err := settings.Open(path)
	require.NoError(t, err)
	require.Equal(t, settings.QueueSubtabBindings, reopened.Snapshot().Queues.SelectedSubtab)
	restored := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), reopened)
	require.Equal(t, queueBindings, restored.subtabs.Active)
}

func TestSelectedSubtabSaveFailurePreservesNavigation(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "settings")
	require.NoError(t, os.Mkdir(dir, 0700))
	preferences, err := settings.Open(filepath.Join(dir, settings.FileName))
	require.NoError(t, err)
	require.NoError(t, os.Rename(dir, dir+"-moved"))
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), preferences)
	_, save := q.Update(tea.KeyPressMsg{Code: ']', Text: "]"})
	result := save().(subtabSettingsSaved)
	require.Error(t, result.err)
	_, feedback := q.Update(result)
	require.IsType(t, core.OpenDialog{}, feedback())
	require.Equal(t, queueConsumers, q.subtabs.Active)
	require.False(t, q.savingSubtab)
	_, duplicate := q.Update(result)
	require.Nil(t, duplicate)
}
