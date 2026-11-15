package app

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidationDoesNotCallBackend(t *testing.T) {
	s := New(nil)
	ctx := context.Background()
	require.Error(t, s.CreateQueue(ctx, CreateQueueParams{VHost: "/"}))
	require.Error(t, s.DeleteQueue(ctx, "", "queue"))
	require.Error(t, s.PurgeQueue(ctx, "/", ""))
	_, err := s.Publish(ctx, "/", DefaultExchange, PublishOptions{PayloadEncoding: "base64", Payload: "invalid!"})
	require.Error(t, err)
	_, err = s.ListQueues(ctx, ListParameters{Page: 0, PageSize: 10})
	require.Error(t, err)
	require.Error(t, ValidateConnection(Connection{URL: "not a URL", Username: "guest"}))
	require.NoError(t, ValidateConnection(Connection{URL: "https://localhost:15672/", Username: "guest"}))
}
