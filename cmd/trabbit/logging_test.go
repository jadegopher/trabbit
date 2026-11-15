package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	for _, minimum := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		t.Run(minimum.String(), func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), logFileName)
			require.NoError(t, os.WriteFile(path, []byte("previous run"), logFilePermissions))
			logger, file, err := newFileLogger(path, minimum)
			require.NoError(t, err)
			t.Cleanup(func() { file.Close() })

			logger.Debug("debug record")
			logger.Info("info record")
			logger.Warn("warn record")
			logger.Error("error record")
			require.NoError(t, file.Close())

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NotContains(t, string(content), "previous run")
			for _, record := range []struct {
				level slog.Level
				text  string
			}{
				{slog.LevelDebug, "debug record"},
				{slog.LevelInfo, "info record"},
				{slog.LevelWarn, "warn record"},
				{slog.LevelError, "error record"},
			} {
				if record.level >= minimum {
					require.Contains(t, string(content), record.text)
				} else {
					require.NotContains(t, string(content), record.text)
				}
			}
		})
	}
}

func TestFileLoggerOpenFailure(t *testing.T) {
	t.Parallel()

	logger, file, err := newFileLogger(t.TempDir(), slog.LevelInfo)
	require.Error(t, err)
	require.Nil(t, logger)
	require.Nil(t, file)
}
