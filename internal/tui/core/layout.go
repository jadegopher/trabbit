package core

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type LayoutDirection int

const (
	Vertical LayoutDirection = iota
	Horizontal
)

const (
	DefaultSpacing = 1
)

type SlotType int

const (
	Fixed SlotType = iota
	Relative
)

type Slot struct {
	Type  SlotType
	Value int
}

type Layout struct {
	Width     int
	Height    int
	Direction LayoutDirection
	Slots     map[string]Slot
	order     []string
	views     map[string]tea.View
}

func NewLayout(width, height int, dir LayoutDirection) *Layout {
	return &Layout{
		Width:     width,
		Height:    height,
		Direction: dir,
		Slots:     make(map[string]Slot),
		views:     make(map[string]tea.View),
	}
}

func (l *Layout) AddFixed(name string, size int) *Layout {
	if _, ok := l.Slots[name]; !ok {
		l.order = append(l.order, name)
	}
	l.Slots[name] = Slot{Type: Fixed, Value: size}
	return l
}

func (l *Layout) AddFixedView(name string, size int, view tea.View) *Layout {
	l.AddFixed(name, size)
	l.views[name] = view
	return l
}

func (l *Layout) AddRelative(name string, weight int) *Layout {
	if _, ok := l.Slots[name]; !ok {
		l.order = append(l.order, name)
	}
	l.Slots[name] = Slot{Type: Relative, Value: weight}
	return l
}

func (l *Layout) AddRelativeView(name string, weight int, view tea.View) *Layout {
	l.AddRelative(name, weight)
	l.views[name] = view
	return l
}

func (l *Layout) AddSpacing(name string) *Layout {
	return l.AddFixedView(name, 0, tea.NewView(""))
}

func (l *Layout) Render() tea.View {
	return l.JoinViews(l.views)
}

func (l *Layout) GetSize(name string) tea.WindowSizeMsg {
	if _, ok := l.Slots[name]; !ok {
		return tea.WindowSizeMsg{}
	}
	size := l.getValue(name)
	if l.Direction == Vertical {
		return tea.WindowSizeMsg{Width: l.Width, Height: size}
	}
	return tea.WindowSizeMsg{Width: size, Height: l.Height}
}

func (l *Layout) Join(views map[string]string) string {
	var ordered []string
	for _, name := range l.order {
		if v, ok := views[name]; ok {
			ordered = append(ordered, v)
		}
	}

	if l.Direction == Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, ordered...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, ordered...)
}

func (l *Layout) JoinViews(views map[string]tea.View) tea.View {
	var ordered []string
	var cursor *tea.Cursor

	x, y := 0, 0
	for _, name := range l.order {
		v, ok := views[name]
		if !ok {
			continue
		}

		ordered = append(ordered, v.Content)

		if v.Cursor != nil {
			c := *v.Cursor
			c.X += x
			c.Y += y
			cursor = &c
		}

		w, h := lipgloss.Size(v.Content)
		size := w
		if l.Direction == Vertical {
			size = h
		}

		if l.Direction == Vertical {
			y += size
		} else {
			x += size
		}
	}

	var content string
	if l.Direction == Vertical {
		content = lipgloss.JoinVertical(lipgloss.Left, ordered...)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, ordered...)
	}

	res := tea.NewView(content)
	res.Cursor = cursor
	return res
}

func (l *Layout) getValueForView(name string, views map[string]tea.View) int {
	slot, ok := l.Slots[name]
	if !ok {
		return 0
	}

	if slot.Type == Fixed {
		if slot.Value > 0 {
			return slot.Value
		}
		if v, ok := views[name]; ok {
			w, h := lipgloss.Size(v.Content)
			if l.Direction == Vertical {
				return h
			}
			return w
		}
		return 0
	}

	return l.getValue(name)
}

func (l *Layout) Place(content string) string {
	return lipgloss.Place(l.Width, l.Height, lipgloss.Center, lipgloss.Center, content)
}

func (l *Layout) PlaceView(v tea.View) tea.View {
	w, h := lipgloss.Size(v.Content)
	v.Content = l.Place(v.Content)
	if v.Cursor != nil {
		c := *v.Cursor
		c.X += max(0, (l.Width-w)/2)
		c.Y += max(0, (l.Height-h)/2)
		v.Cursor = &c
	}
	return v
}

func (l *Layout) GetHeight(name string) int {
	if l.Direction == Vertical {
		return l.getValue(name)
	}
	return l.Height
}

func (l *Layout) GetWidth(name string) int {
	if l.Direction == Horizontal {
		return l.getValue(name)
	}
	return l.Width
}

func (l *Layout) GetOffset(name string) (int, int) {
	if _, ok := l.Slots[name]; !ok {
		return 0, 0
	}

	x, y := 0, 0
	for _, n := range l.order {
		if n == name {
			break
		}
		size := l.getValue(n)
		if l.Direction == Vertical {
			y += size
		} else {
			x += size
		}
	}
	return x, y
}

func (l *Layout) getValue(name string) int {
	slot, ok := l.Slots[name]
	if !ok {
		return 0
	}

	totalFixed := 0
	totalRelative := 0
	for _, s := range l.Slots {
		if s.Type == Fixed {
			totalFixed += s.Value
		} else {
			totalRelative += s.Value
		}
	}

	totalAvailable := l.Height
	if l.Direction == Horizontal {
		totalAvailable = l.Width
	}

	remaining := totalAvailable - totalFixed
	if remaining < 0 {
		remaining = 0
	}

	if slot.Type == Fixed {
		return slot.Value
	} else if totalRelative > 0 {
		sumRelative := 0
		for _, n := range l.order {
			s := l.Slots[n]
			if s.Type == Relative {
				prevSumRelative := sumRelative
				sumRelative += s.Value
				if n == name {
					return (remaining * sumRelative / totalRelative) - (remaining * prevSumRelative / totalRelative)
				}
			}
		}
	}

	return 0
}
