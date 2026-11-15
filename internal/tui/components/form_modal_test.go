package components

import (
	"charm.land/lipgloss/v2"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestFormKeepsWidthFooterAndCanScrollToEnd(t *testing.T) {
	theme := core.DefaultTheme()
	fields := []FormField{}
	for i := 0; i < 20; i++ {
		fields = append(fields, FormField{Label: "Field", Item: NewTextInput(theme)})
	}
	m := NewFormModal(theme, "Form", fields)
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	width := lipgloss.Width(m.View().Content)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 15})
	assert.Equal(t, width, lipgloss.Width(m.View().Content))
	assert.LessOrEqual(t, lipgloss.Height(m.View().Content), 15)
	for i := 0; i < 100; i++ {
		m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	}
	body := m.renderContent()
	assert.Equal(t, lipgloss.Height(body.Content)-m.bodyHeight(body), m.scrollOffset)
	lines := strings.Split(m.View().Content, "\n")
	assert.Contains(t, strings.Join(lines[len(lines)-5:], "\n"), "submit")
	assert.Contains(t, m.View().Content, "▐")
}

func TestArgumentsNavigateAndEscapeNeedsConfirmation(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	args := NewKeyValues(theme, "Arguments", "Key", "Value")
	args.Rows[0].Key.SetValue("ttl")
	args.AddPair()
	m := NewFormModal(theme, "Add queue", []FormField{{Label: "Name", Item: NewTextInput(theme)}, {Item: args}})
	m.WrapNavigation = true
	m.ConfirmCancel = true
	m.Init()
	for i := 0; i < 5; i++ {
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	assert.Equal(t, 0, m.cursor, "down traverses key/value rows then returns to Name")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})

	if assert.NotNil(t, cmd) {
		request := cmd().(core.ConfirmDialog)
		assert.Same(t, m, request.Owner)
		assert.True(t, request.Accept.(core.DialogResult).Close)
	}

}

func TestFormViewDoesNotMoveCursorOrScroll(t *testing.T) {
	theme := core.DefaultTheme()
	fields := []FormField{}
	for i := 0; i < 20; i++ {
		fields = append(fields, FormField{Label: "界:", Item: NewTextInput(theme)})
	}
	m := NewFormModal(theme, "Form", fields)
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 15})
	for i := 0; i < 19; i++ {
		m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	first := m.View()
	offset := m.scrollOffset
	for i := 0; i < 20; i++ {
		assert.Equal(t, first.Cursor, m.View().Cursor)
		assert.Equal(t, offset, m.scrollOffset)
	}
}

func TestCtrlAMovesCursorWithoutErasing(t *testing.T) {
	theme := core.DefaultTheme()
	field := NewTextInput(theme)
	field.SetValue("keep this text")
	m := NewFormModal(theme, "Form", []FormField{{Label: "Name", Item: field}})
	m.Init()
	m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	assert.Equal(t, "keep this text", field.Value())
	assert.Equal(t, 0, field.Position())
	m.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	assert.Empty(t, field.Value())
}

func TestFormFailureRetainsDraftAndBusyResize(t *testing.T) {
	theme := core.DefaultTheme()
	field := NewTextInput(theme)
	field.SetValue("draft")
	m := NewFormModal(theme, "Form", []FormField{{Label: "Name", Item: field}})
	m.Init()
	m.OnSubmit = func() (tea.Msg, error) { return nil, errors.New("invalid") }
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, "draft", field.Value())
	assert.False(t, m.busy)
	m.OnSubmit = func() (tea.Msg, error) { return struct{}{}, nil }
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.True(t, m.busy)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 15})
	assert.Equal(t, 15, m.screenHeight)
	assert.Nil(t, m.View().Cursor)
	m.Update(core.DialogFeedback{Dialog: m, Err: errors.New("offline")})
	assert.False(t, m.busy)
	assert.Equal(t, "draft", field.Value())
	assert.NotNil(t, m.View().Cursor)
}

func TestFormModal_Scrolling(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()

	// Create a modal with many fields to force scrolling
	fields := []FormField{}
	for i := 0; i < 20; i++ {
		fields = append(
			fields, FormField{
				Label: "Field: ",
				Item:  NewTextInput(theme),
			},
		)
	}

	m := NewFormModal(theme, "Scroll Test Modal", fields)
	m.WrapNavigation = false
	m.Init()

	// Set small window size to force scrolling
	// 10 lines is very small, modal maxHeight will be 10 (clamped)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	t.Run(
		"Initial view - scrolled to top", func(t *testing.T) {
			view := m.View()
			core.AssertGolden(t, "form_modal_scroll_top", view.Content)
			assert.Equal(t, 0, m.scrollOffset)
			if assert.NotNil(t, view.Cursor) {
				assert.Equal(t, 4, view.Cursor.Y)
			}
		},
	)

	t.Run(
		"Scroll to bottom via navigation", func(t *testing.T) {
			// Navigate down until we reach the bottom
			for i := 0; i < 25; i++ {
				m.Update(tea.KeyPressMsg{Text: core.KeyDown})
				_ = m.View() // Trigger auto-scroll logic in View
			}

			// We should be at the last index and it shouldn't have wrapped
			assert.Equal(t, 19, m.cursor)
			assert.Greater(t, m.scrollOffset, 0)

			view := m.View()
			core.AssertGolden(t, "form_modal_scroll_bottom", view.Content)

			// Now go back to top
			for i := 0; i < 25; i++ {
				m.Update(tea.KeyPressMsg{Text: core.KeyUp})
				_ = m.View()
			}

			assert.Equal(t, 0, m.cursor)
			assert.Equal(t, 0, m.scrollOffset)
		},
	)

	t.Run(
		"Manual scroll via mouse wheel", func(t *testing.T) {
			m.cursor = 0
			m.scrollOffset = 0
			// Simulate mouse wheel down
			m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
			assert.Equal(t, 0, m.cursor)
			assert.Equal(t, 1, m.scrollOffset)

			// Simulate mouse wheel up
			m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
			assert.Equal(t, 0, m.cursor)
		},
	)
}
