package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

// FormBody is an embeddable, scrollable field column without modal behavior.
// Its parent controls focus boundaries and submission.
type FormBody struct {
	Fields                       []FormField
	WrapNavigation               bool
	theme                        core.Theme
	index, width, height, offset int
	focused                      bool
	labelWidth                   int
}

func NewFormBody(theme core.Theme, fields []FormField) *FormBody {
	return &FormBody{theme: theme, Fields: fields}
}

func (f *FormBody) SetSize(width, height int) {
	f.width, f.height = max(0, width), max(0, height)
	f.labelWidth = 0
	for _, field := range f.Fields {
		if field.Label != "" {
			f.labelWidth = max(f.labelWidth, lipgloss.Width(field.Label)+1)
		}
	}
	f.labelWidth = min(f.labelWidth, max(0, f.width-3))
	for _, field := range f.Fields {
		switch item := field.Item.(type) {
		case *TextInput:
			labelWidth := 0
			if field.Label != "" {
				labelWidth = f.labelWidth
			}
			item.SetWidth(max(1, width-labelWidth-core.ScrollbarColumns))
		case *KeyValues:
			item.SetWidth(max(1, width-core.ScrollbarColumns))
		}
	}
	f.reveal()
}

func (f *FormBody) Focus() tea.Cmd {
	f.focused = true
	cmd := f.Fields[f.index].Item.Focus()
	f.reveal()
	return cmd
}

func (f *FormBody) Scroll(delta int) {
	f.offset = max(0, min(f.offset+delta, lipgloss.Height(f.content().Content)-f.height))
}

func (f *FormBody) Blur() {
	f.focused = false
	for _, field := range f.Fields {
		field.Item.Blur()
	}
}

// Move traverses nested inputs before changing fields. Unless wrapping is enabled,
// it returns false at a boundary so the host can focus an adjacent component.
func (f *FormBody) Move(back bool) (bool, tea.Cmd) {
	item := f.Fields[f.index].Item
	if (back && item.PrevField()) || (!back && item.NextField()) {
		f.reveal()
		return true, item.Focus()
	}
	next := f.index + 1
	if back {
		next = f.index - 1
	}
	if next < 0 || next >= len(f.Fields) {
		if !f.WrapNavigation {
			return false, nil
		}
		next = (next + len(f.Fields)) % len(f.Fields)
	}
	item.Blur()
	f.index = next
	if rows, ok := f.Fields[next].Item.(*KeyValues); ok {
		rows.FocusBoundary(back)
	}
	f.reveal()
	return true, f.Focus()
}

func (f *FormBody) Update(msg tea.Msg) tea.Cmd {
	if !f.focused {
		return nil
	}
	switch msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		defer f.reveal()
	}
	item := f.Fields[f.index].Item
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if handled, cmd := item.OnToggle(key); handled {
			return cmd
		}
	}
	return item.Update(msg)
}

// fieldView shares row geometry between rendering, cursor placement and hit testing.
func (f *FormBody) fieldView(index int) tea.View {
	field := f.Fields[index]
	focused := f.focused && index == f.index
	value := tea.NewView(field.Item.View(focused))
	if focused {
		value.Cursor = field.Item.Cursor()
	}
	if field.Label == "" {
		return value
	}

	style := f.theme.LabelStyle
	if focused {
		style = f.theme.LabelFocused
	}
	label := core.CropView(tea.NewView(style.Render(field.Label)), f.labelWidth, 1, 0)
	return core.NewLayout(f.width-core.ScrollbarColumns, lipgloss.Height(value.Content), core.Horizontal).
		AddFixedView("label", f.labelWidth, label).
		AddRelativeView("value", 1, value).Render()
}

func (f *FormBody) content() tea.View {
	var lines []string
	var cursor *tea.Cursor
	y := 0
	for index := range f.Fields {
		row := f.fieldView(index)
		if row.Cursor != nil {
			copy := *row.Cursor
			copy.Y += y
			cursor = &copy
		}
		lines = append(lines, row.Content)
		y += lipgloss.Height(row.Content)
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.Cursor = cursor
	return v
}

func (f *FormBody) reveal() {
	v := f.content()
	y := 0
	for i := 0; i < f.index; i++ {
		y += lipgloss.Height(f.fieldView(i).Content)
	}
	if v.Cursor != nil {
		y = v.Cursor.Y
	}
	if y < f.offset {
		f.offset = y
	}
	if y >= f.offset+f.height {
		f.offset = y - f.height + 1
	}
	f.offset = max(0, min(f.offset, lipgloss.Height(v.Content)-f.height))
}

func (f *FormBody) View() tea.View {
	v := f.content()
	total := lipgloss.Height(v.Content)
	v = core.CropView(v, max(0, f.width-core.ScrollbarColumns), f.height, f.offset)
	if f.width <= 0 || f.height <= 0 {
		return v
	}
	v.Content = f.theme.RenderScrollbarViewport(f.width, f.height, v.Content, total, f.offset, f.focused)
	return v
}

// Click maps a viewport row back to its field and nested key/value input.
func (f *FormBody) Click(row int) tea.Cmd {
	y := row + f.offset
	for index, field := range f.Fields {
		height := lipgloss.Height(f.fieldView(index).Content)
		if y < height {
			f.Blur()
			f.index = index
			if rows, ok := field.Item.(*KeyValues); ok {
				rows.ClickRow(y)
			}
			return f.Focus()
		}
		y -= height
	}
	return nil
}

func (f *FormBody) FocusedItem() FormItem { return f.Fields[f.index].Item }

func (f *FormBody) Focused() bool { return f.focused }

func (f *FormBody) ScrollOffset() int { return f.offset }

func (f *FormBody) ScrollPage(back bool) {
	delta := max(1, f.height)
	if back {
		delta = -delta
	}
	f.Scroll(delta)
}
