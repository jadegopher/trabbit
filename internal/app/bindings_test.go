package app

import (
	"context"
	"testing"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestBindingValidationProtectsDefaultExchange(t *testing.T) {
	t.Parallel()

	service := New(nil)
	for _, source := range []string{"", " ", DefaultExchange} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			params := CreateBindingParams{VHost: "/", Queue: "orders", Source: source}
			require.Error(t, service.CreateQueueBinding(context.Background(), params))
			require.Error(t, service.DeleteQueueBinding(context.Background(), domain.QueueBinding{VHost: "/", Queue: "orders", Source: source, PropertiesKey: "key"}))
		})
	}
	require.Error(t, service.DeleteQueueBinding(context.Background(), domain.QueueBinding{VHost: "/", Queue: "orders", Source: "events"}))
	require.NoError(t, ValidateBinding(CreateBindingParams{VHost: "/", Queue: "orders", Source: "events"}), "routing keys may be empty")
	_, err := service.ListQueueBindings(context.Background(), "", "orders")
	require.Error(t, err)
}
