package tabs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func openDetailPagination(t *testing.T, q *QueueTab, subtab int) (*components.FormModal, components.ActionPageChange) {
	t.Helper()
	q.subtabs.Active = subtab
	_, open := q.Update(tea.KeyPressMsg{Code: 'v', Text: core.KeyV})
	require.NotNil(t, open)
	modal := open().(core.OpenDialog).Model.(*components.FormModal)
	require.False(t, modal.CloseOnSubmit)
	value, err := modal.OnSubmit()
	require.NoError(t, err)
	return modal, value.(components.ActionPageChange)
}

func TestDetailPaginationSaveAndRestoreAreIndependent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), settings.FileName)
	store, err := settings.Open(path)
	require.NoError(t, err)
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), store)
	q.consumers.SetLoading("/", "orders")
	q.queueBindings.SetLoading("/", "orders")
	q.consumers.SetData(make([]domain.QueueConsumer, 120))
	q.queueBindings.SetData(make([]domain.QueueBinding, 120))
	query := q.query
	for _, tc := range []struct{ subtab, size int }{{queueConsumers, 5}, {queueBindings, 20}} {
		modal, change := openDetailPagination(t, q, tc.subtab)
		require.Equal(t, settings.DefaultDetailPageSize, change.PageSize)
		change.PageSize, change.Page = tc.size, 2
		_, save := q.Update(core.DialogResult{Dialog: modal, Value: change})
		require.NotNil(t, save)
		_, duplicate := q.Update(core.DialogResult{Dialog: modal, Value: change})
		require.Nil(t, duplicate)
		_, before := openDetailPagination(t, q, tc.subtab)
		require.Equal(t, settings.DefaultDetailPageSize, before.PageSize, "draft is not applied before its command")
		result := save()
		_, close := q.Update(result)
		require.Equal(t, core.DialogResult{Dialog: modal, Close: true}, close())
		_, duplicate = q.Update(result)
		require.Nil(t, duplicate)
		_, after := openDetailPagination(t, q, tc.subtab)
		require.Equal(t, tc.size, after.PageSize)
		require.Equal(t, 2, after.Page)
		require.Equal(t, query, q.query, "detail settings cannot change queue pagination")
	}
	reopened, err := settings.Open(path)
	require.NoError(t, err)
	recreated := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), reopened)
	_, consumers := openDetailPagination(t, recreated, queueConsumers)
	_, bindings := openDetailPagination(t, recreated, queueBindings)
	require.Equal(t, 5, consumers.PageSize)
	require.Equal(t, 20, bindings.PageSize)
	require.NotEmpty(t, consumers.Owner, "empty lists still need their own pager identity")
	require.NotEqual(t, consumers.Owner, bindings.Owner)
}

func TestDetailPaginationFailedSaveKeepsFormAndPreferences(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.Mkdir(dir, 0700))
	store, err := settings.Open(filepath.Join(dir, settings.FileName))
	require.NoError(t, err)
	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), store)
	require.NoError(t, os.Rename(dir, dir+"-moved"))
	for _, subtab := range []int{queueConsumers, queueBindings} {
		modal, change := openDetailPagination(t, q, subtab)
		change.PageSize = 5
		_, save := q.Update(core.DialogResult{Dialog: modal, Value: change})
		failed := save()
		_, feedback := q.Update(failed)
		result := feedback().(core.DialogFeedback)
		require.Same(t, modal, result.Dialog)
		require.Error(t, result.Err)
		_, after := openDetailPagination(t, q, subtab)
		require.Equal(t, settings.DefaultDetailPageSize, after.PageSize)
		_, retry := q.Update(core.DialogResult{Dialog: modal, Value: change})
		require.NotNil(t, retry)
		_, duplicate := q.Update(failed)
		require.Nil(t, duplicate, "an old failure must not consume the retry")
		q.Update(retry())
	}
	require.Equal(t, settings.Defaults(), store.Snapshot())
}

func TestDetailPaginationDelayedSaveCannotNavigateAnotherQueue(t *testing.T) {
	t.Parallel()

	q := NewQueueTab(core.DefaultTheme(), app.New(nil), context.Background(), settings.NewMemory())
	for _, subtab := range []int{queueConsumers, queueBindings} {
		q.consumers.SetLoading("/", "old")
		q.queueBindings.SetLoading("/", "old")
		modal, change := openDetailPagination(t, q, subtab)
		change.PageSize, change.Page = 5, 3
		_, save := q.Update(core.DialogResult{Dialog: modal, Value: change})
		q.consumers.SetLoading("/", "new")
		q.queueBindings.SetLoading("/", "new")
		q.consumers.SetData(make([]domain.QueueConsumer, 120))
		q.queueBindings.SetData(make([]domain.QueueBinding, 120))
		q.Update(save())
		_, current := openDetailPagination(t, q, subtab)
		require.Equal(t, 5, current.PageSize, "saved size applies across queues")
		require.Equal(t, 1, current.Page, "old destination does not apply to the new queue")
		_, stale := q.Update(core.DialogResult{Dialog: modal, Value: change})
		require.Equal(t, core.DialogResult{Dialog: modal, Close: true}, stale())
	}
}
