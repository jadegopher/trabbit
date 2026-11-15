package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/stretchr/testify/require"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testClient(fn roundTrip) *Client {
	return &Client{endpoint: "http://rabbit", username: "guest", password: "secret", http: &http.Client{Transport: fn}}
}

func TestRequestsCarryContextAndEscapeResourceNames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := testClient(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "/api/queues/%2F/name%2Fwith%20space", r.URL.EscapedPath())
		require.ErrorIs(t, r.Context().Err(), context.Canceled)
		return nil, r.Context().Err()
	})
	require.ErrorIs(t, c.DeleteQueue(ctx, "/", "name/with space"), context.Canceled)
}

func TestMissingRateDetailsAreSafe(t *testing.T) {
	t.Parallel()
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{\"name\":\"empty\",\"messages\":0}"))}, nil
	})
	stats, err := c.GetQueueDetailedStats(context.Background(), "/", "empty", time.Minute)
	require.NoError(t, err)
	require.Empty(t, stats.MessagesReady)
	require.Empty(t, stats.MessagesTotal)
}

func TestQueueDetailsPreserveIdentityAndThroughputHistory(t *testing.T) {
	t.Parallel()

	c := testClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
			"name":"orders", "vhost":"/", "type":"quorum", "state":"running",
			"message_stats": {
				"publish":100,
				"publish_details":{"rate":2,"samples":[{"timestamp":70000,"sample":100},{"timestamp":65000,"sample":90}]},
				"deliver_get_details":{"rate":1,"samples":[{"timestamp":70000,"sample":50},{"timestamp":65000,"sample":45}]},
				"ack_details":{"rate":1,"samples":[{"timestamp":70000,"sample":50},{"timestamp":65000,"sample":45}]}
			}
		}`))}, nil
	})
	stats, err := c.GetQueueDetailedStats(context.Background(), "/", "orders", time.Minute)
	require.NoError(t, err)
	require.Equal(t, "orders", stats.Name)
	require.Equal(t, "/", stats.VHost)
	require.Equal(t, "quorum", stats.Type)
	require.Equal(t, "running", stats.State)
	require.Equal(t, float64(2), stats.MessageStats.PublishDetails.Rate)
	require.Len(t, stats.MessageStats.PublishDetails.Samples, 2)
	require.Equal(t, int64(70000), stats.MessageStats.PublishDetails.Samples[0].Timestamp)
	require.Equal(t, 100, stats.MessageStats.PublishDetails.Samples[0].Value)
	require.Len(t, stats.MessageStats.DeliverGetDetails.Samples, 2)
	require.Len(t, stats.MessageStats.AckDetails.Samples, 2)
}

func TestPublishDoesNotMutateCallerProperties(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{\"routed\":false}"))}, nil
	})
	properties := map[string]interface{}{"content_type": "text/plain"}
	result, err := c.Publish(context.Background(), "/", DefaultExchange, PublishOptions{Properties: PublishMessageProperties{Properties: properties, Headers: map[string]interface{}{"x": true}}})
	require.NoError(t, err)
	require.False(t, result.Routed)
	require.NotContains(t, properties, "headers")
	_, err = c.Publish(context.Background(), "/", DefaultExchange, PublishOptions{})
	require.NoError(t, err, "nil maps must be supported")
}

func TestApplicationPreservesTransportErrorContext(t *testing.T) {
	c := testClient(func(r *http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
	err := app.New(c).PurgeQueue(context.Background(), "/", "orders")
	var operation *app.OperationError
	require.True(t, errors.As(err, &operation))
	require.Equal(t, "purge queue", operation.Operation)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestQueueHistoryRangeReachesBroker(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		seconds  string
		interval string
	}{
		{seconds: "60", interval: "5"},
		{seconds: "600", interval: "5"},
		{seconds: "1800", interval: "5"},
		{seconds: "3600", interval: "60"},
	} {
		t.Run(tc.seconds, func(t *testing.T) {
			t.Parallel()

			history, err := time.ParseDuration(tc.seconds + "s")
			require.NoError(t, err)
			c := testClient(func(r *http.Request) (*http.Response, error) {
				for _, prefix := range []string{"lengths", "msg_rates", "data_rates"} {
					require.Equal(t, tc.seconds, r.URL.Query().Get(prefix+"_age"), prefix)
					require.Equal(t, tc.interval, r.URL.Query().Get(prefix+"_incr"), prefix)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})
			_, err = app.New(c).GetQueueDetailedStats(context.Background(), "/", "orders", history)
			require.NoError(t, err)
		})
	}
}
