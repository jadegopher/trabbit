package core

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestLayoutVertical(t *testing.T) {
	t.Parallel()
	l := NewLayout(80, 20, Vertical).
		AddFixed("top", 3).
		AddRelative("content", 1).
		AddFixed("bottom", 2)

	assert.Equal(t, 3, l.GetHeight("top"))
	assert.Equal(t, 15, l.GetHeight("content")) // 20 - 3 - 2 = 15
	assert.Equal(t, 2, l.GetHeight("bottom"))

	assert.Equal(t, tea.WindowSizeMsg{Width: 80, Height: 3}, l.GetSize("top"))
	assert.Equal(t, tea.WindowSizeMsg{Width: 80, Height: 15}, l.GetSize("content"))
	assert.Equal(t, tea.WindowSizeMsg{Width: 80, Height: 2}, l.GetSize("bottom"))

	x, y := l.GetOffset("top")
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)

	x, y = l.GetOffset("content")
	assert.Equal(t, 0, x)
	assert.Equal(t, 3, y)

	x, y = l.GetOffset("bottom")
	assert.Equal(t, 0, x)
	assert.Equal(t, 18, y)
}

func TestLayoutHorizontal(t *testing.T) {
	t.Parallel()
	l := NewLayout(100, 20, Horizontal).
		AddFixed("left", 10).
		AddRelative("middle", 3).
		AddRelative("right", 1)

	assert.Equal(t, 10, l.GetWidth("left"))
	// 100 - 10 = 90
	// middle: 90 * 3 / 4 = 67
	// right: 90 * 1 / 4 = 23 (distributed remainder)
	assert.Equal(t, 67, l.GetWidth("middle"))
	assert.Equal(t, 23, l.GetWidth("right"))

	x, y := l.GetOffset("left")
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)

	x, y = l.GetOffset("middle")
	assert.Equal(t, 10, x)
	assert.Equal(t, 0, y)

	x, y = l.GetOffset("right")
	assert.Equal(t, 77, x)
	assert.Equal(t, 0, y)
}

func TestLayoutInvalidSlot(t *testing.T) {
	t.Parallel()
	l := NewLayout(80, 20, Vertical).AddFixed("top", 3)
	assert.Equal(t, 0, l.GetHeight("invalid"))
	assert.Equal(t, tea.WindowSizeMsg{}, l.GetSize("invalid"))
}

func TestLayoutOverflow(t *testing.T) {
	t.Parallel()
	l := NewLayout(80, 10, Vertical).
		AddFixed("top", 8).
		AddFixed("bottom", 4).
		AddRelative("content", 1)

	assert.Equal(t, 8, l.GetHeight("top"))
	assert.Equal(t, 4, l.GetHeight("bottom"))
	assert.Equal(t, 0, l.GetHeight("content"))
}

func TestLayoutJoinViews(t *testing.T) {
	t.Parallel()
	l := NewLayout(0, 0, Vertical).
		AddFixed("header", 2).
		AddFixed("input", 1)

	header := tea.NewView("Line 1\nLine 2")
	input := tea.NewView("Input")
	input.Cursor = tea.NewCursor(1, 0)

	views := map[string]tea.View{
		"header": header,
		"input":  input,
	}

	res := l.JoinViews(views)
	assert.NotNil(t, res.Cursor)
	assert.Equal(t, 1, res.Cursor.X)
	assert.Equal(t, 2, res.Cursor.Y)
}

func TestLayoutJoinViewsAuto(t *testing.T) {
	t.Parallel()
	l := NewLayout(0, 0, Vertical).
		AddFixed("header", 0).
		AddFixed("input", 0)

	header := tea.NewView("Header\nMore Header")
	input := tea.NewView("Input")
	input.Cursor = tea.NewCursor(1, 0)

	views := map[string]tea.View{
		"header": header,
		"input":  input,
	}

	res := l.JoinViews(views)
	assert.NotNil(t, res.Cursor)
	assert.Equal(t, 1, res.Cursor.X)
	assert.Equal(t, 2, res.Cursor.Y)
}

func TestLayoutHorizontalJoinViews(t *testing.T) {
	t.Parallel()
	l := NewLayout(0, 0, Horizontal).
		AddFixed("label", 10).
		AddFixed("input", 0)

	label := tea.NewView("Label:    ") // 10 chars
	input := tea.NewView("Input")
	input.Cursor = tea.NewCursor(2, 0)

	views := map[string]tea.View{
		"label": label,
		"input": input,
	}

	res := l.JoinViews(views)
	assert.NotNil(t, res.Cursor)
	assert.Equal(t, 12, res.Cursor.X) // 10 + 2
	assert.Equal(t, 0, res.Cursor.Y)
}
