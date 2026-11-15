package widgets

import (
	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const bindingModalWidthColumns = 90

type ActionCreateBinding struct{ Params app.CreateBindingParams }

type ActionUnbind struct{ Binding domain.QueueBinding }

func NewModalAddBinding(theme core.Theme, vhost, queue string) *components.FormModal {
	source := components.NewTextInput(theme)
	routing := components.NewTextInput(theme)
	arguments := components.NewKeyValues(theme, "Arguments:", "  Key:", "  Value:")
	modal := components.NewFormModal(theme, "Add binding to "+queue, []components.FormField{
		{Label: "From exchange: ", Item: source},
		{Label: "Routing key: ", Item: routing},
		{Item: arguments},
	})
	modal.SetWidth(bindingModalWidthColumns)
	modal.ConfirmCancel = true
	modal.WrapNavigation = true
	modal.OnSubmit = func() (tea.Msg, error) {
		args, err := arguments.Map()
		if err != nil {
			return nil, err
		}
		p := app.CreateBindingParams{VHost: vhost, Queue: queue, Source: source.Value(), RoutingKey: routing.Value(), Arguments: args}
		if err := app.ValidateBinding(p); err != nil {
			return nil, err
		}
		return ActionCreateBinding{Params: p}, nil
	}
	return modal
}
