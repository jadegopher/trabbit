package core

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/stretchr/testify/assert"
)

func TestNewShortcut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		help     string
		keys     []string
		wantKeys []string
		wantHelp string
	}{
		{
			name:     "single_key",
			help:     "quit",
			keys:     []string{KeyQ},
			wantKeys: []string{KeyQ},
			wantHelp: "q",
		},
		{
			name:     "multiple_keys_symbols",
			help:     "navigate",
			keys:     []string{KeyUp, KeyDown},
			wantKeys: []string{KeyUp, KeyDown},
			wantHelp: fmt.Sprintf("%s %s", symbolUp, symbolDown),
		},
		{
			name:     "sorting_and_deduplication",
			help:     "nav",
			keys:     []string{KeyDown, KeyUp, KeyDown},
			wantKeys: []string{KeyUp, KeyDown},
			wantHelp: fmt.Sprintf("%s %s", symbolUp, symbolDown),
		},
		{
			name:     "tab_symbol",
			help:     "next",
			keys:     []string{KeyTab},
			wantKeys: []string{KeyTab},
			wantHelp: fmt.Sprintf("%s", symbolTab),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()
				got := NewShortcut(tt.help, tt.keys...)
				assert.ElementsMatch(t, tt.wantKeys, got.Keys())
				h := got.Help()
				assert.Equal(t, tt.help, h.Desc)
				assert.Equal(t, tt.wantHelp, h.Key)
			},
		)
	}
}

func TestPrepareHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		shortcuts []key.Binding
		wantCount int
		wantHelp  []string
	}{
		{
			name: "merging_same_description",
			shortcuts: []key.Binding{
				NewShortcut("nav", KeyUp),
				NewShortcut("nav", KeyDown),
			},
			wantCount: 1,
			wantHelp:  []string{fmt.Sprintf("%s %s", symbolUp, symbolDown)},
		},
		{
			name: "multiple_descriptions",
			shortcuts: []key.Binding{
				NewShortcut("nav", KeyUp),
				NewShortcut("quit", KeyQ),
			},
			wantCount: 2,
			wantHelp:  []string{fmt.Sprintf("%s", symbolUp), "q"},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()
				got := prepareHelp(tt.shortcuts...)
				assert.Len(t, got, tt.wantCount)
				for i, h := range tt.wantHelp {
					assert.Equal(t, h, got[i].Help().Key)
				}
			},
		)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run(
		"Cancel", func(t *testing.T) {
			t.Parallel()
			b := Cancel()
			assert.Equal(t, []string{KeyEsc, KeyCtrlQ}, b.Keys())
			h := b.Help()
			assert.Equal(t, "cancel", h.Desc)
			assert.Equal(t, fmt.Sprintf("%s %s", symbolEsc, GetSymbol(KeyCtrlQ)), h.Key)
		},
	)

	t.Run(
		"NavigateUp", func(t *testing.T) {
			t.Parallel()
			b := NavigateUp()
			// NewShortcut sorts keys: ShiftTab(11) < Up(20)
			assert.Equal(t, []string{KeyShiftTab, KeyUp}, b.Keys())
			h := b.Help()
			assert.Equal(t, "navigate", h.Desc)
			assert.Equal(t, fmt.Sprintf("%s %s", symbolShiftTab, symbolUp), h.Key)
		},
	)
}

func TestShortcutReferenceCollapsesCommandProtocolAliases(t *testing.T) {
	t.Parallel()
	binding := ShortcutReference(Redo())
	assert.Equal(t, "Ctrl+Shift+Z / Shift+Cmd+Z / Ctrl+Y", binding.Help().Key)
	assert.NotContains(t, binding.Help().Key, "meta")
	assert.NotContains(t, binding.Help().Key, "super")
	assert.Equal(t, Redo().Keys(), binding.Keys(), "readable help must preserve matching")
	assert.NotContains(t, Publish().Keys(), KeyAltEnter)
	assert.NotContains(t, Publish().Help().Key, "meta")
}
