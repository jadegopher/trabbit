package widgets

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type ActionCreateQueue struct {
	Params app.CreateQueueParams
}

// todo: autocompletion for headers
func NewModalAddQueue(theme core.Theme, vhosts []string) *components.FormModal {
	ti := components.NewTextInput(theme)
	ti.SetWidth(60)
	vhostSelect := components.NewSelect(theme, vhosts, 0)
	typeSelect := components.NewSelect(theme, []string{"default", "classic", "quorum", "stream"}, 0)
	durable := components.NewToggle(theme, true)
	args := components.NewKeyValues(theme, "Arguments:", "  Key:", "  Value:")

	fields := []components.FormField{
		{Label: "Name: ", Item: ti},
		{Label: "VHost: ", Item: vhostSelect},
		{Label: "Type: ", Item: typeSelect},
		{Label: "Durable: ", Item: durable},
		{Label: "", Item: args},
	}

	m := components.NewFormModal(theme, "Add New Queue", fields)
	m.SetWidth(90)
	m.WrapNavigation = true
	m.ConfirmCancel = true
	m.OnSubmit = func() (tea.Msg, error) {
		arguments, err := args.Map()
		if err != nil {
			return nil, err
		}

		params := app.CreateQueueParams{
			Name:      ti.Value(),
			VHost:     vhostSelect.Value(),
			Durable:   durable.Value,
			Arguments: arguments,
		}

		qType := typeSelect.Value()
		if qType != "default" {
			params.Arguments["x-queue-type"] = qType
		}

		if err := app.ValidateQueue(params.VHost, params.Name); err != nil {
			return nil, err
		}
		return ActionCreateQueue{Params: params}, nil
	}

	return m
}
