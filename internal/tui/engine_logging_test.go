package tui

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func TestEngineLogsLifecycleWithoutMessageContents(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	e := NewEngineWithConnector(nil, settings.NewMemory())
	e.logger = slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	defer e.Close()

	screen := &savedScene{}
	e.Update(core.OpenScreen{Model: screen})
	e.open(components.NewError(errors.New("secret-dialog-content"), e.theme), nil)
	e.closeDialog()
	e.Update(core.CloseScreen{Model: screen})
	e.Update(scopedMsg{session: e.session + 1, value: tea.PasteMsg{Content: "secret-payload"}})
	e.Update(sceneMsg{owner: screen, value: tea.PasteMsg{Content: "secret-payload"}})

	for _, message := range []string{
		"Opening screen", "Returning to screen", "Opening dialog", "Closing dialog",
		"Ignoring stale command result", "Ignoring result from closed scene",
	} {
		require.Contains(t, output.String(), message)
	}
	require.Contains(t, output.String(), "component=engine")
	require.NotContains(t, output.String(), "secret-dialog-content")
	require.NotContains(t, output.String(), "secret-payload")
}
