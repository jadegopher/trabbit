package tabs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func TestQueueChartsFitLayoutWithScrollbar(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	samples := []domain.Sample{
		{Timestamp: 70000, Value: 6000},
		{Timestamp: 40000, Value: 3000},
		{Timestamp: 10000, Value: 0},
	}
	stats := domain.QueueDetailedStats{
		MessagesTotal: samples,
		MessageStats:  domain.MessageStats{PublishDetails: domain.RateDetails{Samples: samples}},
		Policy:        strings.Repeat("long-policy-name", 20),
	}
	q.charts.SetFocused(true)

	// Resize the same populated view, including hiding and restoring the charts.
	for _, width := range []int{160, 101, 240, 99, 201, 100, 160} {
		q.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		q.charts.Update(widgets.MsgQueueDetailedStats{Stats: stats})
		_, layout := q.getLayout()
		if layout.GetWidth("queue_details") == 0 {
			continue
		}
		panelWidth := layout.GetWidth("queue_details")
		panelX, _ := layout.GetOffset("queue_details")
		panelHeight := layout.GetHeight("queue_details")
		for step := 0; step < 60; step++ {
			q.charts.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			panel := ansi.Strip(q.charts.View().Content)
			require.Equal(t, panelWidth, lipgloss.Width(panel))
			require.Equal(t, panelHeight, lipgloss.Height(panel))
			require.Contains(t, panel, "▐│")
			for row, line := range strings.Split(ansi.Strip(q.View().Content), "\n") {
				require.Equal(t, width, lipgloss.Width(line))
				if row == 0 || row >= panelHeight-1 {
					continue
				}
				left := ansi.Cut(line, 0, panelX)
				for _, r := range left {
					require.False(t, r >= '\u2800' && r <= '\u28ff', "chart escaped into table")
				}
				right := ansi.Cut(line, panelX, panelX+panelWidth)
				require.True(t, strings.HasPrefix(right, "│ "))
				require.True(t, strings.HasSuffix(right, "  │") || strings.HasSuffix(right, " ▐│"))
			}
		}
	}
}

func TestTabDoesNotChangeSelectionAndFilterIsInsideBorder(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	q.query.Name = "orders"
	q.table.SetFilter(q.filterLabel())
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{Name: "first"}, {Name: "second"}}})
	q.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	q.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	require.Equal(t, "first", q.table.ActiveQueue().Name)
	view := q.table.View().Content
	require.Less(t, strings.Index(view, "╭"), strings.Index(view, "Filter active"))
	require.Less(t, strings.Index(view, "VHost"), strings.Index(view, "Filter active"))
	lines := strings.Split(ansi.Strip(view), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Filter active") {
			require.Contains(t, line, "1 of 1 ⚬ Total: 0 ⚬ ◆ Filter active")
			require.Less(t, i+1, len(lines))
			require.Contains(t, lines[i+1], "g GoTo")
		}
	}
}

func TestFilterIndicatorAndStaleLoader(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	require.Empty(t, q.filterLabel())
	q.query.Name = "orders"
	q.query.VHost = "/"
	q.query.UseRegex = true
	require.Contains(t, q.filterLabel(), "◆ Filter active")
	require.NotContains(t, q.filterLabel(), "orders")
	require.NotContains(t, q.filterLabel(), "regex")
	q.listID = 2
	q.fetching = false
	_, cmd := q.Update(loadingDelay{id: 2})
	require.Nil(t, cmd, "completed request cannot show delayed loader")
	q.fetching = true
	_, cmd = q.Update(loadingDelay{id: 1})
	require.Nil(t, cmd, "superseded request cannot show loader")
	_, cmd = q.Update(loadingDelay{id: 2})
	require.NotNil(t, cmd, "slow current request shows loading")
}

func settle(q *QueueTab, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			settle(q, c)
		}
		return
	}
	if _, ok := msg.(spinner.TickMsg); ok {
		return
	}
	_, next := q.Update(msg)
	settle(q, next)
}

func TestListThenDetailsUsesCapturedQuery(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	query := app.ListParameters{Page: 1, PageSize: 3}
	backend.EXPECT().ListQueues(mock.Anything, query).Return(
		domain.PagedQueues{
			Items:      domain.Queues{{VHost: "/", Name: "orders"}},
			Pagination: domain.Pagination{Page: 1, PageSize: 3, PageCount: 1},
		}, nil,
	).Once()
	backend.EXPECT().GetQueueDetailedStats(
		mock.Anything,
		"/",
		"orders",
		time.Minute,
	).Return(domain.QueueDetailedStats{Messages: domain.Messages{Total: 42}}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 160, Height: 47})
	cmd := q.Refresh()
	q.query.Name = "changed after command creation"
	settle(q, cmd)
	require.Equal(t, "orders", q.table.ActiveQueue().Name)
	require.Contains(t, q.charts.View().Content, "42")
}

func TestDeleteUsesConfirmedIdentity(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().DeleteQueue(mock.Anything, "/", "confirmed").Return(nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	dialog := components.NewConfirm("delete", core.DefaultTheme())
	_, cmd := q.Update(core.DialogResult{Dialog: dialog, Value: ActionDeleteQueue{VHost: "/", Name: "confirmed"}})
	open := cmd().(core.OpenDialog)
	result := open.After().(operationResult)
	require.NoError(t, result.err)
	require.Same(t, open.Model, result.dialog)
}

func TestQueueResizeDoesNotFetch(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	_, cmd := q.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	require.Nil(t, cmd)
}
func TestStaleListsAndDetailsAreIgnored(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.listID = 3
	q.detailsID = 4
	q.Update(listResult{id: 2, data: domain.PagedQueues{Items: domain.Queues{{Name: "old", VHost: "/"}}}})
	require.Nil(t, q.table.ActiveQueue())
	_, cmd := q.Update(detailResult{id: 3, err: errors.New("old error")})
	require.Nil(t, cmd)
}
func TestFailedPaginationRestoresDisplayedPage(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.query.Page = 2
	q.query.PageSize = 20
	q.listID = 1
	q.Update(listResult{id: 1, err: errors.New("offline")})
	require.Equal(t, 1, q.query.Page)
	require.Equal(t, 3, q.query.PageSize)
}

func TestGoToChangesPageAndSizeTogether(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	query := app.ListParameters{Page: 3, PageSize: 5}
	backend.EXPECT().ListQueues(mock.Anything, query).Return(
		domain.PagedQueues{
			Items:      domain.Queues{{VHost: "/", Name: "orders"}},
			Pagination: domain.Pagination{Page: 3, PageSize: 5, PageCount: 4, FilteredCount: 20},
		}, nil,
	).Once()
	backend.EXPECT().GetQueueDetailedStats(
		mock.Anything,
		"/",
		"orders",
		time.Minute,
	).Return(domain.QueueDetailedStats{}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	_, cmd := q.Update(core.DialogResult{Value: components.ActionPageChange{Page: 3, PageSize: 5}})
	settle(q, cmd)
	require.Equal(t, query, q.query)
	require.Equal(t, 5, q.displayedPageSize)
	q.Update(components.ActionPageChange{Page: 4})
	require.Equal(t, 5, q.query.PageSize, "arrow navigation retains the selected size")
}

func TestQueuePageSizePersistsAcrossTabs(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), settings.FileName)
	preferences, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, preferences.Update(func(config *settings.Config) { config.Queues.Table.PageSize = 25 }))
	backend := mocks.NewMockQueueClient(t)
	query := app.ListParameters{Page: 2, PageSize: 20}
	backend.EXPECT().ListQueues(mock.Anything, query).Return(
		domain.PagedQueues{
			Items:      domain.Queues{{VHost: "/", Name: "orders"}},
			Pagination: domain.Pagination{Page: 2, PageSize: 20, PageCount: 3, FilteredCount: 60},
		}, nil,
	).Once()
	backend.EXPECT().GetQueueDetailedStats(
		mock.Anything,
		"/",
		"orders",
		time.Minute,
	).Return(domain.QueueDetailedStats{}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), preferences)
	require.Equal(t, 25, q.query.PageSize)
	_, cmd := q.Update(core.DialogResult{Value: components.ActionPageChange{Page: 2, PageSize: 20}})
	require.Equal(t, 25, preferences.Snapshot().Queues.Table.PageSize, "Update must defer disk work to its command")
	settle(q, cmd)
	require.Equal(t, query, q.query)

	reopened, err := settings.Open(path)
	require.NoError(t, err)
	recreated := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), reopened)
	require.Equal(t, 20, recreated.query.PageSize)
	require.Equal(t, 1, recreated.query.Page, "the destination page is not a saved preference")
}

func TestQueuePageSizeSaveFailureKeepsCurrentSettings(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.Mkdir(dir, 0700))
	preferences, err := settings.Open(filepath.Join(dir, settings.FileName))
	require.NoError(t, err)
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), preferences)
	require.NoError(t, os.Rename(dir, dir+"-moved"))
	_, cmd := q.Update(core.DialogResult{Value: components.ActionPageChange{Page: 2, PageSize: 20}})
	require.NotNil(t, cmd)
	_, next := q.Update(cmd())
	require.NotNil(t, next)
	dialog := next().(core.OpenDialog).Model
	require.IsType(t, &components.ErrorModal{}, dialog)
	require.Contains(t, dialog.View().Content, "save settings")
	require.Equal(t, 1, q.query.Page)
	require.Equal(t, settings.DefaultQueuePageSize, q.query.PageSize)
	require.Equal(t, settings.DefaultQueuePageSize, preferences.Snapshot().Queues.Table.PageSize)
	require.False(t, q.savingSettings)
}

func TestQueueSelectionDebouncesOverview(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().GetQueueDetailedStats(mock.Anything, "/", "third", time.Minute).Return(
		domain.QueueDetailedStats{},
		nil,
	).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	q.table.Update(
		widgets.MsgQueues{
			Queues: domain.Queues{
				{VHost: "/", Name: "first"}, {VHost: "/", Name: "second"}, {VHost: "/", Name: "third"},
			},
		},
	)
	q.fetchDetails()
	oldID := q.detailsID
	cancelled := false
	cancel := q.detailsCancel
	q.detailsCancel = func() {
		cancel()
		cancelled = true
	}

	q.table.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	_, secondDebounce := q.Update(widgets.MsgQueueSelectionChanged{VHost: "/", Name: "second"})
	require.True(t, cancelled, "selection cancels the previous request")
	require.False(t, q.detailsFetching)
	_, cmd := q.Update(detailResult{id: oldID, err: errors.New("superseded")})
	require.Nil(t, cmd)
	_, cmd = q.Update(loadingDelay{id: oldID, details: true})
	require.Nil(t, cmd)

	q.table.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, debounce := q.Update(widgets.MsgQueueSelectionChanged{VHost: "/", Name: "third"})
	require.Equal(t, "third", q.table.ActiveQueue().Name)
	_, cmd = q.Update(secondDebounce().(tea.BatchMsg)[0]())
	require.Nil(t, cmd, "an older timer cannot fetch the current selection")
	require.True(t, q.detailsPending)
	backend.AssertNotCalled(t, "GetQueueDetailedStats", mock.Anything, mock.Anything, mock.Anything)

	// Execute the actual timer command to cover the message routed by Bubble Tea.
	batch := debounce().(tea.BatchMsg)
	expired := batch[0]()
	_, cmd = q.Update(expired)
	require.False(t, q.detailsPending)
	settle(q, cmd)
	_, cmd = q.Update(expired)
	require.Nil(t, cmd, "a consumed timer cannot fetch twice")
}

func TestListRefreshRespectsPendingOverviewDebounce(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	queues := domain.Queues{{VHost: "/", Name: "orders"}}
	q.table.Update(widgets.MsgQueues{Queues: queues})
	_, debounce := q.Update(widgets.MsgQueueSelectionChanged{VHost: "/", Name: "orders"})
	q.Update(
		listResult{
			id: q.listID, data: domain.PagedQueues{
				Items: queues, Pagination: domain.Pagination{Page: 1, PageSize: 3, PageCount: 1},
			},
		},
	)
	require.True(t, q.detailsPending)
	require.False(t, q.detailsFetching, "polling must not start details while scrolling")
	_, cmd := q.Update(debounce().(tea.BatchMsg)[0]())
	require.Nil(t, cmd)
}

func TestHiddenQueueTabSuspendsOverviewDebounce(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().GetQueueDetailedStats(
		mock.Anything,
		"/",
		"orders",
		time.Minute,
	).Return(domain.QueueDetailedStats{}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	_, debounce := q.Update(widgets.MsgQueueSelectionChanged{VHost: "/", Name: "orders"})
	q.SetVisible(false)
	expired := debounce().(tea.BatchMsg)[0]()
	_, cmd := q.Update(expired)
	require.Nil(t, cmd, "hidden tabs cannot fetch a pending overview")
	require.True(t, q.detailsPending)

	resumed := q.SetVisible(true)
	_, cmd = q.Update(expired)
	require.Nil(t, cmd, "showing the tab must not revive the old timer")
	settle(q, resumed)
	require.False(t, q.detailsPending)
}

func TestEmptyQueueListCancelsPendingDetails(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	_, debounce := q.Update(widgets.MsgQueueSelectionChanged{VHost: "/", Name: "orders"})
	q.Update(
		listResult{
			id: q.listID, data: domain.PagedQueues{
				Pagination: domain.Pagination{Page: 1, PageSize: 3, PageCount: 1},
			},
		},
	)

	require.False(t, q.detailsPending)
	require.False(t, q.detailsFetching)
	_, cmd := q.Update(debounce().(tea.BatchMsg)[0]())
	require.Nil(t, cmd, "clearing the selection must invalidate its timer")
}

func TestQueueDetailsFailureReportsRefreshError(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "pepe", Name: "stustu"}}})
	q.detailsID = 4
	q.detailsFetching = true

	_, cmd := q.Update(detailResult{id: 4, vhost: "pepe", name: "stustu", err: errors.New("connection refused")})
	require.NotNil(t, cmd)
	failure := cmd().(RefreshFailed)
	require.EqualError(t, failure.Err, "connection refused")
	require.False(t, q.detailsFetching)
	require.NotContains(t, q.View().Content, "connection refused")

	_, cmd = q.Update(detailResult{id: 4, vhost: "other", name: "queue", err: errors.New("wrong selection")})
	require.Nil(t, cmd)
}

func TestQueueViewSettingsSaveAndRefreshHistory(t *testing.T) {
	t.Parallel()

	store, err := settings.Open(filepath.Join(t.TempDir(), settings.FileName))
	require.NoError(t, err)
	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().GetQueueDetailedStats(
		mock.Anything,
		"/",
		"orders",
		30*time.Minute,
	).Return(domain.QueueDetailedStats{}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), store)
	q.table.Update(widgets.MsgQueues{Queues: domain.Queues{{VHost: "/", Name: "orders"}}})
	_, open := q.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	dialog := open().(core.OpenDialog).Model
	require.IsType(t, &components.FormModal{}, dialog)
	chosen := settings.ChartSettings{Style: "lines", Size: "large", TimeRange: "30m"}
	_, save := q.Update(core.DialogResult{Dialog: dialog, Value: widgets.ActionChartSettings{Settings: chosen}})
	require.Equal(t, settings.DefaultChartSettings(), store.Snapshot().Queues.Charts)
	oldID := q.detailsID
	saved := save()
	_, refresh := q.Update(saved)
	_, duplicate := q.Update(saved)
	require.Nil(t, duplicate, "duplicate save results must not start another request")
	require.Equal(t, chosen, store.Snapshot().Queues.Charts)
	require.Greater(t, q.detailsID, oldID)
	for _, cmd := range refresh().(tea.BatchMsg) {
		if cmd == nil {
			continue
		}
		if batch, ok := cmd().(tea.BatchMsg); ok {
			for _, request := range batch {
				if request != nil {
					q.Update(request())
				}
			}
		}
	}
	_, reopen := q.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	modal := reopen().(core.OpenDialog).Model.(*components.FormModal)
	value, err := modal.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, widgets.ActionChartSettings{Settings: chosen}, value)
}

func TestQueueViewSettingsFailedSaveKeepsPreferences(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.Mkdir(dir, 0700))
	store, err := settings.Open(filepath.Join(dir, settings.FileName))
	require.NoError(t, err)
	require.NoError(t, os.Rename(dir, dir+"-moved"))
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), store)
	dialog := widgets.NewModalChartSettings(core.DefaultTheme(), store.Snapshot().Queues.Charts)
	chosen := settings.ChartSettings{Style: "lines", Size: "large", TimeRange: "1h"}
	_, save := q.Update(core.DialogResult{Dialog: dialog, Value: widgets.ActionChartSettings{Settings: chosen}})
	_, feedback := q.Update(save())
	result := feedback().(core.DialogFeedback)
	require.Same(t, dialog, result.Dialog)
	require.Error(t, result.Err)
	require.Equal(t, settings.DefaultChartSettings(), store.Snapshot().Queues.Charts)
	require.False(t, q.savingChartSettings)
}

func TestFailedDeleteIsModalOnlyAndNeverReplayed(t *testing.T) {
	t.Parallel()

	backend := mocks.NewMockQueueClient(t)
	backend.EXPECT().DeleteQueue(mock.Anything, "/", "orders").Return(errors.New("connection refused")).Once()
	backend.EXPECT().ListQueues(mock.Anything, mock.Anything).Return(domain.PagedQueues{}, nil).Once()
	q := NewQueueTab(core.DefaultTheme(), app.New(backend), context.Background(), settings.NewMemory())
	q.Update(tea.WindowSizeMsg{Width: core.MinWidth, Height: 24})
	confirm := components.NewConfirm("delete", core.DefaultTheme())
	_, cmd := q.Update(core.DialogResult{Dialog: confirm, Value: ActionDeleteQueue{VHost: "/", Name: "orders"}})
	open := cmd().(core.OpenDialog)
	result := open.After().(operationResult)
	_, cmd = q.Update(result)
	replace := cmd().(core.ReplaceDialog)
	require.Same(t, open.Model, replace.Old)
	require.IsType(t, &components.ErrorModal{}, replace.Model)
	require.Contains(t, replace.Model.View().Content, "connection")
	require.Contains(t, replace.Model.View().Content, "refused")
	require.NotContains(t, q.View().Content, "connection refused")
	require.Contains(t, ansi.Strip(q.View().Content), "Delete")
	require.False(t, q.fetching, "failed actions must not schedule another request")

	// Closing the error dialog and later successful polling must only read data.
	settle(q, q.SetVisible(false))
	settle(q, q.SetVisible(true))
	settle(q, q.Refresh())
	require.NotContains(t, q.View().Content, "connection refused")
}
