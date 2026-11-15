package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestOperationFailuresAreLoggedOnceByService(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		operation string
		call      func(*app.Service) error
	}{
		{"list queues", func(s *app.Service) error {
			_, err := s.ListQueues(context.Background(), app.ListParameters{Page: 1, PageSize: 10, VHost: "/", Name: "secret-filter"})
			return err
		}},
		{"list vhosts", func(s *app.Service) error {
			_, err := s.ListVHosts(context.Background())
			return err
		}},
		{"queue details", func(s *app.Service) error {
			_, err := s.GetQueueDetailedStats(context.Background(), "/", "orders", time.Minute)
			return err
		}},
		{"create queue", func(s *app.Service) error {
			return s.CreateQueue(context.Background(), app.CreateQueueParams{VHost: "/", Name: "orders", Arguments: map[string]interface{}{"token": "secret-argument"}})
		}},
		{"delete queue", func(s *app.Service) error {
			return s.DeleteQueue(context.Background(), "/", "orders")
		}},
		{"purge queue", func(s *app.Service) error {
			return s.PurgeQueue(context.Background(), "/", "orders")
		}},
		{"publish", func(s *app.Service) error {
			_, err := s.Publish(context.Background(), "/", app.DefaultExchange, app.PublishOptions{
				Payload: "secret-payload", PayloadEncoding: "string",
				Properties: app.PublishMessageProperties{Headers: map[string]interface{}{"token": "secret-header"}},
			})
			return err
		}},
		{"list queue bindings", func(s *app.Service) error {
			_, err := s.ListQueueBindings(context.Background(), "/", "orders")
			return err
		}},
		{"bind queue", func(s *app.Service) error {
			return s.CreateQueueBinding(context.Background(), app.CreateBindingParams{
				VHost: "/", Queue: "orders", Source: "events", Arguments: map[string]interface{}{"token": "secret-argument"},
			})
		}},
		{"unbind queue", func(s *app.Service) error {
			return s.DeleteQueueBinding(context.Background(), domain.QueueBinding{VHost: "/", Queue: "orders", Source: "events", PropertiesKey: "key"})
		}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
			c := testClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("secret-response"))}, nil
			})
			c.logger = logger
			err := tc.call(app.NewWithLogger(c, logger))
			var operationError *app.OperationError
			require.ErrorAs(t, err, &operationError)
			require.Equal(t, tc.operation, operationError.Operation)
			var httpError *HTTPError
			require.ErrorAs(t, err, &httpError)
			require.Equal(t, http.StatusForbidden, httpError.StatusCode)

			logs := output.String()
			require.Equal(t, 1, strings.Count(logs, `"level":"ERROR"`), logs)
			require.Equal(t, 1, strings.Count(logs, `"level":"DEBUG"`), logs)
			require.Contains(t, logs, `"component":"app"`)
			require.Contains(t, logs, `"operation":"`+tc.operation+`"`)
			require.Contains(t, logs, `"component":"client"`)
			require.Contains(t, logs, `"status":403`)
			require.Contains(t, logs, `"duration":`)
			require.NotContains(t, logs, "secret-filter")
			require.NotContains(t, logs, "secret-payload")
			require.NotContains(t, logs, "secret-header")
			require.NotContains(t, logs, "secret-argument")
			require.NotContains(t, logs, "secret-response")
			require.NotContains(t, logs, c.password)
		})
	}
}

func TestCanceledOperationOnlyLogsAtDebug(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c := testClient(func(r *http.Request) (*http.Response, error) {
		t.Error("canceled request must not reach transport")
		return nil, context.Canceled
	})
	c.logger = logger
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := app.NewWithLogger(c, logger).DeleteQueue(ctx, "/", "orders")
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, output.String(), "level=ERROR")
	require.Equal(t, 2, strings.Count(output.String(), "level=DEBUG"))
	require.Contains(t, output.String(), "context canceled")
}

func TestTransportFailureOmitsURLCredentialsAndQuery(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c := testClient(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	c.endpoint = "http://alice:secret-password@rabbit"
	c.logger = logger
	_, err := app.NewWithLogger(c, logger).ListQueues(context.Background(), app.ListParameters{Page: 1, PageSize: 10, Name: "secret-filter"})
	require.Error(t, err)
	require.Contains(t, output.String(), "connection refused")
	require.Contains(t, output.String(), "path=/api/queues")
	for _, value := range []string{"alice", "secret-password", "secret-filter", "?"} {
		require.NotContains(t, output.String(), value)
	}
}

func TestTransportDiagnosticsCoverLocalFailures(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		body any
		json string
	}{
		{name: "marshal", body: make(chan int)},
		{name: "decode", json: "invalid JSON"},
		{name: "success", json: "{}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			c := testClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tc.json))}, nil
			})
			c.logger = slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
			var result any
			err := c.request(context.Background(), http.MethodPost, "/api/test?token=secret-query", tc.body, &result)
			if tc.name == "success" {
				require.NoError(t, err)
				require.Contains(t, output.String(), "failed=false")
			} else {
				require.Error(t, err)
				require.Contains(t, output.String(), "failed=true")
			}
			require.Equal(t, 1, strings.Count(output.String(), "level=DEBUG"))
			require.NotContains(t, output.String(), "secret-query")
		})
	}
}
