package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/client"
	"github.com/jadegopher/trabbit/internal/log"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui"
	"github.com/jadegopher/trabbit/internal/version"
)

func main() {
	options, err := parseOptions(os.Args[1:], os.Stderr)
	if err == flag.ErrHelp {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if options.showVersion {
		fmt.Println(version.Current)
		return
	}

	logger, logFile, err := newFileLogger(logFileName, options.logLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot open log file:", err)
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: options.logLevel}))
	} else {
		defer logFile.Close()
	}
	log.DefaultLogger = logger

	preferences, err := settings.OpenBesideExecutable()
	if err != nil {
		log.Error("Failed to load settings", log.Err, err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log.Info("Starting TUI app", "version", version.Current, "log_level", options.logLevel)
	connect := func(ctx context.Context, cfg app.Connection) (*app.Service, error) {
		return client.ConnectWithDelay(ctx, cfg, options.throttle)
	}
	engine := tui.NewEngineWithConnection(connect, preferences, options.connection)
	p := tea.NewProgram(engine, terminalOptions(os.Stdout, os.Environ())...)
	if _, err := p.Run(); err != nil {
		log.Error("TUI app failed", log.Err, err)
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	log.Info("TUI app stopped")
}
