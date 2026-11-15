package log

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailureContext(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		err     error
		minimum slog.Level
		level   string
	}{
		{"failure", errors.New("transport unavailable"), slog.LevelInfo, "ERROR"},
		{"deadline", context.DeadlineExceeded, slog.LevelInfo, "ERROR"},
		{"cancellation", fmt.Errorf("superseded: %w", context.Canceled), slog.LevelDebug, "DEBUG"},
		{"quiet cancellation", context.Canceled, slog.LevelInfo, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: tc.minimum}))
			FailureContext(logger, context.Background(), "operation failed", tc.err, "operation", "purge queue")
			if tc.level == "" {
				require.Empty(t, output.String())
				return
			}
			require.Contains(t, output.String(), "level="+tc.level)
			require.Contains(t, output.String(), tc.err.Error())
			require.Contains(t, output.String(), `operation="purge queue"`)
		})
	}
}

func TestFailureContextOmitsRequestURL(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	err := fmt.Errorf("connect: %w", &url.Error{
		Op: "Get", URL: "https://alice:password@rabbit/api?token=secret", Err: errors.New("connection refused"),
	})
	FailureContext(logger, context.Background(), "connection failed", err)

	require.Contains(t, output.String(), "connection refused")
	for _, value := range []string{"alice", "password", "token", "secret", "https://"} {
		require.NotContains(t, output.String(), value)
	}
}
