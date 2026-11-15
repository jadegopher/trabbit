package main

import (
	"log/slog"
	"os"
)

const (
	logFileName        = "debug.log"
	logFilePermissions = 0600
)

// Keep file ownership in startup so the logger closes after the program exits.
func newFileLogger(path string, level slog.Level) (*slog.Logger, *os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, logFilePermissions)
	if err != nil {
		return nil, nil, err
	}
	return slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: level})), file, nil
}
