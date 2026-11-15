package components

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

type FormItem interface {
	Update(tea.Msg) tea.Cmd
	View(focused bool) string
	Focus() tea.Cmd
	Blur()
	Reset()
	OnClean() tea.Cmd
	OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd)
	OnUp() (bool, tea.Cmd)
	OnDown() (bool, tea.Cmd)
	Cursor() *tea.Cursor

	// Navigation
	NextField() bool
	PrevField() bool
}

type TextArea struct {
	textarea.Model
	undo                      []textSnapshot
	redo                      []textSnapshot
	outerWidth                int
	rowNumbers                bool
	numberWidth, numberedRows int
	scrollContent             string
	scrollOffset, scrollTotal int
	dragging                  bool
	cursorViewport            bool
}

type textSnapshot struct {
	value    string
	row, col int
}

func NewTextArea(theme core.Theme) *TextArea {
	ta := textarea.New()
	ta.SetStyles(theme.TextAreaStyles())
	ta.SetVirtualCursor(false)
	ta.Placeholder = "Payload..."
	// Standard editor shortcuts replace the textarea's Emacs-style select-all key.
	ta.KeyMap.LineStart = core.LineStart()
	ta.KeyMap.LineEnd = core.LineEnd()
	ta.KeyMap.SelectAll = core.SelectAll()
	ta.KeyMap.CopySelection = core.Copy()
	ta.KeyMap.Paste = core.Paste()
	ta.KeyMap.InputBegin = core.DocumentStart()
	ta.KeyMap.InputEnd = core.DocumentEnd()
	ta.Focus()
	ta.CharLimit = textAreaPayloadLimitBytes
	return &TextArea{Model: ta}
}

func (t *TextArea) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		if t.scrollContent == "" || t.cursorViewport {
			defer t.preserveViewport()()
		}
		t.EndMouseSelection()
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, core.ClearText()):
			if t.Focused() {
				t.Reset()
			}
			return nil
		case key.Matches(k, core.Cut()):
			return t.Cut()
		case key.Matches(k, core.Undo()):
			t.restoreHistory(&t.undo, &t.redo)
			return nil
		case key.Matches(k, core.Redo()):
			t.restoreHistory(&t.redo, &t.undo)
			return nil
		}
	}
	before := t.snapshot()
	defer func() {
		if t.Value() != before.value {
			t.remember(before)
			t.syncRowNumbers()
		}
	}()
	switch msg.(type) {
	case tea.KeyPressMsg, tea.PasteMsg:
		t.scrollContent = ""
	}

	if k, ok := msg.(tea.KeyPressMsg); ok {
		if handled, cmd := t.selectBoundary(k); handled {
			return cmd
		}
	}
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(msg)
	return cmd
}

func (t *TextArea) View(focused bool) string {
	return t.Model.View()
}

func (t *TextArea) Focus() tea.Cmd {
	return t.Model.Focus()
}

func (t *TextArea) Blur() {
	t.EndMouseSelection()
	t.Model.Blur()
}

func (t *TextArea) Reset() {
	t.SetValue("")
}

func (t *TextArea) OnClean() tea.Cmd {
	t.Reset()
	return nil
}

func (t *TextArea) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	return false, nil
}

func (t *TextArea) OnUp() (bool, tea.Cmd)   { return false, nil }
func (t *TextArea) OnDown() (bool, tea.Cmd) { return false, nil }

func (t *TextArea) Cursor() *tea.Cursor {
	if t.scrollContent != "" {
		if !t.cursorViewport {
			return nil
		}
		cursor := t.Model.Cursor()
		if cursor == nil {
			return nil
		}
		position := *cursor
		position.Y += t.ScrollYOffset() - t.scrollOffset
		if position.Y < 0 || position.Y >= t.Height() {
			return nil
		}
		return &position
	}
	return t.Model.Cursor()
}

func (t *TextArea) NextField() bool { return false }
func (t *TextArea) PrevField() bool { return false }

type Select struct {
	Options []string
	Index   int
	Theme   core.Theme
}

func NewSelect(theme core.Theme, options []string, initialIdx int) *Select {
	return &Select{
		Options: options,
		Index:   initialIdx,
		Theme:   theme,
	}
}

func (s *Select) View(isSelected bool) string {
	val := ""
	width := 0
	for _, option := range s.Options {
		width = max(width, lipgloss.Width(option))
	}
	if len(s.Options) > 0 {
		val = s.Options[s.Index]
	}

	// Keep the closing marker and the field footprint fixed when presets differ
	// in terminal-cell width, so shorter values cover the previous selection.
	val += strings.Repeat(" ", max(0, width-lipgloss.Width(val)))
	if isSelected {
		val = "< " + val + " >"
	} else {
		val += "    "
	}

	return RenderSelector(s.Theme, val, isSelected)
}

func (s *Select) Update(msg tea.Msg) tea.Cmd { return nil }
func (s *Select) Focus() tea.Cmd             { return nil }
func (s *Select) Blur()                      {}
func (s *Select) Reset()                     { s.Index = 0 }
func (s *Select) OnClean() tea.Cmd {
	s.Reset()
	return nil
}

func (s *Select) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(msg, core.SelectPrevious()) {
		s.Prev()
		return true, nil
	}
	if key.Matches(msg, core.SelectNext()) {
		s.Next()
		return true, nil
	}
	return false, nil
}

func (s *Select) OnUp() (bool, tea.Cmd)   { return false, nil }
func (s *Select) OnDown() (bool, tea.Cmd) { return false, nil }

func (s *Select) Cursor() *tea.Cursor { return nil }

func (s *Select) NextField() bool { return false }
func (s *Select) PrevField() bool { return false }

func (s *Select) Next() {
	if len(s.Options) > 0 {
		s.Index = (s.Index + 1) % len(s.Options)
	}
}

func (s *Select) Prev() {
	if len(s.Options) > 0 {
		s.Index = (s.Index - 1 + len(s.Options)) % len(s.Options)
	}
}

func (s *Select) Value() string {
	if len(s.Options) > 0 {
		return s.Options[s.Index]
	}
	return ""
}

type Toggle struct {
	Value bool
	Theme core.Theme
}

func NewToggle(theme core.Theme, initialValue bool) *Toggle {
	return &Toggle{
		Value: initialValue,
		Theme: theme,
	}
}

func (t *Toggle) View(isSelected bool) string {
	val := fmt.Sprintf("%v", t.Value)
	if isSelected {
		val = "< " + val + " >"
	}
	return RenderSelector(t.Theme, val, isSelected)
}

func (t *Toggle) Update(msg tea.Msg) tea.Cmd { return nil }
func (t *Toggle) Focus() tea.Cmd             { return nil }
func (t *Toggle) Blur()                      {}
func (t *Toggle) Reset()                     { t.Value = false }
func (t *Toggle) OnClean() tea.Cmd {
	t.Reset()
	return nil
}

func (t *Toggle) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	t.Toggle()
	return true, nil
}

func (t *Toggle) OnUp() (bool, tea.Cmd)   { return false, nil }
func (t *Toggle) OnDown() (bool, tea.Cmd) { return false, nil }

func (t *Toggle) Cursor() *tea.Cursor { return nil }

func (t *Toggle) NextField() bool { return false }
func (t *Toggle) PrevField() bool { return false }

func (t *Toggle) Toggle() {
	t.Value = !t.Value
}

type TextInput struct {
	textinput.Model
	theme      core.Theme
	undo, redo []textSnapshot
}

func NewTextInput(theme core.Theme) *TextInput {
	return &TextInput{
		Model: theme.NewTextInput(),
		theme: theme,
	}
}

func (t *TextInput) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, core.Undo()):
			t.restoreHistory(&t.undo, &t.redo)
			return nil
		case key.Matches(k, core.Redo()):
			t.restoreHistory(&t.redo, &t.undo)
			return nil
		}
	}
	before := textSnapshot{value: t.Value(), col: t.Position()}
	defer func() {
		if before.value != t.Value() {
			t.undo = append(t.undo, before)
			if len(t.undo) > editorHistoryLimit {
				t.undo = t.undo[len(t.undo)-editorHistoryLimit:]
			}
			t.redo = nil
		}
	}()
	var cmd tea.Cmd
	t.Model, cmd = t.Model.Update(msg)

	t.syncTextStyle()
	return cmd
}

func (t *TextInput) syncTextStyle() {
	styles := t.Model.Styles()
	if t.Value() != "" {
		styles.Blurred.Text = t.theme.PlainStyle.Foreground(t.theme.TextColor)
	} else {
		styles.Blurred.Text = t.theme.BlurredText
	}
	t.Model.SetStyles(styles)
}

func (t *TextInput) SetValue(value string) {
	t.Model.SetValue(value)
	t.syncTextStyle()
}

func (t *TextInput) View(focused bool) string {
	return t.Model.View()
}

func (t *TextInput) Focus() tea.Cmd {
	return t.Model.Focus()
}

func (t *TextInput) Blur() {
	t.Model.Blur()
}

func (t *TextInput) Reset() {
	t.SetValue("")
}

func (t *TextInput) OnClean() tea.Cmd {
	t.Reset()
	return nil
}

func (t *TextInput) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	return false, nil
}

func (t *TextInput) OnUp() (bool, tea.Cmd)   { return false, nil }
func (t *TextInput) OnDown() (bool, tea.Cmd) { return false, nil }

func (t *TextInput) Cursor() *tea.Cursor {
	return t.Model.Cursor()
}

func (t *TextInput) NextField() bool { return false }
func (t *TextInput) PrevField() bool { return false }

const (
	keyValueDefaultInputColumns = 30
	keyValueMinInputColumns     = 1
	keyValueLabelGapColumns     = 1
	keyValueInputRows           = 2
	keyValuePairGapRows         = 1
	keyValueMinPairs            = 1
	keyValueHeaderRows          = 1
	keyValueNumberSuffix        = ". "
)

const (
	keyValueKeyInput = iota
	keyValueValueInput
)

type KeyValueField struct {
	Key        *TextInput
	Value      *TextInput
	Theme      core.Theme
	focusedIdx int // 0 for Key, 1 for Value

	KeyLabel   string
	ValueLabel string
}

func NewKeyValueField(theme core.Theme, keyLabel, valueLabel string) *KeyValueField {
	key := NewTextInput(theme)
	key.Placeholder = "Key"
	key.SetWidth(keyValueDefaultInputColumns)
	val := NewTextInput(theme)
	val.Placeholder = "Value"
	val.SetWidth(keyValueDefaultInputColumns)

	return &KeyValueField{
		Key:        key,
		Value:      val,
		Theme:      theme,
		KeyLabel:   keyLabel,
		ValueLabel: valueLabel,
	}
}

func (k *KeyValueField) Update(msg tea.Msg) tea.Cmd {
	if k.focusedIdx == 0 {
		return k.Key.Update(msg)
	}
	return k.Value.Update(msg)
}

func (k *KeyValueField) View(focused bool) string {
	keyView := k.Key.View(focused && k.focusedIdx == 0)
	valView := k.Value.View(focused && k.focusedIdx == 1)

	keyStyle, valueStyle := k.Theme.LabelStyle, k.Theme.LabelStyle
	if focused {
		if k.focusedIdx == keyValueKeyInput {
			keyStyle = k.Theme.LabelFocused
		} else {
			valueStyle = k.Theme.LabelFocused
		}
	}
	keyLabel := keyStyle.Width(k.labelWidth()).Render(k.KeyLabel)
	valLabel := valueStyle.Width(k.labelWidth()).Render(k.ValueLabel)

	gap := strings.Repeat(" ", keyValueLabelGapColumns)
	return keyLabel + gap + keyView + "\n" + valLabel + gap + valView
}

func (k *KeyValueField) labelWidth() int {
	return max(lipgloss.Width(k.KeyLabel), lipgloss.Width(k.ValueLabel))
}

func (k *KeyValueField) Focus() tea.Cmd {
	if k.focusedIdx == 0 {
		return k.Key.Focus()
	}
	return k.Value.Focus()
}

func (k *KeyValueField) Blur() {
	k.Key.Blur()
	k.Value.Blur()
}

func (k *KeyValueField) Reset() {
	k.Key.Reset()
	k.Value.Reset()
	k.focusedIdx = 0
}

func (k *KeyValueField) OnClean() tea.Cmd {
	if k.focusedIdx == 0 {
		return k.Key.OnClean()
	}
	return k.Value.OnClean()
}

func (k *KeyValueField) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	return false, nil
}

func (k *KeyValueField) OnUp() (bool, tea.Cmd) {
	if k.PrevField() {
		return true, k.Focus()
	}
	return false, nil
}

func (k *KeyValueField) OnDown() (bool, tea.Cmd) {
	if k.NextField() {
		return true, k.Focus()
	}
	return false, nil
}

func (k *KeyValueField) NextField() bool {
	if k.focusedIdx == 0 {
		k.focusedIdx = 1
		k.Key.Blur()
		return true
	}
	return false
}

func (k *KeyValueField) PrevField() bool {
	if k.focusedIdx == 1 {
		k.focusedIdx = 0
		k.Value.Blur()
		return true
	}
	return false
}

func (k *KeyValueField) Cursor() *tea.Cursor {
	if k.focusedIdx == 0 {
		c := k.Key.Cursor()
		if c != nil {
			newC := *c
			newC.X += k.labelWidth() + keyValueLabelGapColumns
			return &newC
		}
	} else {
		c := k.Value.Cursor()
		if c != nil {
			newC := *c
			newC.X += k.labelWidth() + keyValueLabelGapColumns
			newC.Y += keyValueValueInput
			return &newC
		}
	}
	return nil
}

func (k *KeyValueField) Values() (string, string) {
	return k.Key.Value(), k.Value.Value()
}

type KeyValues struct {
	width  int
	Rows   []*KeyValueField
	Theme  core.Theme
	cursor int // row index

	Header     string
	KeyLabel   string
	ValueLabel string
}

func NewKeyValues(theme core.Theme, header, keyLabel, valueLabel string) *KeyValues {
	return &KeyValues{
		Rows:       []*KeyValueField{NewKeyValueField(theme, keyLabel, valueLabel)},
		Theme:      theme,
		Header:     header,
		KeyLabel:   keyLabel,
		ValueLabel: valueLabel,
	}
}

func (a *KeyValues) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, core.AddKeyValue()):
			return a.AddPair()
		case key.Matches(k, core.DeleteKeyValue()):
			return a.DeletePair()
		}
	}
	if len(a.Rows) == 0 {
		return nil
	}
	return a.Rows[a.cursor].Update(msg)
}

// AddPair appends a pair and moves focus to its key without changing existing drafts.
func (a *KeyValues) AddPair() tea.Cmd {
	a.Blur()
	a.Rows = append(a.Rows, NewKeyValueField(a.Theme, a.KeyLabel, a.ValueLabel))
	a.cursor = len(a.Rows) - 1
	a.resizeInputs()
	return a.Focus()
}

// DeletePair keeps one editable pair and focuses the nearest surviving key.
func (a *KeyValues) DeletePair() tea.Cmd {
	a.Blur()
	if len(a.Rows) <= keyValueMinPairs {
		a.Reset()
	} else {
		a.Rows = append(a.Rows[:a.cursor], a.Rows[a.cursor+1:]...)
		a.cursor = min(a.cursor, len(a.Rows)-1)
		a.Rows[a.cursor].focusedIdx = keyValueKeyInput
		a.resizeInputs()
	}
	return a.Focus()
}

func (a *KeyValues) View(focused bool) string {
	view := ""
	if a.Header != "" {
		view += a.Theme.HeaderStyle.Render(a.Header) + "\n"
	}
	var rows []string
	for i := range a.Rows {
		rows = append(rows, a.renderRow(i, focused && i == a.cursor))
	}
	return view + strings.Join(rows, strings.Repeat("\n", keyValuePairGapRows+1))
}

func (a *KeyValues) Focus() tea.Cmd {
	if len(a.Rows) > 0 {
		return a.Rows[a.cursor].Focus()
	}
	return nil
}

func (a *KeyValues) Blur() {
	for _, row := range a.Rows {
		row.Blur()
	}
}

func (a *KeyValues) Reset() {
	a.Rows = []*KeyValueField{NewKeyValueField(a.Theme, a.KeyLabel, a.ValueLabel)}
	a.cursor = 0
	a.resizeInputs()
}

func (a *KeyValues) OnClean() tea.Cmd {
	if len(a.Rows) > 0 {
		return a.Rows[a.cursor].OnClean()
	}
	return nil
}

func (a *KeyValues) OnToggle(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if len(a.Rows) > 0 {
		return a.Rows[a.cursor].OnToggle(msg)
	}
	return false, nil
}

func (a *KeyValues) OnUp() (bool, tea.Cmd) {
	if a.PrevField() {
		return true, a.Focus()
	}
	return false, nil
}

func (a *KeyValues) OnDown() (bool, tea.Cmd) {
	if a.NextField() {
		return true, a.Focus()
	}
	return false, nil
}

func (a *KeyValues) NextField() bool {
	if len(a.Rows) == 0 {
		return false
	}
	if a.Rows[a.cursor].NextField() {
		return true
	}
	if a.cursor < len(a.Rows)-1 {
		a.Rows[a.cursor].Blur()
		a.cursor++
		a.Rows[a.cursor].focusedIdx = 0
		return true
	}
	return false
}

func (a *KeyValues) PrevField() bool {
	if len(a.Rows) == 0 {
		return false
	}
	if a.Rows[a.cursor].PrevField() {
		return true
	}
	if a.cursor > 0 {
		a.Rows[a.cursor].Blur()
		a.cursor--
		a.Rows[a.cursor].focusedIdx = 1
		return true
	}
	return false
}

func (a *KeyValues) Cursor() *tea.Cursor {
	if len(a.Rows) == 0 {
		return nil
	}
	c := a.Rows[a.cursor].Cursor()
	if c != nil {
		newC := *c
		newC.X += a.numberWidth()
		// Match the rendered height of preceding rows and their separators.
		offset := 0
		for i := 0; i < a.cursor; i++ {
			offset += lipgloss.Height(a.renderRow(i, false)) + keyValuePairGapRows
		}
		if a.Header != "" {
			offset += keyValueHeaderRows
		}
		newC.Y += offset
		return &newC
	}
	return nil
}

func (a *KeyValues) Map() (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for _, row := range a.Rows {
		k, v := row.Values()
		if k == "" {
			if v != "" {
				return nil, fmt.Errorf("%s: key is required when a value is supplied", a.Header)
			}
			continue
		}

		var val interface{}
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			val = v
		}

		if _, exists := res[k]; exists {
			return nil, fmt.Errorf("duplicate argument key %q", k)
		}
		res[k] = val
	}
	return res, nil
}

func (a *KeyValues) renderRow(index int, focused bool) string {
	style := a.Theme.LabelStyle
	if focused {
		style = a.Theme.LabelFocused
	}
	number := fmt.Sprintf("%*d%s", a.numberWidth()-len(keyValueNumberSuffix), index+1, keyValueNumberSuffix)
	indent := strings.Repeat(" ", a.numberWidth())
	return style.Render(number) + strings.ReplaceAll(a.Rows[index].View(focused), "\n", "\n"+indent)
}

func (a *KeyValues) numberWidth() int {
	return len(strconv.Itoa(len(a.Rows))) + len(keyValueNumberSuffix)
}

// SetWidth receives the available content columns, excluding any parent scrollbar.
func (a *KeyValues) SetWidth(width int) {
	a.width = max(keyValueMinInputColumns, width)
	a.resizeInputs()
}

func (a *KeyValues) resizeInputs() {
	if a.width == 0 {
		return
	}
	for _, row := range a.Rows {
		inputWidth := max(keyValueMinInputColumns, a.width-a.numberWidth()-row.labelWidth()-keyValueLabelGapColumns)
		row.Key.SetWidth(inputWidth)
		row.Value.SetWidth(inputWidth)
	}
}

// FocusBoundary selects the first key or last value before the parent restores focus.
func (a *KeyValues) FocusBoundary(back bool) {
	a.Blur()
	a.cursor = 0
	a.Rows[a.cursor].focusedIdx = keyValueKeyInput
	if back {
		a.cursor = len(a.Rows) - 1
		a.Rows[a.cursor].focusedIdx = keyValueValueInput
	}
}

// ClickRow maps a content row to a pair and input, including the header and gaps.
func (a *KeyValues) ClickRow(y int) {
	a.Blur()
	if a.Header != "" {
		y -= keyValueHeaderRows
	}
	y = max(0, y)
	pairHeight := keyValueInputRows + keyValuePairGapRows
	a.cursor = min(len(a.Rows)-1, y/pairHeight)
	a.Rows[a.cursor].focusedIdx = min(keyValueValueInput, y%pairHeight)
}

func (t *TextInput) restoreHistory(from, to *[]textSnapshot) {
	if len(*from) == 0 {
		return
	}
	saved := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, textSnapshot{value: t.Value(), col: t.Position()})
	t.SetValue(saved.value)
	t.SetCursor(saved.col)
}
