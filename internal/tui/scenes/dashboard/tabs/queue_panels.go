package tabs

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/dashboard/widgets"
)

const (
	queueOverview = iota
	queueConsumers
	queueBindings
	queueRuntime
)

type bindingsResult struct {
	id          int
	vhost, name string
	data        []domain.QueueBinding
	err         error
}

type bindingOperationResult struct {
	id           int
	dialog       tea.Model
	vhost, queue string
	err          error
}

func (q *QueueTab) prepareDetails(vhost, name string) {
	q.charts.SetLoading(vhost, name)
	q.consumers.SetLoading(vhost, name)
	q.queueBindings.SetLoading(vhost, name)
	q.runtime.SetLoading(vhost, name)
}

func (q *QueueTab) clearDetails() {
	q.detailsLoader.Stop()
	q.charts.Clear()
	q.consumers.Clear()
	q.queueBindings.Clear()
	q.runtime.Clear()
}

func (q *QueueTab) focusPanels() {
	q.table.SetFocused(q.focus == focusTable)
	details := q.focus == focusDetails
	q.charts.SetFocused(details && q.subtabs.Active == queueOverview)
	q.consumers.SetFocused(details && q.subtabs.Active == queueConsumers)
	q.queueBindings.SetFocused(details && q.subtabs.Active == queueBindings)
	q.runtime.SetFocused(details && q.subtabs.Active == queueRuntime)
}

func (q *QueueTab) updateDetails(msg tea.Msg) tea.Cmd {
	switch q.subtabs.Active {
	case queueConsumers:
		return q.consumers.Update(msg)
	case queueBindings:
		return q.queueBindings.Update(msg)
	case queueRuntime:
		return q.runtime.Update(msg)
	default:
		_, cmd := q.charts.Update(msg)
		return cmd
	}
}

func (q *QueueTab) detailsView() tea.View {
	if q.detailsLoader.IsActive() {
		content := q.detailsLoadingPanel.Center(q.detailsLoader.View())
		return tea.NewView(q.detailsLoadingPanel.Render(content, 0, q.focus == focusDetails))
	}

	switch q.subtabs.Active {
	case queueConsumers:
		return q.consumers.View()
	case queueBindings:
		return q.queueBindings.View()
	case queueRuntime:
		return q.runtime.View()
	default:
		return q.charts.View()
	}
}

func (q *QueueTab) detailError(err error) {
	switch q.subtabs.Active {
	case queueConsumers:
		q.consumers.SetError(err)
	case queueBindings:
		q.queueBindings.SetError(err)
	case queueRuntime:
		q.runtime.SetError(err)
	}
}

func (q *QueueTab) detailActions(msg tea.KeyPressMsg) tea.Cmd {
	active := q.table.ActiveQueue()
	if active == nil {
		return nil
	}
	if key.Matches(msg, core.AddBinding()) {
		return core.Cmd(core.OpenDialog{Model: widgets.NewModalAddBinding(q.theme, active.VHost, active.Name)})
	}
	if q.focus != focusDetails || q.subtabs.Active != queueBindings {
		return nil
	}
	if key.Matches(msg, core.Unbind()) {
		binding := q.queueBindings.ActiveBinding()
		if binding == nil || binding.VHost != active.VHost || binding.Queue != active.Name {
			return nil
		}
		message := fmt.Sprintf("Unbind exchange %s from queue %s in vhost %s?\nRouting key: %q\nArguments: %v", binding.Source, binding.Queue, binding.VHost, binding.RoutingKey, binding.Arguments)
		return core.Cmd(core.OpenDialog{Model: components.NewConfirm(message, q.theme).WithResult(widgets.ActionUnbind{Binding: *binding})})
	}
	return nil
}

// Binding commands capture their target and dialog. Results cannot act on a
// different queue or close a replacement dialog, and duplicate submits are inert.
func (q *QueueTab) bindingOperation(dialog tea.Model, vhost, queue string, run func() error) tea.Cmd {
	if q.bindingBusy {
		return nil
	}
	q.bindingBusy = true
	q.bindingOperationID++
	id := q.bindingOperationID
	_, form := dialog.(*components.FormModal)
	if !form {
		dialog = components.NewLoader("Updating binding...", q.theme)
	}
	q.bindingDialog = dialog
	request := func() tea.Msg {
		return bindingOperationResult{id: id, dialog: dialog, vhost: vhost, queue: queue, err: run()}
	}
	if form {
		return request
	}
	return core.Cmd(core.OpenDialog{Model: dialog, After: request})
}

func (q *QueueTab) helpBindings() core.Shortcuts {
	bindings := core.Shortcuts{q.bindings.Subtabs, core.AddBinding()}
	if q.subtabs.Active == queueBindings {
		bindings = append(bindings, core.Unbind())
	}
	for _, binding := range q.bindings.ShortHelp() {
		if binding.Help().Desc == q.bindings.Subtabs.Help().Desc {
			continue
		}
		if binding.Help().Desc == q.bindings.ViewSettings.Help().Desc && q.subtabs.Active == queueRuntime {
			continue
		}
		bindings = append(bindings, binding)
	}
	return bindings
}

var queueSubtabIDs = []string{
	settings.QueueSubtabOverview,
	settings.QueueSubtabConsumers,
	settings.QueueSubtabBindings,
	settings.QueueSubtabRuntime,
}

type subtabSettingsSaved struct {
	id       int
	selected string
	err      error
}

type detailPaginationSaved struct {
	id     int
	subtab string
	change components.ActionPageChange
	dialog tea.Model
	err    error
}

// Detail pages use local snapshots. Save size changes before applying them;
// capture the list owner so a delayed result cannot navigate another queue.
func (q *QueueTab) changeDetailPage(change components.ActionPageChange, dialog tea.Model) tea.Cmd {
	var subtab string
	var currentSize int
	config := q.settings.Snapshot().Queues
	switch change.Owner {
	case q.consumers.PageOwner():
		subtab, currentSize = settings.QueueSubtabConsumers, config.Consumers.PageSize
	case q.queueBindings.PageOwner():
		subtab, currentSize = settings.QueueSubtabBindings, config.Bindings.PageSize
	default:
		return closePaginationDialog(dialog)
	}
	if _, saving := q.detailPaginationSaves[subtab]; saving {
		return nil
	}
	if change.PageSize == 0 || change.PageSize == currentSize {
		q.applyDetailPage(subtab, change)
		return closePaginationDialog(dialog)
	}

	q.detailPaginationSaveID++
	id := q.detailPaginationSaveID
	q.detailPaginationSaves[subtab] = id
	preferences := q.settings
	return func() tea.Msg {
		err := preferences.Update(func(config *settings.Config) {
			switch subtab {
			case settings.QueueSubtabConsumers:
				config.Queues.Consumers.PageSize = change.PageSize
			case settings.QueueSubtabBindings:
				config.Queues.Bindings.PageSize = change.PageSize
			}
		})
		return detailPaginationSaved{id: id, subtab: subtab, change: change, dialog: dialog, err: err}
	}
}

func (q *QueueTab) finishDetailPagination(result detailPaginationSaved) tea.Cmd {
	id, saving := q.detailPaginationSaves[result.subtab]
	if !saving || id != result.id {
		return nil
	}
	delete(q.detailPaginationSaves, result.subtab)
	dialog := result.dialog
	if result.err != nil {
		if dialog != nil {
			return core.Cmd(core.DialogFeedback{Dialog: dialog, Err: result.err})
		}
		return core.Cmd(core.OpenDialog{Model: components.NewError(result.err, q.theme)})
	}

	switch result.subtab {
	case settings.QueueSubtabConsumers:
		q.consumers.SetPageSize(result.change.PageSize)
	case settings.QueueSubtabBindings:
		q.queueBindings.SetPageSize(result.change.PageSize)
	}
	q.applyDetailPage(result.subtab, result.change)
	return closePaginationDialog(dialog)
}

func (q *QueueTab) applyDetailPage(subtab string, change components.ActionPageChange) {
	switch subtab {
	case settings.QueueSubtabConsumers:
		q.consumers.Update(change)
	case settings.QueueSubtabBindings:
		q.queueBindings.Update(change)
	}
}

func closePaginationDialog(dialog tea.Model) tea.Cmd {
	if dialog == nil {
		return nil
	}
	return core.Cmd(core.DialogResult{Dialog: dialog, Close: true})
}

// Serialize subtab writes and save the latest choice after an in-flight write.
// Navigation remains immediate; command scheduling cannot reorder disk updates.
func (q *QueueTab) saveSubtab() tea.Cmd {
	if q.savingSubtab {
		return nil
	}
	q.savingSubtab = true
	q.subtabSaveID++
	id := q.subtabSaveID
	preferences, selected := q.settings, queueSubtabIDs[q.subtabs.Active]
	return func() tea.Msg {
		err := preferences.Update(func(config *settings.Config) { config.Queues.SelectedSubtab = selected })
		return subtabSettingsSaved{id: id, selected: selected, err: err}
	}
}
