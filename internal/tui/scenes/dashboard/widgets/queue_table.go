package widgets

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	colVHostTitle = "VHost"
	colNameTitle  = "Name"
	colTypeTitle  = "Type"
	colStateTitle = "State"

	colVHostWidth = 5
	colNameWidth  = 15
	colTypeWidth  = 10
	colStateWidth = 10

	colVHostGrowWeight = 1
	colNameGrowWeight  = 2
)

// Table owns queue snapshots and selection messages. DataTable owns the viewport,
// selection behavior and pinned footer, as it does for the detail tables.
type Table struct {
	filter        string
	table         *components.DataTable
	theme         core.Theme
	pagination    *components.Pagination
	loader        components.Spinner
	totalCount    int
	filteredCount int
	queues        domain.Queues
	fetchID       int
}

const QueueTableComponentName = "queue-table"

type TableOption func(*Table)

// WithPagination adds page navigation and the GoTo settings dialog.
func WithPagination(maxPageSize int) TableOption {
	return func(t *Table) {
		t.pagination = components.NewPagination(t.theme, maxPageSize)
		t.updatePaginationStatus()
	}
}

func NewTable(theme core.Theme, options ...TableOption) *Table {
	table := components.NewDataTable(theme, []components.DataColumn{
		{Title: colVHostTitle, Width: colVHostWidth, Grow: colVHostGrowWeight},
		{Title: colNameTitle, Width: colNameWidth, Grow: colNameGrowWeight},
		{Title: colTypeTitle, Width: colTypeWidth},
		{Title: colStateTitle, Width: colStateWidth},
	}, true)
	table.SetTruncateCells(true)
	table.SetWrapNavigation(true)
	table.SetFocused(true)
	result := &Table{
		table:  table,
		theme:  theme,
		loader: components.NewSpinner(theme, "Loading queues..."),
	}
	for _, option := range options {
		option(result)
	}
	return result
}

func (t *Table) SetFocused(focused bool) { t.table.SetFocused(focused) }

func (t *Table) ActiveQueue() *domain.Queue {
	cursor := t.table.Cursor()
	if cursor < 0 || cursor >= len(t.queues) {
		return nil
	}
	return &t.queues[cursor]
}

func (t *Table) Init() tea.Cmd {
	if t.pagination != nil {
		return t.pagination.Init()
	}
	return nil
}

func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if pressed, ok := msg.(tea.KeyPressMsg); ok && key.Matches(pressed, core.SwitchFocus()) {
		return t, nil
	}

	selected := t.ActiveQueue()
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		t.setSize(m.Width, m.Height)
	case MsgQueues:
		if m.RequestID < t.fetchID {
			return t, nil
		}
		t.fetchID = m.RequestID
		t.queues = m.Queues
		t.totalCount, t.filteredCount = m.Pagination.TotalCount, m.Pagination.FilteredCount
		rows := make([]components.DataRow, len(m.Queues))
		for i, queue := range m.Queues {
			rows[i] = components.DataRow{
				ID:         queue.VHost + "\x00" + queue.Name,
				Cells:      []string{queue.VHost, queue.Name, queue.Type, queue.State},
				Selectable: true,
			}
		}
		t.table.SetRows(rows)
		t.FinishLoading()
		cmds = append(cmds, t.updatePagination(m.Pagination))
		t.updatePaginationStatus()
	default:
		cmds = append(cmds, t.updatePagination(msg), t.loader.Update(msg), t.table.Update(msg))
		if t.loader.IsActive() {
			t.table.SetStatus(t.loader.View())
		}
	}

	active := t.ActiveQueue()
	if active != nil && (selected == nil || selected.VHost != active.VHost || selected.Name != active.Name) {
		cmds = append(cmds, core.Cmd(MsgQueueSelectionChanged{VHost: active.VHost, Name: active.Name}))
	}
	return t, tea.Batch(cmds...)
}

func (t *Table) TotalWidth() int { return t.table.MinimumWidth() }

// The scene supplies the outer panel size; shared tables accept its inner size.
func (t *Table) setSize(width, height int) {
	t.table.SetSize(max(0, width-core.BorderOverhead), max(0, height-core.BorderOverhead))
}

func (t *Table) View() tea.View { return t.table.View() }

func (t *Table) SetLoading(page int) tea.Cmd {
	return t.updatePagination(components.ActionPageChange{Page: page})
}

func (t *Table) FinishLoading() {
	t.loader.Stop()
	t.table.SetStatus("")
	t.updatePagination(components.ActionPageChangeComplete{})
}

func (t *Table) SetVisible(visible bool) tea.Cmd {
	if !visible {
		t.loader.Stop()
		t.table.SetStatus("")
	}
	return nil
}

func (t *Table) ShowLoading() tea.Cmd {
	cmd := t.loader.Start()
	t.table.SetStatus(t.loader.View())
	return cmd
}

func (t *Table) SetFilter(label string) {
	t.filter = label
	t.updatePaginationStatus()
}

// The footer count describes the current filter across all pages.
func (t *Table) updatePaginationStatus() {
	if t.pagination == nil {
		t.table.SetFooter(t.filter)
		return
	}
	count := t.totalCount
	if t.filter != "" {
		count = t.filteredCount
	}
	status := fmt.Sprintf("Total: %d", count)
	if t.filter != "" {
		status += t.theme.RenderHelpSeparator() + t.filter
	}
	t.pagination.SetStatus(status)
	t.table.SetFooter(t.pagination.View().Content)
}

func (t *Table) updatePagination(msg tea.Msg) tea.Cmd {
	if t.pagination == nil {
		return nil
	}
	_, cmd := t.pagination.Update(msg)
	t.table.SetFooter(t.pagination.View().Content)
	return cmd
}
