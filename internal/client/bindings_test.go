package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestQueueBindingsTransport(t *testing.T) {
	t.Parallel()

	c := testClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/queues/%2Fteam/orders%2Fincoming/bindings", r.URL.EscapedPath())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[
			{"source":"","routing_key":"orders/incoming","properties_key":"default","arguments":{}},
			{"source":"events","routing_key":"order.*","properties_key":"order.*~hash","arguments":{"x-priority":2}}
		]`))}, nil
	})
	bindings, err := app.New(c).ListQueueBindings(context.Background(), "/team", "orders/incoming")
	require.NoError(t, err)
	require.Len(t, bindings, 2)
	require.True(t, bindings[0].IsDefault())
	require.Equal(t, domain.QueueBinding{VHost: "/team", Queue: "orders/incoming", Source: "events", RoutingKey: "order.*", PropertiesKey: "order.*~hash", Arguments: map[string]interface{}{"x-priority": float64(2)}}, bindings[1])
}

func TestCreateBindingPreservesArgumentsAndEmptyRoutingKey(t *testing.T) {
	t.Parallel()

	c := testClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/bindings/%2Fteam/e/events%2Fnew/q/orders%20in", r.URL.EscapedPath())
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "", body["routing_key"])
		require.Equal(t, map[string]interface{}{"number": float64(2), "boolean": true, "string": "2", "nested": map[string]interface{}{"a": "b"}}, body["arguments"])
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	err := app.New(c).CreateQueueBinding(context.Background(), app.CreateBindingParams{
		VHost: "/team", Queue: "orders in", Source: "events/new",
		Arguments: map[string]interface{}{"number": 2, "boolean": true, "string": "2", "nested": map[string]interface{}{"a": "b"}},
	})
	require.NoError(t, err)
}

func TestUnbindUsesReturnedPropertiesKeyAndCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := testClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/api/bindings/%2F/e/source%2Fname/q/orders/key%2F%25~hash", r.URL.EscapedPath())
		require.ErrorIs(t, r.Context().Err(), context.Canceled)
		return nil, r.Context().Err()
	})
	err := app.New(c).DeleteQueueBinding(ctx, domain.QueueBinding{VHost: "/", Queue: "orders", Source: "source/name", PropertiesKey: "key/%~hash"})
	require.ErrorIs(t, err, context.Canceled)
	var operation *app.OperationError
	require.True(t, errors.As(err, &operation))
	require.Equal(t, "unbind queue", operation.Operation)
}

func TestQueueConsumerAndRuntimeFields(t *testing.T) {
	t.Parallel()

	c := testClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "60", r.URL.Query().Get("data_rates_age"))
		require.Equal(t, "5", r.URL.Query().Get("data_rates_incr"))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
			"name":"orders","vhost":"/",
			"consumer_details":[{"channel_details":{"name":"172.19.0.1:51754 (4)"},"consumer_tag":"consumer-tag",
				"ack_required":true,"exclusive":false,"prefetch_count":10000,"active":false,"activity_status":"up",
				"consumer_timeout":1800000,"arguments":{"x-priority":2}}],
			"reductions_details":{"rate":288181,"samples":[{"timestamp":10000,"sample":5000},{"timestamp":5000,"sample":0}]},
			"garbage_collection":{"min_bin_vheap_size":46422,"min_heap_size":233,"fullsweep_after":65535,"minor_gcs":0}
		}`))}, nil
	})
	data, err := c.GetQueueDetailedStats(context.Background(), "/", "orders", time.Minute)
	require.NoError(t, err)
	require.Len(t, data.ConsumerDetails, 1)
	consumer := data.ConsumerDetails[0]
	require.Equal(t, "172.19.0.1:51754 (4)", consumer.Channel)
	require.True(t, consumer.AckRequired)
	require.NotNil(t, consumer.Active)
	require.False(t, *consumer.Active)
	require.Equal(t, int64(1800000), *consumer.TimeoutMillis)
	require.Equal(t, float64(288181), data.Runtime.Reductions.Rate)
	require.Len(t, data.Runtime.Reductions.Samples, 2)
	require.Equal(t, int64(0), *data.Runtime.MinorGCs, "zero is present, not unavailable")
}
