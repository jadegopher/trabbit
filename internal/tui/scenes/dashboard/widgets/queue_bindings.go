package widgets

import (
	"encoding/json"
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	bindingSourceColumns     = 28
	bindingRoutingKeyColumns = 32
	bindingArgumentsColumns  = 60
)

type QueueBindings struct {
	table          *components.DataTable
	bindings       []domain.QueueBinding
	pager          *components.Pagination
	page, pageSize int
	vhost, name    string
}

func NewQueueBindings(theme core.Theme) *QueueBindings {
	bindings := &QueueBindings{table: components.NewDataTable(theme, []components.DataColumn{
		{Title: "From", Width: bindingSourceColumns},
		{Title: "Routing key", Width: bindingRoutingKeyColumns},
		{Title: "Arguments", Width: bindingArgumentsColumns},
	}, true), pager: components.NewPagination(theme, settings.MaxDetailPageSize), page: 1, pageSize: settings.DefaultDetailPageSize}
	bindings.table.SetRecordLayout(true)
	bindings.Clear()
	return bindings
}

func (b *QueueBindings) SetLoading(vhost, name string) {
	if b.vhost == vhost && b.name == name {
		return
	}
	b.Clear()
	b.vhost, b.name = vhost, name
	b.pager.SetOwner(b.PageOwner())
	b.table.SetStatus("")
}

func (b *QueueBindings) Clear() {
	b.vhost, b.name = "", ""
	b.pager.SetOwner(b.PageOwner())
	b.bindings = nil
	b.page = 1
	b.rebuild()
	b.table.ResetScroll()
	b.table.SetStatus("Select a queue")
}

func (b *QueueBindings) SetData(bindings []domain.QueueBinding) {
	b.bindings = bindings
	b.rebuild()
	b.table.SetStatus("")
	if len(bindings) == 0 {
		b.table.SetStatus("No bindings")
	}
}

func (b *QueueBindings) PageOwner() string { return fmt.Sprintf("bindings:%q/%q", b.vhost, b.name) }

func (b *QueueBindings) rebuild() {
	count := len(b.bindings)
	pages := max(1, (count+b.pageSize-1)/b.pageSize)
	b.page = max(1, min(b.page, pages))
	start := min(count, (b.page-1)*b.pageSize)
	end := min(count, start+b.pageSize)
	rows := make([]components.DataRow, end-start)
	for i, binding := range b.bindings[start:end] {
		source := binding.Source
		if binding.IsDefault() {
			source = "(Default exchange binding)"
		}
		rows[i] = components.DataRow{
			ID:         binding.Source + "\x00" + binding.PropertiesKey,
			Cells:      []string{source, binding.RoutingKey, compactArguments(binding.Arguments)},
			Selectable: !binding.IsDefault(),
		}
	}
	b.table.SetRows(rows)
	b.pager.SetPagination(domain.Pagination{Page: b.page, PageSize: b.pageSize, PageCount: pages, TotalCount: count, FilteredCount: count})
	b.pager.SetStatus(fmt.Sprintf("Total: %d", count))
	b.table.SetFooter(b.pager.View().Content)
}

func (b *QueueBindings) SetError(err error) {
	if len(b.bindings) == 0 {
		b.table.SetStatus("Bindings unavailable: " + err.Error())
	}
}

func (b *QueueBindings) ActiveBinding() *domain.QueueBinding {
	cursor := b.table.Cursor()
	if cursor < 0 {
		return nil
	}
	cursor += (b.page - 1) * b.pageSize
	if cursor < 0 || cursor >= len(b.bindings) || b.bindings[cursor].IsDefault() {
		return nil
	}
	return &b.bindings[cursor]
}

func (b *QueueBindings) SetSize(width, height int) { b.table.SetSize(width, height) }

func (b *QueueBindings) SetFocused(focused bool) { b.table.SetFocused(focused) }

func (b *QueueBindings) Update(msg tea.Msg) tea.Cmd {
	if pressed, ok := msg.(tea.KeyPressMsg); ok && key.Matches(pressed, core.GoToPage()) {
		return b.PaginationSettings()
	}
	if change, ok := msg.(components.ActionPageChange); ok {
		if change.Owner != b.PageOwner() {
			return nil
		}
		b.page = change.Page
		if change.PageSize > 0 {
			b.pageSize = min(settings.MaxDetailPageSize, change.PageSize)
		}
		b.table.ResetScroll()
		b.rebuild()
		return nil
	}
	_, cmd := b.pager.Update(msg)
	b.table.Update(msg)
	b.table.SetFooter(b.pager.View().Content)
	return cmd
}

func (b *QueueBindings) View() tea.View { return b.table.View() }

func compactArguments(arguments map[string]interface{}) string {
	if len(arguments) == 0 {
		return ""
	}
	value, err := json.Marshal(arguments)
	if err != nil {
		return "—"
	}
	return string(value)
}

func queueBool(value bool) string {
	if value {
		return "●"
	}
	return "○"
}

func optionalText(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func optionalMetric(value *int64) string {
	if value == nil || *value < 0 {
		return "—"
	}
	return strconv.FormatInt(*value, 10)
}

// SetPageSize applies a saved preference to the current snapshot without a request.
func (b *QueueBindings) SetPageSize(size int) {
	b.pageSize = max(1, min(settings.MaxDetailPageSize, size))
	b.page = 1
	b.table.ResetScroll()
	b.rebuild()
}

func (b *QueueBindings) PaginationSettings() tea.Cmd {
	modal := b.pager.SettingsModal()
	modal.CloseOnSubmit = false
	return core.Cmd(core.OpenDialog{Model: modal})
}
