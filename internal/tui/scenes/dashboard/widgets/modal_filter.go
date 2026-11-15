package widgets

import (
	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type ActionFilterQueues struct {
	Filter   string
	VHost    string
	UseRegex bool
}

func NewModalFilter(
	theme core.Theme,
	vhosts []string,
	initial ActionFilterQueues,
) *components.FormModal {
	ti := components.NewTextInput(theme)
	ti.SetValue(initial.Filter)

	vhostOptions := append([]string{""}, vhosts...)
	vhostIdx := 0
	for i, v := range vhostOptions {
		if v == initial.VHost {
			vhostIdx = i
			break
		}
	}

	vhostSelect := components.NewSelect(theme, vhostOptions, vhostIdx)
	regexToggle := components.NewToggle(theme, initial.UseRegex)

	fields := []components.FormField{
		{Label: "Filter: ", Item: ti},
		{Label: "VHost: ", Item: vhostSelect},
		{Label: "Regex: ", Item: regexToggle},
	}

	m := components.NewFormModal(theme, "Filter Queues", fields)
	m.CloseOnSubmit = true
	m.Actions = []components.FormAction{{
		Binding: core.ResetFilter(),
		Run: func() tea.Cmd {
			return core.Cmd(core.DialogResult{Dialog: m, Value: ActionFilterQueues{}, Close: true})
		},
	}}
	m.OnSubmit = func() (tea.Msg, error) {
		return ActionFilterQueues{Filter: ti.Value(), VHost: vhostSelect.Value(), UseRegex: regexToggle.Value}, nil
	}

	return m
}
