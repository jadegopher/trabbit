package widgets

import (
	"fmt"
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	consumerChannelColumns   = 30
	consumerTagColumns       = 48
	consumerAckColumns       = 12
	consumerExclusiveColumns = 9
	consumerPrefetchColumns  = 14
	consumerActiveColumns    = 6
	consumerActivityColumns  = 15
	consumerTimeoutColumns   = 17
	consumerMillisPerSecond  = int64(time.Second / time.Millisecond)
	consumerArgumentsColumns = 40
)

type QueueConsumers struct {
	table          *components.DataTable
	pager          *components.Pagination
	consumers      []domain.QueueConsumer
	vhost, name    string
	page, pageSize int
}

func NewQueueConsumers(theme core.Theme) *QueueConsumers {
	consumers := &QueueConsumers{
		table: components.NewDataTable(theme, []components.DataColumn{
			{Title: "Channel", Width: consumerChannelColumns},
			{Title: "Consumer tag", Width: consumerTagColumns},
			{Title: "Ack required", Width: consumerAckColumns},
			{Title: "Exclusive", Width: consumerExclusiveColumns},
			{Title: "Prefetch count", Width: consumerPrefetchColumns},
			{Title: "Active", Width: consumerActiveColumns},
			{Title: "Activity status", Width: consumerActivityColumns},
			{Title: "Timeout (seconds)", Width: consumerTimeoutColumns},
			{Title: "Arguments", Width: consumerArgumentsColumns},
		}, false),
		pager: components.NewPagination(theme, settings.MaxDetailPageSize),
		page:  1, pageSize: settings.DefaultDetailPageSize,
	}
	consumers.table.SetResponsive(true)
	consumers.Clear()
	return consumers
}

func (c *QueueConsumers) PageOwner() string { return fmt.Sprintf("consumers:%q/%q", c.vhost, c.name) }

func (c *QueueConsumers) SetLoading(vhost, name string) {
	if c.vhost == vhost && c.name == name {
		return
	}
	c.Clear()
	c.vhost, c.name = vhost, name
	c.pager.SetOwner(c.PageOwner())
	c.table.SetStatus("")
}

func (c *QueueConsumers) Clear() {
	c.vhost, c.name = "", ""
	c.pager.SetOwner(c.PageOwner())
	c.consumers = nil
	c.page = 1
	c.table.ResetScroll()
	c.rebuild()
	c.table.SetStatus("Select a queue")
}

func (c *QueueConsumers) SetData(consumers []domain.QueueConsumer) {
	c.consumers = consumers
	c.table.SetStatus("")
	c.rebuild()
	if len(consumers) == 0 {
		c.table.SetStatus("No consumers")
	}
}

func (c *QueueConsumers) SetError(err error) {
	if len(c.consumers) == 0 {
		c.table.SetStatus("Consumers unavailable: " + err.Error())
	}
}

// Queue details return the complete subscription list. Page that snapshot
// locally instead of applying vhost-wide page counts to one queue.
func (c *QueueConsumers) rebuild() {
	count := len(c.consumers)
	pages := max(1, (count+c.pageSize-1)/c.pageSize)
	c.page = max(1, min(c.page, pages))
	start := min(count, (c.page-1)*c.pageSize)
	end := min(count, start+c.pageSize)
	rows := make([]components.DataRow, 0, end-start)
	for _, consumer := range c.consumers[start:end] {
		active := "—"
		if consumer.Active != nil {
			active = queueBool(*consumer.Active)
		}
		rows = append(rows, components.DataRow{Cells: []string{
			consumer.Channel, consumer.Tag, queueBool(consumer.AckRequired), queueBool(consumer.Exclusive),
			strconv.Itoa(consumer.PrefetchCount), active, optionalText(consumer.ActivityStatus),
			consumerTimeoutSeconds(consumer.TimeoutMillis), compactArguments(consumer.Arguments),
		}})
	}
	c.table.SetRows(rows)
	c.pager.SetPagination(domain.Pagination{Page: c.page, PageSize: c.pageSize, PageCount: pages, TotalCount: count, FilteredCount: count})
	c.pager.SetStatus(fmt.Sprintf("Total: %d", count))
	c.updateFooter()
}

func (c *QueueConsumers) updateFooter() { c.table.SetFooter(c.pager.View().Content) }

func (c *QueueConsumers) SetSize(width, height int) { c.table.SetSize(width, height) }

func (c *QueueConsumers) SetFocused(focused bool) { c.table.SetFocused(focused) }

func (c *QueueConsumers) Update(msg tea.Msg) tea.Cmd {
	if pressed, ok := msg.(tea.KeyPressMsg); ok && key.Matches(pressed, core.GoToPage()) {
		return c.PaginationSettings()
	}
	if change, ok := msg.(components.ActionPageChange); ok {
		if change.Owner != c.PageOwner() {
			return nil
		}
		c.page = change.Page
		if change.PageSize > 0 {
			c.pageSize = min(settings.MaxDetailPageSize, change.PageSize)
		}
		c.table.ResetScroll()
		c.rebuild()
		return nil
	}
	_, cmd := c.pager.Update(msg)
	c.table.Update(msg)
	c.updateFooter()
	return cmd
}

func (c *QueueConsumers) View() tea.View { return c.table.View() }

func consumerTimeoutSeconds(milliseconds *int64) string {
	if milliseconds == nil || *milliseconds < 0 {
		return "—"
	}
	return strconv.FormatFloat(float64(*milliseconds)/float64(consumerMillisPerSecond), 'f', -1, 64)
}

// SetPageSize applies a saved preference to the current snapshot without a request.
func (c *QueueConsumers) SetPageSize(size int) {
	c.pageSize = max(1, min(settings.MaxDetailPageSize, size))
	c.page = 1
	c.table.ResetScroll()
	c.rebuild()
}

func (c *QueueConsumers) PaginationSettings() tea.Cmd {
	modal := c.pager.SettingsModal()
	modal.CloseOnSubmit = false
	return core.Cmd(core.OpenDialog{Model: modal})
}
