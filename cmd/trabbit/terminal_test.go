package main

import (
	"bytes"
	"regexp"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/stretchr/testify/require"
)

func TestRendererEnvironment(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		env        []string
		compatible bool
	}{
		{"jediterm", []string{"TERM=xterm-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm", "COLORTERM=truecolor"}, true},
		{"term_program", []string{"TERM=xterm-256color", "TERM_PROGRAM=JetBrains-JediTerm"}, true},
		{"ghostty", []string{"TERM=xterm-ghostty", "TERM_PROGRAM=ghostty"}, false},
		{"tmux", []string{"TERM=tmux-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm"}, false},
		{"screen", []string{"TERM=screen-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original := append([]string(nil), tc.env...)
			got := rendererEnvironment(tc.env)
			require.Equal(t, original, tc.env)
			if !tc.compatible {
				require.Nil(t, got)
				return
			}
			require.Equal(t, "linux", uv.Environ(got).Getenv("TERM"))
			require.Equal(t, uv.Environ(tc.env).Getenv("COLORTERM"), uv.Environ(got).Getenv("COLORTERM"))
		})
	}
}

// Exercise the actual renderer used by Bubble Tea, with its Unix hard-tab
// optimization enabled. String geometry tests cannot catch cursor desync.
func TestJediTermCursorMovement(t *testing.T) {
	t.Parallel()

	const width, height = 208, 30
	const fromColumn, toColumn, row = 198, 192, 5
	env := []string{"TERM=xterm-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm"}
	move := func(environ []string) string {
		var output bytes.Buffer
		renderer := uv.NewTerminalRenderer(&output, environ)
		renderer.SetFullscreen(true)
		renderer.Resize(width, height)
		renderer.SetTabStops(width)
		renderer.SetBackspace(true)
		renderer.SetPosition(fromColumn, row)
		renderer.MoveTo(toColumn, row)
		require.NoError(t, renderer.Flush())
		return output.String()
	}

	// JediTerm ignores the first sequence, leaving its cursor at column 198
	// while the renderer believes it is at 192. Absolute positioning works.
	require.Equal(t, "\x1b[Z", move(env))
	require.Equal(t, "\x1b[6;193H", move(rendererEnvironment(env)))
}

func TestJediTermRefreshAndScreenChangeCommands(t *testing.T) {
	t.Parallel()

	const width, height, frames = 208, 30, 20
	const chartLeft, chartRight = 75, 202
	var output bytes.Buffer
	env := rendererEnvironment([]string{"TERM=xterm-256color", "TERMINAL_EMULATOR=JetBrains-JediTerm"})
	renderer := uv.NewTerminalRenderer(&output, env)
	renderer.SetFullscreen(true)
	renderer.SetScrollOptim(true)
	renderer.SetTabStops(width)
	renderer.SetBackspace(true)
	renderer.Resize(width, height)
	buffer := uv.NewRenderBuffer(width, height)
	unsupported := regexp.MustCompile("\x1b\\[[0-9;]*[ZIb]")
	for frame := range frames {
		for y := range height {
			for x := range width {
				content := " "
				if x == 0 || x == width-1 || x == chartLeft {
					content = "│"
				} else if frame < frames-1 && x > chartLeft && x < chartRight && (x+y+frame)%3 == 0 {
					content = "⣀"
				}
				buffer.SetCell(x, y, &uv.Cell{Content: content, Width: 1})
			}
		}
		renderer.Render(buffer)
		require.NoError(t, renderer.Flush())
		require.False(t, unsupported.Match(output.Bytes()), "unsupported cursor command in frame %d", frame)
		output.Reset()
	}
}
