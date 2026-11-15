package main

import (
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
)

// terminalOptions keeps terminal compatibility at startup, outside view models.
func terminalOptions(output io.Writer, environ []string) []tea.ProgramOption {
	compatible := rendererEnvironment(environ)
	if compatible == nil {
		return nil
	}

	// Detect colors using the real terminal, before selecting the renderer's
	// conservative capability set. Do not change the process environment.
	return []tea.ProgramOption{
		tea.WithEnvironment(compatible),
		tea.WithColorProfile(colorprofile.Detect(output, environ)),
	}
}

func rendererEnvironment(environ []string) []string {
	env := uv.Environ(environ)
	if env.Getenv("TERMINAL_EMULATOR") != "JetBrains-JediTerm" &&
		!strings.EqualFold(env.Getenv("TERM_PROGRAM"), "JetBrains-JediTerm") {
		return nil
	}

	// A multiplexer interprets our cursor commands itself.
	term := env.Getenv("TERM")
	if strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return nil
	}

	// JediTerm advertises xterm but ignores CBT (CSI Z). Ultraviolet assumes
	// that move succeeded, so later chart cells overwrite borders and survive
	// screen changes. Bubble Tea currently exposes no per-capability switch.
	// Its linux capability set retains absolute positioning and erasure while
	// excluding CBT, CHT and REP, which JediTerm does not implement.
	// https://github.com/JetBrains/jediterm/issues/328
	compatible := make([]string, 0, len(environ))
	for _, entry := range environ {
		if !strings.HasPrefix(entry, "TERM=") {
			compatible = append(compatible, entry)
		}
	}
	return append(compatible, "TERM=linux")
}
