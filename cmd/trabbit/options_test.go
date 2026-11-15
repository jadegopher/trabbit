package main

import (
	"bytes"
	"flag"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/client"
	"github.com/stretchr/testify/require"
)

func TestConnectionFlags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []string
		want *app.Connection
	}{
		{name: "interactive"},
		{
			name: "embedded credentials",
			args: []string{"--url", "http://alice:secret@localhost:15672"},
			want: &app.Connection{URL: "http://localhost:15672", Username: "alice", Password: "secret"},
		},
		{
			name: "separate credentials",
			args: []string{"--url", "https://rabbit.example/proxy/", "--user", "alice", "--pass", "secret"},
			want: &app.Connection{URL: "https://rabbit.example/proxy/", Username: "alice", Password: "secret"},
		},
		{
			name: "escaped credentials",
			args: []string{"--url", "https://al%40ice:p%40ss%3Aword%2F%25@rabbit.example"},
			want: &app.Connection{URL: "https://rabbit.example", Username: "al@ice", Password: "p@ss:word/%"},
		},
		{
			name: "explicit overrides",
			args: []string{"--url", "http://alice:secret@localhost", "--user", "bob", "--pass", "new"},
			want: &app.Connection{URL: "http://localhost", Username: "bob", Password: "new"},
		},
		{
			name: "explicit empty password overrides",
			args: []string{"--url", "http://alice:secret@localhost", "--pass="},
			want: &app.Connection{URL: "http://localhost", Username: "alice"},
		},
		{
			name: "user override retains embedded password",
			args: []string{"--url", "http://alice:secret@localhost", "--user", "bob"},
			want: &app.Connection{URL: "http://localhost", Username: "bob", Password: "secret"},
		},
		{
			name: "password override retains embedded user",
			args: []string{"--url", "http://alice:secret@localhost", "--pass", "new"},
			want: &app.Connection{URL: "http://localhost", Username: "alice", Password: "new"},
		},
		{
			name: "empty password allowed",
			args: []string{"--url", "http://localhost", "--user", "alice"},
			want: &app.Connection{URL: "http://localhost", Username: "alice"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			options, err := parseOptions(tc.args, io.Discard)
			require.NoError(t, err)
			require.Equal(t, tc.want, options.connection)
			require.Equal(t, client.DefaultRequestDelay, options.throttle)
		})
	}
}

func TestInvalidConnectionFlags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []string
		err  string
	}{
		{"user without URL", []string{"--user", "alice"}, "--url is required"},
		{"password without URL", []string{"--pass", "secret"}, "--url is required"},
		{"empty URL", []string{"--url="}, "--url is required"},
		{"missing user", []string{"--url", "http://localhost"}, "username is required"},
		{"explicit empty user", []string{"--url", "http://alice:secret@localhost", "--user="}, "username is required"},
		{"AMQP URL", []string{"--url", "amqp://alice:secret@localhost"}, "http or https"},
		{"missing host", []string{"--url", "http:///path", "--user", "alice"}, "http or https"},
		{"malformed password", []string{"--url", "http://alice:secret%zz@localhost"}, "http or https"},
		{"malformed host", []string{"--url", "http://alice:secret@host:invalid"}, "http or https"},
		{"positional URL", []string{"http://alice:secret@localhost"}, "unexpected positional arguments"},
		{"negative throttle", []string{"--throttle=-1s"}, "throttle must not be negative"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			_, err := parseOptions(tc.args, &output)
			require.ErrorContains(t, err, tc.err)
			require.NotContains(t, err.Error(), "secret")
			require.NotContains(t, output.String(), "secret")
		})
	}
}

func TestStartupHelpVersionAndThrottle(t *testing.T) {
	t.Parallel()

	var help bytes.Buffer
	_, err := parseOptions([]string{"--help"}, &help)
	require.ErrorIs(t, err, flag.ErrHelp)
	for _, name := range []string{"-url", "-user", "-pass", "-version", "-throttle", "-log-level"} {
		require.Contains(t, help.String(), name)
	}
	require.Contains(t, help.String(), "(default warn)")

	options, err := parseOptions([]string{"--version", "--url=invalid"}, io.Discard)
	require.NoError(t, err)
	require.True(t, options.showVersion)
	require.Nil(t, options.connection)

	options, err = parseOptions([]string{"--throttle=500ms"}, io.Discard)
	require.NoError(t, err)
	require.Equal(t, 500*time.Millisecond, options.throttle)
	require.Nil(t, options.connection)
}

func TestLogLevelFlags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []string
		want slog.Level
	}{
		{name: "default", want: slog.LevelWarn},
		{name: "debug", args: []string{"--log-level=debug"}, want: slog.LevelDebug},
		{name: "info", args: []string{"--log-level=info"}, want: slog.LevelInfo},
		{name: "warn", args: []string{"--log-level=warn"}, want: slog.LevelWarn},
		{name: "error", args: []string{"--log-level=error"}, want: slog.LevelError},
		{name: "uppercase", args: []string{"--log-level=DEBUG"}, want: slog.LevelDebug},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options, err := parseOptions(tc.args, io.Discard)
			require.NoError(t, err)
			require.Equal(t, tc.want, options.logLevel)
			require.Nil(t, options.connection)
		})
	}

	for _, level := range []string{"trace", "", "info+1"} {
		t.Run("invalid "+level, func(t *testing.T) {
			t.Parallel()

			_, err := parseOptions([]string{"--log-level=" + level}, io.Discard)
			require.ErrorContains(t, err, "log level must be debug, info, warn or error")
		})
	}
}
