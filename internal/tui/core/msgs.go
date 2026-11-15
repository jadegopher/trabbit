package core

import tea "charm.land/bubbletea/v2"

type OpenDialog struct {
	Model tea.Model
	After tea.Cmd
}
type ReplaceDialog struct{ Old, Model tea.Model }
type DialogResult struct {
	Dialog tea.Model
	Value  tea.Msg
	Close  bool
}
type DialogFeedback struct {
	Dialog tea.Model
	Err    error
}
type VisibilityMsg struct{ Visible bool }

// OpenScreen covers the current scene and preserves it for a matching return.
type OpenScreen struct{ Model tea.Model }

type CloseScreen struct{ Model tea.Model }

func Cmd(msg tea.Msg) tea.Cmd { return func() tea.Msg { return msg } }

// ConfirmDialog temporarily covers an existing dialog and restores it on cancel.
// Accept is delivered after the covered dialog is restored.
type ConfirmDialog struct {
	Owner   tea.Model
	Message string
	Accept  tea.Msg
}
