package components

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestTextInput_Style(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	ti := NewTextInput(theme)

	// Initially empty, should use blurred text style when blurred
	ti.Blur()
	ti.Update(struct{}{}) // Trigger style update
	assert.Equal(t, theme.BlurredText.GetForeground(), ti.Model.Styles().Blurred.Text.GetForeground())

	// Filled, should use contrast text style when blurred
	ti.Model.SetValue("filled")
	ti.Update(struct{}{})
	assert.Equal(t, theme.TextColor, ti.Model.Styles().Blurred.Text.GetForeground())
}

func TestKeyValueField_Navigation(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	kv := NewKeyValueField(theme, "K:", "V:")
	kv.Key.Focus()
	kv.Value.Blur()
	kv.Focus()

	// Initially Key is focused
	assert.Equal(t, 0, kv.focusedIdx)
	assert.True(t, kv.Key.Focused())
	assert.False(t, kv.Value.Focused())

	// Down moves to Value
	ok, _ := kv.OnDown()
	assert.True(t, ok)
	assert.Equal(t, 1, kv.focusedIdx)
	assert.False(t, kv.Key.Focused())
	assert.True(t, kv.Value.Focused())

	// Up moves to Key
	ok, _ = kv.OnUp()
	assert.True(t, ok)
	assert.Equal(t, 0, kv.focusedIdx)
	assert.True(t, kv.Key.Focused())
}

func TestKeyValues_ExplicitRows(t *testing.T) {
	t.Parallel()

	kvs := NewKeyValues(core.DefaultTheme(), "Header", "K:", "V:")
	kvs.SetWidth(25)
	kvs.Focus()
	kvs.Update(tea.PasteMsg{Content: "key1"})
	require.Len(t, kvs.Rows, 1, "typing does not add a pair")
	first := kvs.Rows[0]
	first.Value.SetValue("keep me")

	kvs.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	require.Len(t, kvs.Rows, 2)
	require.True(t, kvs.Rows[1].Key.Focused())
	require.False(t, first.Key.Focused())
	require.Equal(t, "key1", first.Key.Value())
	require.Equal(t, "keep me", first.Value.Value())
	require.Equal(t, first.Key.Width(), kvs.Rows[1].Key.Width())
	kvs.Update(tea.PasteMsg{Content: "key2"})
	second := kvs.Rows[1]
	kvs.FocusBoundary(false)
	kvs.Focus()
	first.Key.SetValue("")
	first.Value.SetValue("")
	kvs.Update(struct{}{})
	require.Len(t, kvs.Rows, 2, "clearing does not delete a pair")

	kvs.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	require.Len(t, kvs.Rows, 1)
	require.Same(t, second, kvs.Rows[0])
	require.Equal(t, "key2", kvs.Rows[0].Key.Value())
	require.True(t, kvs.Rows[0].Key.Focused())

	kvs.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	require.Len(t, kvs.Rows, 1, "the last deletion leaves an empty editable pair")
	require.Empty(t, kvs.Rows[0].Key.Value())
	require.Empty(t, kvs.Rows[0].Value.Value())
	require.True(t, kvs.Rows[0].Key.Focused())
}

func TestKeyValues_Map(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		value    string
		expected interface{}
		absent   bool
	}{
		{"string", "str", `"foo"`, "foo", false},
		{"bool", "bool", "true", true, false},
		{"number", "num", "123", float64(123), false},
		{"array", "arr", `["a","b","c"]`, []interface{}{"a", "b", "c"}, false},
		{"nil", "empty", `null`, nil, false},
		{"empty pair", "", "", nil, true},
	}

	theme := core.DefaultTheme()
	kvs := NewKeyValues(theme, "Header", "K:", "V:")

	for i, tt := range tests {
		if i >= len(kvs.Rows) {
			kvs.Rows = append(kvs.Rows, NewKeyValueField(theme, "K:", "V:"))
		}
		kvs.cursor = i
		kvs.Rows[i].Key.Model.SetValue(tt.key)
		kvs.Rows[i].Value.Model.SetValue(tt.value)
		kvs.Update(struct{}{})
	}

	m, err := kvs.Map()
	require.NoError(t, err)

	for _, tt := range tests {
		if tt.key == "" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			val, ok := m[tt.key]
			if tt.absent {
				assert.False(t, ok)
			} else {
				assert.True(t, ok)
				assert.Equal(t, tt.expected, val)
			}
		})
	}

	_, ok := m[""]
	assert.False(t, ok)
}

func TestKeyValues_Navigation(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	kvs := NewKeyValues(theme, "Header", "K:", "V:")

	// Add second row
	kvs.Rows[0].Key.Model.SetValue("k1")
	kvs.AddPair()
	kvs.FocusBoundary(false)

	// Initially row 0, key focused
	assert.Equal(t, 0, kvs.cursor)
	assert.Equal(t, 0, kvs.Rows[0].focusedIdx)

	// Down within row 0 moves to value
	ok, _ := kvs.OnDown()
	assert.True(t, ok)
	assert.Equal(t, 1, kvs.Rows[0].focusedIdx)

	// Down moves to row 1, key focused
	ok, _ = kvs.OnDown()
	assert.True(t, ok)
	assert.Equal(t, 1, kvs.cursor)
	assert.Equal(t, 0, kvs.Rows[1].focusedIdx)

	// Up moves back to row 0, value focused
	ok, _ = kvs.OnUp()
	assert.True(t, ok)
	assert.Equal(t, 0, kvs.cursor)
	assert.Equal(t, 1, kvs.Rows[0].focusedIdx)
}

func TestKeyValues_Cursor(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	kvs := NewKeyValues(theme, "Header", "K:", "V:")
	kvs.Focus()

	// Header is 1 line.
	// Row 0 Key is focused.
	c := kvs.Cursor()
	assert.NotNil(t, c)
	assert.Equal(t, 1, c.Y) // Header(1) + Key(0)

	// Row 0 Value is focused.
	kvs.Rows[0].focusedIdx = 1
	kvs.Rows[0].Focus()
	c = kvs.Cursor()
	assert.NotNil(t, c)
	assert.Equal(t, 2, c.Y) // Header(1) + Value(1)

	// Row 1 Key is focused.
	kvs.Rows[0].Key.Model.SetValue("k1")
	kvs.AddPair()
	c = kvs.Cursor()
	assert.NotNil(t, c)
	assert.Equal(t, 4, c.Y) // Header(1) + Row0(2) + Spacer(1) + Key(0)
}

func TestUIKeyValues(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()
	kvs := NewKeyValues(theme, "Arguments:", "  Key:", "  Value (JSON):")
	kvs.Rows[0].Key.Model.SetValue("x-max-length")
	kvs.Rows[0].Value.Model.SetValue("1000")
	kvs.AddPair()
	kvs.FocusBoundary(false)
	kvs.Focus()

	core.AssertGolden(t, "key_values_view", kvs.View(true))
}

func TestKeyValuesUsesThemeForHeadersAndFocusedLabels(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	// Same-length transforms expose style selection without global color settings.
	theme.HeaderStyle = theme.HeaderStyle.Transform(strings.ToUpper)
	theme.LabelFocused = theme.LabelFocused.Transform(strings.ToUpper)
	for _, header := range []string{"Properties", "Headers"} {
		pairs := NewKeyValues(theme, header, "Key:", "Value:")
		pairs.Focus()
		require.Contains(t, pairs.View(false), strings.ToUpper(header))
		require.Contains(t, pairs.View(true), "KEY:")
		require.NotContains(t, pairs.View(true), "VALUE:")
		pairs.OnDown()
		require.Contains(t, pairs.View(true), "VALUE:")
		require.NotContains(t, pairs.View(true), "KEY:")
		pairs.Rows[0].Key.SetValue("populated")
		pairs.Blur()
		require.Contains(t, pairs.View(false), strings.ToUpper(header))
		require.NotContains(t, pairs.View(false), "VALUE:")
	}
}

func TestKeyValuesNumberingWidthsAndDeletingLastPair(t *testing.T) {
	t.Parallel()
	pairs := NewKeyValues(core.DefaultTheme(), "Headers", "Key:", "Value:")
	pairs.SetWidth(24)
	for len(pairs.Rows) < 10 {
		pairs.AddPair()
	}
	pairs.Rows[0].Key.SetValue("keep")
	pairs.Rows[0].Value.SetValue("true")
	require.Contains(t, ansi.Strip(pairs.View(true)), "10. Key:")
	for _, row := range pairs.Rows {
		require.Equal(t, pairs.Rows[0].Key.Width(), row.Key.Width())
	}
	view := strings.Split(pairs.View(true), "\n")
	cursor := pairs.Cursor()
	require.NotNil(t, cursor)
	require.Contains(t, view[cursor.Y], "10.")
	require.Equal(t, pairs.numberWidth()+pairs.Rows[0].labelWidth()+keyValueLabelGapColumns, cursor.X)
	oldWidth := pairs.Rows[0].Key.Width()
	pairs.Update(tea.KeyPressMsg{Code: tea.KeyF3})
	require.Len(t, pairs.Rows, 9)
	require.True(t, pairs.Rows[8].Key.Focused())
	require.Greater(t, pairs.Rows[0].Key.Width(), oldWidth, "number gutter shrinks after deleting pair 10")
	values, err := pairs.Map()
	require.NoError(t, err)
	require.Equal(t, true, values["keep"])
}

func TestArgumentsAcceptPlainTextAndPreserveTypes(t *testing.T) {
	t.Parallel()
	args := NewKeyValues(core.DefaultTheme(), "Arguments", "Key", "Value")
	row := args.Rows[0]
	row.Key.SetValue("custom")
	for _, value := range []string{"hello world", "{incomplete", ""} {
		row.Value.SetValue(value)
		values, err := args.Map()
		assert.NoError(t, err)
		assert.Equal(t, value, values["custom"])
		assert.NotContains(t, args.View(true), "JSON")
	}
	row.Value.SetValue("123")
	values, err := args.Map()
	assert.NoError(t, err)
	assert.Equal(t, float64(123), values["custom"])
	row.Value.SetValue("true")
	values, err = args.Map()
	assert.NoError(t, err)
	assert.Equal(t, true, values["custom"])
}

func TestKeyValuesRejectsValueWithoutKeyAndRetainsDraft(t *testing.T) {
	t.Parallel()
	pairs := NewKeyValues(core.DefaultTheme(), "Headers", "Key:", "Value:")
	row := pairs.Rows[0]
	row.Key.SetValue("name")
	row.Value.SetValue("keep me")
	pairs.Update(struct{}{})
	row.Key.SetValue("")
	pairs.Update(struct{}{})
	require.Same(t, row, pairs.Rows[0])
	require.Equal(t, "keep me", row.Value.Value())
	_, err := pairs.Map()
	require.ErrorContains(t, err, "key is required")
	row.Key.SetValue("restored")
	row.Value.SetValue("")
	values, err := pairs.Map()
	require.NoError(t, err)
	require.Equal(t, "", values["restored"])
}

func TestTextInputProgrammaticValueAndHistoryColors(t *testing.T) {
	t.Parallel()
	theme := core.DefaultTheme()
	input := NewTextInput(theme)
	input.Focus()
	input.SetValue("preset")
	require.Equal(t, theme.TextColor, input.Styles().Blurred.Text.GetForeground())
	input.Reset()
	require.Equal(t, theme.Inactive, input.Styles().Blurred.Text.GetForeground())
	input.Update(tea.PasteMsg{Content: "typed"})
	input.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModSuper})
	require.Empty(t, input.Value())
	require.Equal(t, theme.Inactive, input.Styles().Blurred.Text.GetForeground())
	input.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModSuper | tea.ModShift})
	require.Equal(t, "typed", input.Value())
	require.Equal(t, theme.TextColor, input.Styles().Blurred.Text.GetForeground())
}
