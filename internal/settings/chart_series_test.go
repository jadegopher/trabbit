package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChartSeriesOverridesPersistAndKeepLegacyDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte(`{"queues":{"charts":{"style":"lines"}}}`), 0600))
	store, err := Open(path)
	require.NoError(t, err)
	charts := store.Snapshot().Queues.Charts
	for _, name := range MessageSeries() {
		require.True(t, charts.MessageVisible(name))
	}
	for _, name := range ThroughputSeries() {
		require.Equal(t, name == "Publish" || name == "DeliverGet" || name == "Ack", charts.ThroughputVisible(name))
	}
	charts.Messages = map[string]bool{"Ready": false, "Total": false}
	charts.Throughput = map[string]bool{"Publish": false, "Redeliver": true, "GetNoAck": true}
	require.NoError(t, store.Update(func(config *Config) { config.Queues.Charts = charts }))
	reopened, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, charts, reopened.Snapshot().Queues.Charts)

	// Neither the edit source nor a snapshot can mutate the shared overrides.
	charts.Messages["Ready"] = true
	snapshot := store.Snapshot()
	snapshot.Queues.Charts.Throughput["Publish"] = true
	require.False(t, store.Snapshot().Queues.Charts.MessageVisible("Ready"))
	require.False(t, store.Snapshot().Queues.Charts.ThroughputVisible("Publish"))
	require.Error(t, store.Update(func(config *Config) { config.Queues.Charts.Messages["unknown"] = true }))
	require.Error(t, store.Update(func(config *Config) { config.Queues.Charts.Throughput["unknown"] = true }))
	require.Equal(t, reopened.Snapshot(), store.Snapshot())
}

func TestDetailPaginationDefaultsAndIndependentPersistence(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	require.NoError(t, os.WriteFile(path, []byte(`{"queues":{"table":{"page_size":25}}}`), 0600))
	store, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, DefaultDetailPageSize, store.Snapshot().Queues.Consumers.PageSize)
	require.Equal(t, DefaultDetailPageSize, store.Snapshot().Queues.Bindings.PageSize)
	require.NoError(t, store.Update(func(config *Config) {
		config.Queues.Consumers.PageSize = 5
		config.Queues.Bindings.PageSize = 20
	}))
	reopened, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, 5, reopened.Snapshot().Queues.Consumers.PageSize)
	require.Equal(t, 20, reopened.Snapshot().Queues.Bindings.PageSize)
	require.Equal(t, 25, reopened.Snapshot().Queues.Table.PageSize)
	for _, size := range []int{0, -1, MaxDetailPageSize + 1} {
		require.Error(t, store.Update(func(config *Config) { config.Queues.Consumers.PageSize = size }))
		require.Error(t, store.Update(func(config *Config) { config.Queues.Bindings.PageSize = size }))
	}
	require.Equal(t, reopened.Snapshot(), store.Snapshot())
}
