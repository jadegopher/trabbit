package mocks

import (
	"context"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
)

func (m *MockQueueClient) ListQueueBindings(ctx context.Context, vhost, queue string) ([]domain.QueueBinding, error) {
	result := m.Called(ctx, vhost, queue)
	bindings, _ := result.Get(0).([]domain.QueueBinding)
	return bindings, result.Error(1)
}

func (m *MockQueueClient) CreateQueueBinding(ctx context.Context, p app.CreateBindingParams) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockQueueClient) DeleteQueueBinding(ctx context.Context, b domain.QueueBinding) error {
	return m.Called(ctx, b).Error(0)
}
