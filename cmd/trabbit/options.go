package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/client"
)

const defaultLogLevel = slog.LevelWarn

type startupOptions struct {
	connection  *app.Connection
	throttle    time.Duration
	logLevel    slog.Level
	showVersion bool
}

func parseOptions(args []string, output io.Writer) (startupOptions, error) {
	var options startupOptions
	options.logLevel = defaultLogLevel
	var connection app.Connection
	flags := flag.NewFlagSet("trabbit", flag.ContinueOnError)
	flags.SetOutput(output)
	logHelp := fmt.Sprintf("Minimum level written to %s: debug, info, warn, error (default %s)", logFileName, strings.ToLower(defaultLogLevel.String()))
	flags.Func("log-level", logHelp, func(value string) error {
		switch strings.ToLower(value) {
		case "debug":
			options.logLevel = slog.LevelDebug
		case "info":
			options.logLevel = slog.LevelInfo
		case "warn":
			options.logLevel = slog.LevelWarn
		case "error":
			options.logLevel = slog.LevelError
		default:
			return fmt.Errorf("log level must be debug, info, warn or error")
		}
		return nil
	})
	flags.BoolVar(&options.showVersion, "version", false, "Print the application version and exit")
	flags.DurationVar(&options.throttle, "throttle", client.DefaultRequestDelay, "Delay every RabbitMQ request by this duration (e.g. 1s or 500ms; 0 disables)")
	flags.StringVar(&connection.URL, "url", "", "RabbitMQ management HTTP(S) URL; may include user:password@host; connect automatically")
	flags.StringVar(&connection.Username, "user", "", "Username (overrides credentials in --url)")
	flags.StringVar(&connection.Password, "pass", "", "Password (overrides credentials in --url)")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if options.showVersion {
		return options, nil
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf("unexpected positional arguments; use --url, --user and --pass")
	}
	if options.throttle < 0 {
		return options, fmt.Errorf("throttle must not be negative")
	}

	provided := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	if !provided["url"] && !provided["user"] && !provided["pass"] {
		return options, nil
	}
	if connection.URL == "" {
		return options, fmt.Errorf("--url is required when supplying connection flags")
	}

	endpoint, err := url.Parse(connection.URL)
	if err != nil {
		// URL parse errors include the input, which may contain a password.
		return options, fmt.Errorf("URL must be an http or https endpoint")
	}
	if endpoint.User != nil {
		if !provided["user"] {
			connection.Username = endpoint.User.Username()
		}
		if !provided["pass"] {
			connection.Password, _ = endpoint.User.Password()
		}
		endpoint.User = nil
	}
	connection.URL = endpoint.String()
	if err := app.ValidateConnection(connection); err != nil {
		return options, err
	}
	options.connection = &connection
	return options, nil
}
