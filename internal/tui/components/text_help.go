package components

import "github.com/jadegopher/trabbit/internal/tui/core"

// Shortcuts lists everyday payload editing controls from the live keymap.
// Advanced navigation remains available without crowding this quick reference.
func (t *TextArea) Shortcuts() []core.Shortcut {
	k := t.KeyMap
	return []core.Shortcut{
		core.ClearText(), core.Undo(), core.Redo(),
		core.Cut(), k.CopySelection, k.Paste, k.SelectAll,
		core.SelectText(k.SelectCharacterBackward, k.SelectCharacterForward, k.SelectLineUp, k.SelectLineDown),
		k.LineStart, k.LineEnd,
	}
}
