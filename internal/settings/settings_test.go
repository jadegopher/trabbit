package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenCreatesDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, Defaults(), store.Snapshot())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"overview": {}, "connections": {}, "channels": {}, "exchanges": {}, "admin": {},
		"queues": {"selected_subtab":"overview", "table": {"page_size": 3}, "overview": {}, "charts": {"style":"dotted", "size":"medium", "time_range":"1m"}, "consumers":{"page_size":100}, "bindings":{"page_size":100}}
	}`, string(data))
}

func TestRoundTripPreservesTabSections(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte(`{
		"admin": {"custom": {"enabled": true}},
		"queues": {"table": {"page_size": 25}, "overview": {"chart": "messages"}}
	}`), 0600))
	store, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, 25, store.Snapshot().Queues.Table.PageSize)
	require.NotNil(t, store.Snapshot().Connections)
	require.NoError(t, store.Update(func(config *Config) { config.Queues.Table.PageSize = 50 }))

	reopened, err := Open(path)
	require.NoError(t, err)
	config := reopened.Snapshot()
	require.Equal(t, 50, config.Queues.Table.PageSize)
	require.JSONEq(t, `{"enabled": true}`, string(config.Admin["custom"]))
	require.JSONEq(t, `"messages"`, string(config.Queues.Overview["chart"]))
}

func TestOpenRejectsInvalidSettings(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		`{`, `{"queues":{"table":{"page_size":"25"}}}`,
		`{"queues":{"table":{"page_size":0}}}`,
		`{"queues":{"table":{"page_size":-1}}}`,
		`{"queues":{"table":{"page_size":1001}}}`,
	} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), FileName)
			require.NoError(t, os.WriteFile(path, []byte(value), 0600))
			_, err := Open(path)
			require.Error(t, err)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, value, string(data), "invalid files must not be overwritten")
		})
	}
}

func TestInvalidUpdateKeepsSavedSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Error(t, store.Update(func(config *Config) { config.Queues.Table.PageSize = 0 }))
	require.Equal(t, DefaultQueuePageSize, store.Snapshot().Queues.Table.PageSize)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestFailedWriteKeepsSharedSettings(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.Mkdir(dir, 0700))
	store, err := Open(filepath.Join(dir, FileName))
	require.NoError(t, err)
	require.NoError(t, os.Rename(dir, dir+"-moved"))
	err = store.Update(func(config *Config) { config.Queues.Table.PageSize = 20 })
	require.ErrorContains(t, err, "save settings")
	require.Equal(t, DefaultQueuePageSize, store.Snapshot().Queues.Table.PageSize)
}

func TestSnapshotsAndEditsDoNotExposeSharedMaps(t *testing.T) {
	t.Parallel()

	store := NewMemory()
	require.NoError(t, store.Update(func(config *Config) { config.Admin["enabled"] = json.RawMessage(`true`) }))
	snapshot := store.Snapshot()
	snapshot.Admin["enabled"][0] = 'f'
	snapshot.Queues.Table.PageSize = 99
	require.Equal(t, "true", string(store.Snapshot().Admin["enabled"]))
	require.Equal(t, DefaultQueuePageSize, store.Snapshot().Queues.Table.PageSize)
}

func TestConcurrentTabUpdatesPreserveBothChanges(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	store, err := Open(path)
	require.NoError(t, err)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, edit := range []func(*Config){
		func(config *Config) { config.Queues.Table.PageSize = 100 },
		func(config *Config) { config.Admin["enabled"] = json.RawMessage(`true`) },
	} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- store.Update(edit)
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	reopened, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, 100, reopened.Snapshot().Queues.Table.PageSize)
	require.Equal(t, "true", string(reopened.Snapshot().Admin["enabled"]))
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".settings-*"))
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestChartSettingsPersistAndDefaultMissingFields(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte(`{"queues":{"charts":{"size":"large"}}}`), 0600))
	store, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, ChartSettings{Style: "dotted", Size: "large", TimeRange: "1m"}, store.Snapshot().Queues.Charts)
	chosen := ChartSettings{Style: "lines", Size: "small", TimeRange: "30m"}
	require.NoError(t, store.Update(func(c *Config) { c.Queues.Charts = chosen }))
	reopened, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, chosen, reopened.Snapshot().Queues.Charts)

	for _, invalid := range []ChartSettings{
		{Style: "custom", Size: "small", TimeRange: "1m"},
		{Style: "lines", Size: "huge", TimeRange: "1m"},
		{Style: "lines", Size: "small", TimeRange: "2h"},
	} {
		require.Error(t, store.Update(func(c *Config) { c.Queues.Charts = invalid }))
		require.Equal(t, chosen, store.Snapshot().Queues.Charts)
	}
}

func TestSelectedSubtabDefaultsAndValidation(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte(`{"queues":{"table":{"page_size":25}}}`), 0600))
	store, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, QueueSubtabOverview, store.Snapshot().Queues.SelectedSubtab)
	for _, selected := range []string{QueueSubtabConsumers, QueueSubtabBindings, QueueSubtabRuntime, QueueSubtabOverview} {
		require.NoError(t, store.Update(func(config *Config) { config.Queues.SelectedSubtab = selected }))
		reopened, err := Open(path)
		require.NoError(t, err)
		require.Equal(t, selected, reopened.Snapshot().Queues.SelectedSubtab)
	}
	require.Error(t, store.Update(func(config *Config) { config.Queues.SelectedSubtab = "unknown" }))
	require.Equal(t, QueueSubtabOverview, store.Snapshot().Queues.SelectedSubtab)
}
