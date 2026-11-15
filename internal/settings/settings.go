// Package settings owns shared application preferences and their JSON storage.
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	FileName              = "settings.json"
	DefaultQueuePageSize  = 3
	MaxQueuePageSize      = 1000
	DefaultDetailPageSize = 100
	MaxDetailPageSize     = 500
)

// Stable identifiers keep stored queue subtabs independent of labels and order.
const (
	QueueSubtabOverview  = "overview"
	QueueSubtabConsumers = "consumers"
	QueueSubtabBindings  = "bindings"
	QueueSubtabRuntime   = "runtime"
)

// Section keeps preferences for tabs whose settings have not been defined yet.
type Section map[string]json.RawMessage

type Config struct {
	Overview    Section       `json:"overview"`
	Connections Section       `json:"connections"`
	Channels    Section       `json:"channels"`
	Exchanges   Section       `json:"exchanges"`
	Queues      QueueSettings `json:"queues"`
	Admin       Section       `json:"admin"`
}

type QueueSettings struct {
	SelectedSubtab string        `json:"selected_subtab"`
	Table          TableSettings `json:"table"`
	Overview       Section       `json:"overview"`
	Charts         ChartSettings `json:"charts"`
	Consumers      TableSettings `json:"consumers"`
	Bindings       TableSettings `json:"bindings"`
}

type TableSettings struct {
	PageSize int `json:"page_size"`
}

func Defaults() Config {
	return Config{
		Overview: Section{}, Connections: Section{}, Channels: Section{},
		Exchanges: Section{}, Admin: Section{},
		Queues: QueueSettings{
			SelectedSubtab: QueueSubtabOverview, Table: TableSettings{PageSize: DefaultQueuePageSize},
			Overview: Section{}, Charts: DefaultChartSettings(),
			Consumers: TableSettings{PageSize: DefaultDetailPageSize},
			Bindings:  TableSettings{PageSize: DefaultDetailPageSize},
		},
	}
}

// Store serializes updates from all tabs. Callers read copies and persist edits
// in commands; a failed write leaves the shared preferences unchanged.
type Store struct {
	mu     sync.RWMutex
	path   string
	config Config
}

func NewMemory() *Store {
	return &Store{config: Defaults()}
}

func OpenBesideExecutable() (*Store, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate executable for settings: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, fmt.Errorf("resolve executable for settings: %w", err)
	}
	return Open(filepath.Join(filepath.Dir(executable), FileName))
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, config: Defaults()}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := s.Update(func(*Config) {}); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read settings %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &s.config); err != nil {
		return nil, fmt.Errorf("decode settings %s: %w", path, err)
	}
	if err := validate(s.config); err != nil {
		return nil, fmt.Errorf("invalid settings %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) Snapshot() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clone(s.config)
}

// Update holds the lock through the atomic file replacement, so edits from
// different tabs cannot overwrite one another. The callback must not call Store.
func (s *Store) Update(edit func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := clone(s.config)
	edit(&next)
	if err := validate(next); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if s.path != "" {
		if err := writeFile(s.path, append(data, '\n')); err != nil {
			return fmt.Errorf("save settings %s: %w", s.path, err)
		}
	}
	s.config = clone(next)
	return nil
}

func validate(config Config) error {
	switch config.Queues.SelectedSubtab {
	case QueueSubtabOverview, QueueSubtabConsumers, QueueSubtabBindings, QueueSubtabRuntime:
	default:
		return fmt.Errorf("queues.selected_subtab is invalid: %q", config.Queues.SelectedSubtab)
	}

	if err := config.Queues.Charts.Validate(); err != nil {
		return fmt.Errorf("queues.charts: %w", err)
	}
	for name, size := range map[string]int{
		"consumers": config.Queues.Consumers.PageSize,
		"bindings":  config.Queues.Bindings.PageSize,
	} {
		if size < 1 || size > MaxDetailPageSize {
			return fmt.Errorf("queues.%s.page_size must be from 1 to %d", name, MaxDetailPageSize)
		}
	}
	if size := config.Queues.Table.PageSize; size < 1 || size > MaxQueuePageSize {
		return fmt.Errorf("queues.table.page_size must be from 1 to %d", MaxQueuePageSize)
	}
	return nil
}

func clone(config Config) Config {
	config.Overview = cloneSection(config.Overview)
	config.Connections = cloneSection(config.Connections)
	config.Channels = cloneSection(config.Channels)
	config.Exchanges = cloneSection(config.Exchanges)
	config.Admin = cloneSection(config.Admin)
	config.Queues.Overview = cloneSection(config.Queues.Overview)
	config.Queues.Charts = config.Queues.Charts.Clone()
	return config
}

func cloneSection(section Section) Section {
	result := make(Section, len(section))
	for name, value := range section {
		result[name] = append(json.RawMessage(nil), value...)
	}
	return result
}

func writeFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	// Check close errors before replacing the saved file; the defer handles early returns.
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
