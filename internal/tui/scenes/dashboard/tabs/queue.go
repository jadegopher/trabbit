package tabs

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish"
	publishwidgets "github.com/jadegopher/trabbit/internal/tui/scenes/publish/widgets"
)

type ActionDeleteQueue struct{ VHost, Name string }
type ActionPurgeQueue struct{ VHost, Name string }
type loadingDelay struct {
	id      int
	details bool
}

func delayedLoading(id int, details bool) tea.Cmd {
	return tea.Tick(queueLoadingDelay, func(time.Time) tea.Msg { return loadingDelay{id, details} })
}

type chartSettingsSaved struct {
	dialog   tea.Model
	settings settings.ChartSettings
	err      error
}

type paginationSettingsSaved struct {
	change components.ActionPageChange
	err    error
}

type SelectionChanged struct{}
type detailsDebounceExpired struct {
	generation  int
	vhost, name string
}

type RefreshFinished struct{ Err error }

// RefreshFailed reports detail polling failures without rescheduling list polling.
type RefreshFailed struct{ Err error }
type listResult struct {
	id   int
	data domain.PagedQueues
	err  error
}
type detailResult struct {
	id          int
	vhost, name string
	data        domain.QueueDetailedStats
	err         error
}
type vhostsResult struct {
	id     int
	dialog tea.Model
	target string
	data   domain.VHosts
	err    error
}
type operationResult struct {
	dialog tea.Model
	err    error
}
type queueFocus int

const (
	focusTable queueFocus = iota
	focusDetails
)
const (
	helpHeight        = 1
	minDetailsWidth   = 30
	maxQueuePageSize  = settings.MaxQueuePageSize
	detailsDebounce   = 200 * time.Millisecond
	queueLoadingDelay = 250 * time.Millisecond
)

type queueShortcuts struct{ AddQueue, DeleteQueue, PurgeQueue, Publish, Filter, ViewSettings, SwitchFocus, Subtabs key.Binding }

func (b queueShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{b.AddQueue, b.DeleteQueue, b.PurgeQueue, b.Publish, b.Filter, b.ViewSettings, b.Subtabs}
}
func (b queueShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{b.ShortHelp(), {b.SwitchFocus}}
}

type QueueTab struct {
	settings                    *settings.Store
	savingSettings              bool
	savingSubtab                bool
	subtabSaveID                int
	savingChartSettings         bool
	chartSettingsDialog         tea.Model
	detailPaginationSaves       map[string]int
	detailPaginationSaveID      int
	detailsFetching             bool
	detailsPending              bool
	detailsDebouncer            *core.Debouncer
	theme                       core.Theme
	service                     *app.Service
	ctx                         context.Context
	table                       *widgets.Table
	charts                      *widgets.QueueDetails
	detailsLoader               components.Spinner
	detailsLoadingPanel         components.ChartPanel
	consumers                   *widgets.QueueConsumers
	queueBindings               *widgets.QueueBindings
	runtime                     *widgets.QueueRuntime
	bindingBusy                 bool
	bindingOperationID          int
	bindingDialog               tea.Model
	subtabs                     components.Subtabs
	query                       app.ListParameters
	displayedPage               int
	displayedPageSize           int
	focus                       queueFocus
	bindings                    queueShortcuts
	width, height               int
	listID, detailsID, vhostsID int
	listCancel, detailsCancel   context.CancelFunc
	visible                     bool
	fetching                    bool
	lastQuery                   app.ListParameters
}

func NewQueueTab(theme core.Theme, service *app.Service, ctx context.Context, preferences *settings.Store) *QueueTab {
	config := preferences.Snapshot().Queues
	pageSize := config.Table.PageSize
	activeSubtab := queueOverview
	for i, id := range queueSubtabIDs {
		if id == config.SelectedSubtab {
			activeSubtab = i
		}
	}
	charts := widgets.NewQueueCharts(theme)
	charts.Update(widgets.MsgChartSettings{Settings: config.Charts})
	consumers := widgets.NewQueueConsumers(theme)
	consumers.SetPageSize(config.Consumers.PageSize)
	bindings := widgets.NewQueueBindings(theme)
	bindings.SetPageSize(config.Bindings.PageSize)
	return &QueueTab{
		detailPaginationSaves: make(map[string]int),
		detailsDebouncer:      core.NewDebouncer(),
		subtabs:               components.Subtabs{Labels: []string{"Overview", "Consumers", "Bindings", "Runtime metrics"}, Active: activeSubtab},
		settings:              preferences,
		theme:                 theme,
		service:               service,
		ctx:                   ctx,
		table:                 widgets.NewTable(theme, widgets.WithPagination(maxQueuePageSize)),
		charts:                charts,
		detailsLoader:         components.NewSpinner(theme, "Loading details..."),
		detailsLoadingPanel:   components.NewChartPanel(theme),
		consumers:             consumers,
		queueBindings:         bindings,
		runtime:               widgets.NewQueueRuntime(theme),
		query:                 app.ListParameters{Page: 1, PageSize: pageSize},
		displayedPage:         1,
		displayedPageSize:     pageSize,
		visible:               true,
		bindings: queueShortcuts{
			AddQueue:     core.AddQueue(),
			DeleteQueue:  core.DeleteQueue(),
			PurgeQueue:   core.PurgeQueue(),
			Publish:      core.PublishQueue(),
			Filter:       core.Filter(),
			ViewSettings: core.ViewSettings(),
			SwitchFocus:  core.SwitchFocus(),
			Subtabs:      core.SwitchSubtab(),
		},
	}
}
func (q *QueueTab) Init() tea.Cmd { return nil }
func (q *QueueTab) Refresh() tea.Cmd {
	if q.fetching && q.query == q.lastQuery {
		return nil
	}
	q.fetching = true
	q.lastQuery = q.query
	if q.listCancel != nil {
		q.listCancel()
	}
	ctx, cancel := context.WithCancel(q.ctx)
	q.listCancel = cancel
	q.listID++
	id, query, service := q.listID, q.query, q.service
	q.table.SetLoading(query.Page)
	tick := delayedLoading(id, false)
	if !q.visible {
		q.table.SetVisible(false)
		tick = nil
	}
	return tea.Batch(
		tick,
		func() tea.Msg { data, err := service.ListQueues(ctx, query); return listResult{id, data, err} },
	)
}

// Schedule details for the current selection. Request IDs reject old network
// results and loader ticks; the debouncer owns timer generations.
func (q *QueueTab) scheduleDetails() tea.Cmd {
	q.detailsDebouncer.Cancel()
	if q.detailsCancel != nil {
		q.detailsCancel()
	}
	q.detailsID++
	q.detailsFetching = false
	q.detailsPending = true
	q.detailsLoader.Stop()

	active := q.table.ActiveQueue()
	if active == nil {
		q.detailsPending = false
		q.clearDetails()
		return nil
	}
	q.prepareDetails(active.VHost, active.Name)
	if !q.visible {
		return nil
	}
	vhost, name := active.VHost, active.Name
	return q.detailsDebouncer.Trigger(
		detailsDebounce, func(generation int) tea.Msg {
			return detailsDebounceExpired{generation: generation, vhost: vhost, name: name}
		},
	)
}

func (q *QueueTab) fetchDetails() tea.Cmd {
	if !q.visible {
		return q.scheduleDetails()
	}
	q.detailsDebouncer.Cancel()
	if q.detailsCancel != nil {
		q.detailsCancel()
	}
	q.detailsID++
	q.detailsPending = false
	q.detailsFetching = false
	q.detailsLoader.Stop()
	active := q.table.ActiveQueue()
	if active == nil {
		q.clearDetails()
		return nil
	}
	id, vhost, name, service := q.detailsID, active.VHost, active.Name, q.service
	history := q.settings.Snapshot().Queues.Charts.Duration()
	ctx, cancel := context.WithCancel(q.ctx)
	q.detailsCancel = cancel
	q.detailsFetching = true
	q.prepareDetails(vhost, name)
	tick := delayedLoading(id, true)
	if !q.visible {
		q.detailsLoader.Stop()
		tick = nil
	}
	if q.subtabs.Active == queueBindings {
		return tea.Batch(tick, func() tea.Msg {
			data, err := service.ListQueueBindings(ctx, vhost, name)
			return bindingsResult{id: id, vhost: vhost, name: name, data: data, err: err}
		})
	}
	return tea.Batch(
		tick, func() tea.Msg {
			data, err := service.GetQueueDetailedStats(ctx, vhost, name, history)
			return detailResult{id, vhost, name, data, err}
		},
	)
}

func (q *QueueTab) SetVisible(visible bool) tea.Cmd {
	wasVisible := q.visible
	q.visible = visible
	var debounce tea.Cmd
	if wasVisible != visible {
		q.detailsDebouncer.Cancel()
		if q.detailsCancel != nil {
			q.detailsCancel()
		}
		q.detailsID++
		q.detailsFetching = false
		q.detailsLoader.Stop()
		q.detailsPending = q.table.ActiveQueue() != nil
		if visible && q.detailsPending {
			debounce = q.scheduleDetails()
		}
	}
	return tea.Batch(q.table.SetVisible(visible), debounce)
}

func (q *QueueTab) openVHosts(target string) tea.Cmd {
	q.vhostsID++
	id := q.vhostsID
	dialog := components.NewLoader("Loading virtual hosts...", q.theme)
	ctx, service := q.ctx, q.service
	return core.Cmd(
		core.OpenDialog{
			Model: dialog, After: func() tea.Msg {
				data, err := service.ListVHosts(ctx)
				return vhostsResult{id, dialog, target, data, err}
			},
		},
	)
}
func (q *QueueTab) operation(dialog tea.Model, run func() error) tea.Cmd {
	cmd := func() tea.Msg { return operationResult{dialog: dialog, err: run()} }
	if _, ok := dialog.(*components.FormModal); ok {
		return cmd
	}
	loader := components.NewLoader("Working...", q.theme)
	cmd = func() tea.Msg { return operationResult{dialog: loader, err: run()} }
	return core.Cmd(core.OpenDialog{Model: loader, After: cmd})
}
func (q *QueueTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		q.width = m.Width
		q.height = m.Height
		_, content := q.getLayout()
		q.table.SetFilter(q.filterLabel())
		q.table.Update(content.GetSize("table"))
		sz := content.GetSize("queue_details")
		inner := tea.WindowSizeMsg{Width: max(0, sz.Width-core.BorderOverhead), Height: max(0, sz.Height-core.BorderOverhead)}
		q.detailsLoadingPanel.SetSize(inner.Width, inner.Height)
		q.charts.Update(inner)
		q.consumers.SetSize(inner.Width, inner.Height)
		q.queueBindings.SetSize(inner.Width, inner.Height)
		q.runtime.SetSize(inner.Width, inner.Height)
		return q, nil
	case loadingDelay:
		if !q.visible {
			return q, nil
		}
		if m.details {
			if m.id == q.detailsID && q.detailsFetching {
				return q, q.detailsLoader.Start()
			}
		} else if m.id == q.listID && q.fetching {
			return q, q.table.ShowLoading()
		}
		return q, nil
	case detailsDebounceExpired:
		if !q.detailsDebouncer.IsValid(m.generation) || !q.detailsPending || !q.visible {
			return q, nil
		}
		if active := q.table.ActiveQueue(); active != nil && active.VHost == m.vhost && active.Name == m.name {
			return q, q.fetchDetails()
		}
		return q, nil
	case listResult:
		if m.id != q.listID {
			return q, nil
		}
		q.fetching = false
		q.table.FinishLoading()
		if m.err != nil {
			q.query.Page = q.displayedPage
			q.query.PageSize = q.displayedPageSize
			return q, core.Cmd(RefreshFinished{Err: m.err})
		}
		if len(m.data.Items) == 0 && q.query.Page > 1 {
			q.query.Page = max(1, min(q.query.Page-1, m.data.Pagination.PageCount))
			return q, q.Refresh()
		}
		q.query.Page = max(1, m.data.Pagination.Page)
		q.displayedPage = q.query.Page
		q.displayedPageSize = q.query.PageSize
		q.table.Update(widgets.MsgQueues{Queues: m.data.Items, Pagination: m.data.Pagination, RequestID: m.id})
		if q.detailsPending {
			return q, tea.Batch(q.scheduleDetails(), core.Cmd(RefreshFinished{}))
		}
		return q, tea.Batch(q.fetchDetails(), core.Cmd(RefreshFinished{}))
	case detailResult:
		if m.id != q.detailsID {
			return q, nil
		}
		if active := q.table.ActiveQueue(); active == nil || active.VHost != m.vhost || active.Name != m.name {
			return q, nil
		}
		q.detailsFetching = false
		q.detailsLoader.Stop()
		if m.err != nil {
			q.detailError(m.err)
			return q, core.Cmd(RefreshFailed{Err: m.err})
		}
		switch q.subtabs.Active {
		case queueConsumers:
			q.consumers.SetData(m.data.ConsumerDetails)
		case queueRuntime:
			q.runtime.SetData(m.data.Runtime)
		default:
			q.charts.Update(widgets.MsgQueueDetailedStats{VHost: m.vhost, Name: m.name, FetchID: m.id, Stats: m.data})
		}
		return q, nil
	case bindingsResult:
		active := q.table.ActiveQueue()
		if m.id != q.detailsID || q.subtabs.Active != queueBindings || active == nil || active.VHost != m.vhost || active.Name != m.name {
			return q, nil
		}
		q.detailsFetching = false
		q.detailsLoader.Stop()
		if m.err != nil {
			q.queueBindings.SetError(m.err)
			return q, core.Cmd(RefreshFailed{Err: m.err})
		}
		q.queueBindings.SetData(m.data)
		return q, nil
	case bindingOperationResult:
		if !q.bindingBusy || m.id != q.bindingOperationID || m.dialog != q.bindingDialog {
			return q, nil
		}
		q.bindingBusy = false
		q.bindingDialog = nil
		if m.err != nil {
			if _, ok := m.dialog.(*components.FormModal); ok {
				return q, core.Cmd(core.DialogFeedback{Dialog: m.dialog, Err: m.err})
			}
			return q, core.Cmd(core.ReplaceDialog{Old: m.dialog, Model: components.NewError(m.err, q.theme)})
		}
		var refresh tea.Cmd
		if active := q.table.ActiveQueue(); q.visible && q.subtabs.Active == queueBindings && active != nil && active.VHost == m.vhost && active.Name == m.queue {
			refresh = q.fetchDetails()
		}
		return q, tea.Batch(core.Cmd(core.DialogResult{Dialog: m.dialog, Close: true}), refresh)
	case vhostsResult:
		if m.id != q.vhostsID {
			return q, nil
		}
		var dialog tea.Model
		if m.err != nil {
			dialog = components.NewError(m.err, q.theme)
		} else if m.target == "add" {
			dialog = widgets.NewModalAddQueue(q.theme, m.data.Names())
		} else {
			dialog = widgets.NewModalFilter(
				q.theme,
				m.data.Names(),
				widgets.ActionFilterQueues{Filter: q.query.Name, VHost: q.query.VHost, UseRegex: q.query.UseRegex},
			)
		}
		return q, core.Cmd(core.ReplaceDialog{Old: m.dialog, Model: dialog})
	case operationResult:
		if m.err != nil {
			if _, ok := m.dialog.(*components.FormModal); ok {
				return q, core.Cmd(core.DialogFeedback{Dialog: m.dialog, Err: m.err})
			}
			return q, core.Cmd(core.ReplaceDialog{Old: m.dialog, Model: components.NewError(m.err, q.theme)})
		}
		return q, tea.Batch(core.Cmd(core.DialogResult{Dialog: m.dialog, Close: true}), q.Refresh())
	case chartSettingsSaved:
		if !q.savingChartSettings || m.dialog != q.chartSettingsDialog {
			return q, nil
		}
		q.savingChartSettings = false
		q.chartSettingsDialog = nil
		if m.err != nil {
			return q, core.Cmd(core.DialogFeedback{Dialog: m.dialog, Err: m.err})
		}
		q.charts.Update(widgets.MsgChartSettings{Settings: m.settings})
		return q, tea.Batch(core.Cmd(core.DialogResult{Dialog: m.dialog, Close: true}), q.fetchDetails())
	case core.DialogResult:
		switch value := m.Value.(type) {
		case widgets.ActionChartSettings:
			if q.savingChartSettings {
				return q, nil
			}
			q.savingChartSettings = true
			q.chartSettingsDialog = m.Dialog
			preferences, chosen, dialog := q.settings, value.Settings, m.Dialog
			return q, func() tea.Msg {
				err := preferences.Update(func(config *settings.Config) { config.Queues.Charts = chosen })
				return chartSettingsSaved{dialog: dialog, settings: chosen, err: err}
			}
		case components.ActionPageChange:
			if value.Owner != "" {
				return q, q.changeDetailPage(value, m.Dialog)
			}
			return q.Update(value)
		case widgets.ActionFilterQueues:
			q.query.Name = value.Filter
			q.query.VHost = value.VHost
			q.query.UseRegex = value.UseRegex
			q.query.Page = 1
			q.Update(tea.WindowSizeMsg{Width: q.width, Height: q.height})
			return q, q.Refresh()
		case widgets.ActionCreateBinding:
			ctx, service, params := q.ctx, q.service, value.Params
			return q, q.bindingOperation(m.Dialog, params.VHost, params.Queue, func() error { return service.CreateQueueBinding(ctx, params) })
		case widgets.ActionUnbind:
			ctx, service, binding := q.ctx, q.service, value.Binding
			return q, q.bindingOperation(m.Dialog, binding.VHost, binding.Queue, func() error { return service.DeleteQueueBinding(ctx, binding) })
		case widgets.ActionCreateQueue:
			service, ctx, params := q.service, q.ctx, value.Params
			return q, q.operation(m.Dialog, func() error { return service.CreateQueue(ctx, params) })
		case ActionDeleteQueue:
			service, ctx := q.service, q.ctx
			return q, q.operation(
				m.Dialog,
				func() error { return service.DeleteQueue(ctx, value.VHost, value.Name) },
			)
		case ActionPurgeQueue:
			service, ctx := q.service, q.ctx
			return q, q.operation(
				m.Dialog,
				func() error { return service.PurgeQueue(ctx, value.VHost, value.Name) },
			)

		}
		return q, nil
	case subtabSettingsSaved:
		if !q.savingSubtab || m.id != q.subtabSaveID {
			return q, nil
		}
		q.savingSubtab = false
		var feedback tea.Cmd
		if m.err != nil {
			feedback = core.Cmd(core.OpenDialog{Model: components.NewError(m.err, q.theme)})
		}
		if m.selected != queueSubtabIDs[q.subtabs.Active] {
			return q, tea.Batch(feedback, q.saveSubtab())
		}
		return q, feedback
	case detailPaginationSaved:
		return q, q.finishDetailPagination(m)
	case paginationSettingsSaved:
		q.savingSettings = false
		q.table.FinishLoading()
		if m.err != nil {
			return q, core.Cmd(core.OpenDialog{Model: components.NewError(m.err, q.theme)})
		}
		return q.Update(m.change)
	case components.ActionPageChange:
		if m.Owner != "" {
			return q, q.changeDetailPage(m, nil)
		}
		if q.savingSettings {
			return q, nil
		}
		if m.PageSize != 0 && m.PageSize != q.settings.Snapshot().Queues.Table.PageSize {
			// Block pagination until the save completes so later edits cannot
			// overtake this write or fetch a page with an unsaved size.
			q.savingSettings = true
			q.table.SetLoading(m.Page)
			preferences, change := q.settings, m
			return q, func() tea.Msg {
				err := preferences.Update(
					func(config *settings.Config) {
						config.Queues.Table.PageSize = change.PageSize
					},
				)
				return paginationSettingsSaved{change: change, err: err}
			}
		}
		q.query.Page = m.Page
		if m.PageSize != 0 {
			q.query.PageSize = m.PageSize
		}
		return q, q.Refresh()
	case widgets.MsgQueueSelectionChanged:
		if active := q.table.ActiveQueue(); active != nil && active.VHost == m.VHost && active.Name == m.Name {
			return q, tea.Batch(q.scheduleDetails(), core.Cmd(SelectionChanged{}))
		}
		return q, nil
	case tea.KeyPressMsg:
		if key.Matches(m, q.bindings.Subtabs) {
			delta := 1
			if m.String() == core.KeyOpenBracket {
				delta = -1
			}
			q.subtabs.Move(delta)
			q.focus = focusDetails
			q.focusPanels()
			return q, tea.Batch(q.scheduleDetails(), q.saveSubtab())
		}
		if cmd := q.detailActions(m); cmd != nil {
			return q, cmd
		}
		switch {
		case key.Matches(m, q.bindings.ViewSettings) && q.subtabs.Active == queueConsumers:
			return q, q.consumers.PaginationSettings()
		case key.Matches(m, q.bindings.ViewSettings) && q.subtabs.Active == queueBindings:
			return q, q.queueBindings.PaginationSettings()
		case key.Matches(m, q.bindings.ViewSettings) && q.subtabs.Active == queueOverview:
			return q, core.Cmd(
				core.OpenDialog{
					Model: widgets.NewModalChartSettings(
						q.theme,
						q.settings.Snapshot().Queues.Charts,
					),
				},
			)
		case key.Matches(m, q.bindings.SwitchFocus):
			if q.focus == focusTable {
				q.focus = focusDetails
			} else {
				q.focus = focusTable
			}
			q.focusPanels()
			return q, nil
		case key.Matches(m, q.bindings.AddQueue):
			return q, q.openVHosts("add")
		case key.Matches(m, q.bindings.Filter):
			return q, q.openVHosts("filter")
		case key.Matches(m, q.bindings.DeleteQueue), key.Matches(m, q.bindings.PurgeQueue):
			if active := q.table.ActiveQueue(); active != nil {
				var result tea.Msg = ActionDeleteQueue{active.VHost, active.Name}
				verb := "delete"
				if key.Matches(m, q.bindings.PurgeQueue) {
					result = ActionPurgeQueue{active.VHost, active.Name}
					verb = "purge"
				}
				dialog := components.NewConfirm(
					fmt.Sprintf(
						"%s queue %s in vhost %s?",
						verb,
						active.Name,
						active.VHost,
					), q.theme,
				).WithResult(result)
				return q, core.Cmd(core.OpenDialog{Model: dialog})
			}
			return q, nil
		case key.Matches(m, q.bindings.Publish):
			if active := q.table.ActiveQueue(); active != nil {
				target := publishwidgets.PublishTarget{
					VHost:      active.VHost,
					Exchange:   app.DefaultExchange,
					RoutingKey: active.Name,
				}
				return q, core.Cmd(core.OpenScreen{Model: publish.New(q.theme, q.service, target)})
			}
			return q, nil
		}
	case tea.MouseClickMsg:
		_, content := q.getLayout()
		if m.X < content.GetWidth("table") {
			q.focus = focusTable
		} else {
			q.focus = focusDetails
			m.X -= content.GetWidth("table")
			msg = m
		}
	case tea.MouseWheelMsg:
		_, content := q.getLayout()
		if m.X < content.GetWidth("table") {
			q.focus = focusTable
		} else {
			q.focus = focusDetails
			m.X -= content.GetWidth("table")
		}
		msg = m
	}
	q.focusPanels()
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseMsg, tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		if q.focus == focusTable {
			_, cmd := q.table.Update(msg)
			return q, cmd
		}
		return q, q.updateDetails(msg)
	default:
		_, a := q.table.Update(msg)
		b := q.updateDetails(msg)
		return q, tea.Batch(a, b, q.detailsLoader.Update(msg))
	}
}
func (q *QueueTab) View() tea.View {
	root, content := q.getLayout()
	views := map[string]tea.View{"table": q.table.View()}
	if content.GetWidth("queue_details") > 0 {
		panel := q.detailsView()
		views["queue_details"] = q.subtabs.Decorate(panel, q.theme)
	}
	help := q.theme.RenderHelp(q.helpBindings())
	sections := map[string]tea.View{
		"content": content.JoinViews(views),
		"help":    tea.NewView(lipgloss.PlaceHorizontal(q.width, lipgloss.Left, ansi.Truncate(help, q.width, ""))),
	}
	return root.JoinViews(sections)
}
func (q *QueueTab) getLayout() (root *core.Layout, content *core.Layout) {
	root = core.NewLayout(q.width, q.height, core.Vertical).
		AddRelative("content", 1).
		AddFixed("help", helpHeight)

	contentWidth := root.GetWidth("content")
	contentHeight := root.GetHeight("content")

	content = core.NewLayout(contentWidth, contentHeight, core.Horizontal)

	// Ensure the table has at least its natural minimum width.
	// We prefer a 1:2 ratio (table:queue_details) if space allows.
	tableMinWidth := q.table.TotalWidth()
	if contentWidth < tableMinWidth+minDetailsWidth {
		content.AddRelative("table", 1).
			AddFixed("queue_details", 0)
	} else if contentWidth/3 < tableMinWidth {
		content.AddFixed("table", tableMinWidth).
			AddRelative("queue_details", 1)
	} else {
		content.AddRelative("table", 1).
			AddRelative("queue_details", 2)
	}

	return root, content
}

func (q *QueueTab) filterHeight() int {
	if q.query.Name != "" || q.query.VHost != "" || q.query.UseRegex {
		return 1
	}
	return 0
}
func (q *QueueTab) filterLabel() string {
	if q.filterHeight() == 0 {
		return ""
	}
	return q.theme.LabelFocused.Render("◆ Filter active")
}
