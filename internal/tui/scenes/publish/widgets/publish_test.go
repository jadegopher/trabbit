package widgets

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/cursor"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func newTestPublisher() *PublishForm {
	p := NewPublishForm(core.DefaultTheme(), PublishTarget{VHost: "/", Exchange: app.DefaultExchange, RoutingKey: "orders"})
	p.SetSize(90, 25)
	// Existing navigation tests start in the target fields explicitly.
	p.focus = publishFields
	p.Focus()
	return p
}

func TestPublishFieldNavigationWraps(t *testing.T) {
	t.Parallel()
	for _, size := range []struct {
		name   string
		height int
	}{{"full", 25}, {"scrolling", 12}} {
		t.Run(size.name, func(t *testing.T) {
			t.Parallel()
			p := newTestPublisher()
			p.SetSize(90, size.height)
			p.headers.AddPair()
			p.headers.FocusBoundary(false)
			p.headers.Blur()
			fields := []struct {
				item  components.FormItem
				input *components.TextInput
			}{
				{p.vhost, nil},
				{p.exchange, p.exchange},
				{p.routing, p.routing},
				{p.delivery, nil},
				{p.encoding, nil},
				{p.properties, p.properties.Rows[0].Key},
				{p.properties, p.properties.Rows[0].Value},
				{p.headers, p.headers.Rows[0].Key},
				{p.headers, p.headers.Rows[0].Value},
				{p.headers, p.headers.Rows[1].Key},
				{p.headers, p.headers.Rows[1].Value},
			}

			for range fields {
				p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			}
			require.Same(t, p.vhost, p.fields.FocusedItem())
			require.Zero(t, p.fields.ScrollOffset(), "wrapping must reveal the top field")

			for index := len(fields) - 1; index >= 0; index-- {
				p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
				field := fields[index]
				require.Same(t, field.item, p.fields.FocusedItem())
				if field.input != nil {
					require.True(t, field.input.Focused())
					require.NotNil(t, p.fields.View().Cursor, "focused input must be visible")
				}
			}

			for _, field := range fields[1:] {
				p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
				require.Same(t, field.item, p.fields.FocusedItem())
				if field.input != nil {
					require.True(t, field.input.Focused())
				}
			}
			require.Equal(t, publishFields, p.focus)
			require.False(t, p.payload.Focused())
		})
	}
}

func TestPublishPayloadMouseDragAndCut(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue("hello world\nsecond line")
	layout := p.geometry()
	lines := strings.Split(ansi.Strip(p.View().Content), "\n")
	y := layout.editor.Min.Y
	x := lipgloss.Width(strings.SplitN(lines[y], "hello", 2)[0])
	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseMotionMsg{X: x + 5, Y: y, Button: tea.MouseLeft})
	require.Equal(t, "hello", p.payload.SelectedText())
	require.True(t, p.payload.HasSelection())
	p.Update(tea.MouseReleaseMsg{X: x + 6, Y: y, Button: tea.MouseLeft})
	require.Equal(t, "hello ", p.payload.SelectedText(), "release must apply its final coordinates")
	p.Update(tea.MouseMotionMsg{X: x + 9, Y: y})
	require.Equal(t, "hello ", p.payload.SelectedText())

	p.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	require.Equal(t, "world\nsecond line", p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, "hello world\nsecond line", p.payload.Value())

	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	require.False(t, p.payload.HasSelection(), "a plain click must clear selection")
	p.Update(tea.MouseClickMsg{X: layout.editorText.Max.X, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseMotionMsg{X: x, Y: y + 1, Button: tea.MouseLeft})
	require.False(t, p.payload.HasSelection(), "the scrollbar must not start a text selection")
}

func TestPublishClickAfterScrollingKeepsVisibleRows(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.payload.SetValue(strings.Repeat("hello world\n", 80))
	p.payload.MoveToBegin()
	layout := p.geometry()
	y := layout.editor.Min.Y + 1
	p.Update(tea.MouseWheelMsg{X: layout.editor.Min.X, Y: y, Button: tea.MouseWheelDown})
	before := ansi.Strip(p.View().Content)
	line := strings.Split(before, "\n")[y]
	x := lipgloss.Width(strings.SplitN(line, "hello", 2)[0])
	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	require.Equal(t, before, ansi.Strip(p.View().Content))
	require.False(t, p.payload.HasSelection())
	require.NotNil(t, p.View().Cursor)
	require.Equal(t, x, p.View().Cursor.X)
	require.Equal(t, y, p.View().Cursor.Y)
	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseMotionMsg{X: x + 5, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseReleaseMsg{X: x + 5, Y: y, Button: tea.MouseLeft})
	require.Equal(t, "hello", p.payload.SelectedText())
	require.Equal(t, before, ansi.Strip(p.View().Content))
}

func TestPublishPayloadDragOutsideAndFocusChange(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue("hello world\nsecond line")
	layout := p.geometry()
	lines := strings.Split(ansi.Strip(p.View().Content), "\n")
	y := layout.editor.Min.Y
	x := lipgloss.Width(strings.SplitN(lines[y], "hello", 2)[0])
	p.Update(tea.MouseClickMsg{X: x + 5, Y: y, Button: tea.MouseLeft})
	p.Update(tea.MouseMotionMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	p.Update(tea.MouseReleaseMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	require.Equal(t, "hello", p.payload.SelectedText())
	require.Equal(t, publishPayload, p.focus)
	p.Update(tea.MouseMotionMsg{X: x, Y: y + 1})
	require.Equal(t, "hello", p.payload.SelectedText(), "releasing outside must end the drag")

	p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.MouseMotionMsg{X: x + 5, Y: y, Button: tea.MouseLeft})
	require.False(t, p.payload.HasSelection(), "switching panels must end the drag")
}

func TestPublishFreeFormValuesAndSubmission(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.properties.Rows[0].Key.SetValue("content_type")
	p.properties.Rows[0].Value.SetValue("text/plain")
	p.headers.Rows[0].Key.SetValue("custom")
	p.headers.Rows[0].Value.SetValue("{unfinished [anything")
	p.payload.SetValue("line one\nline two")
	cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	request := cmd().(PublishRequest)
	require.Equal(t, "orders", request.Options.RoutingKey)
	require.Equal(t, "line one\nline two", request.Options.Payload)
	require.Equal(t, "text/plain", request.Options.Properties.Properties["content_type"])
	require.Equal(t, "{unfinished [anything", request.Options.Properties.Headers["custom"])
	require.Equal(t, 2, request.Options.Properties.Properties["delivery_mode"])
	require.Nil(t, p.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl}), "duplicate submit")
	p.Complete(request.ID+1, app.PublishResult{}, nil)
	require.True(t, p.Busy())
	p.Complete(request.ID, app.PublishResult{}, errors.New("offline"))
	require.False(t, p.Busy())
	require.Contains(t, p.View().Content, "offline")
	require.Equal(t, request.Options.Payload, p.payload.Value())
	retry := p.submit()().(PublishRequest)
	p.Complete(request.ID, app.PublishResult{}, nil)
	require.True(t, p.Busy(), "old response cannot finish retry")
	p.Complete(retry.ID, app.PublishResult{Routed: false}, nil)
	require.Contains(t, p.View().Content, "NOT routed")
}

func TestPublishPayloadSelectionPasteAndFormat(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.focus = publishPayload
	p.focusCurrent()
	original := strings.Repeat("日本語 payload\n", 150) + "last"
	p.Update(tea.PasteMsg{Content: original})
	require.Equal(t, original, p.payload.Value(), "paste must exceed the viewport and old 99-line limit")
	p.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	require.Equal(t, original, p.payload.SelectedText())
	p.Update(tea.PasteMsg{Content: `{"large":9007199254740993,"arr":[1,2]}`})
	p.beautify()
	require.NoError(t, p.err)
	require.Contains(t, p.payload.Value(), "9007199254740993")
	require.Contains(t, p.payload.Value(), "\n")
	p.payload.SetValue("{broken")
	p.beautify()
	require.Error(t, p.err)
	require.Equal(t, "{broken", p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: '[', Text: "["})
	require.Contains(t, p.payload.Value(), "[")
	p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.Contains(t, p.payload.Value(), "\n")
	p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	require.IsType(t, PublishExit{}, cmd())
}

func TestPublishPanelSizeScrollAndFocus(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue(strings.Repeat("界 👋 long content ", 200))
	p.properties.Rows[0].Key.SetValue(strings.Repeat("界", 100))
	p.properties.Rows[0].Value.SetValue(strings.Repeat("value", 100))
	for _, size := range []tea.WindowSizeMsg{{Width: 90, Height: 25}, {Width: 29, Height: 12}, {Width: 0, Height: 0}, {Width: 100, Height: 30}} {
		p.SetSize(size.Width, size.Height)
		if size.Width == 0 {
			require.Empty(t, p.View().Content)
			continue
		}
		for i := 0; i < 20; i++ {
			p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			view := p.View()
			require.Equal(t, size.Width, lipgloss.Width(view.Content))
			require.Equal(t, size.Height, lipgloss.Height(view.Content))
			for row, line := range strings.Split(ansi.Strip(view.Content), "\n") {
				require.Equal(t, size.Width, lipgloss.Width(line))
				if row >= p.geometry().fields.content.Min.Y && row < p.geometry().fields.content.Max.Y {
					require.True(t, strings.HasPrefix(line, "│"))
					require.True(t, strings.HasSuffix(line, "│"))
				}
			}
			if view.Cursor != nil {
				require.Greater(t, view.Cursor.X, 0)
				require.Less(t, view.Cursor.X, size.Width+1)
				require.Less(t, view.Cursor.Y, size.Height+1)
			}
		}
	}
}

func TestPublishArrowNavigationAndMouseFocus(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p.focusCurrent()
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Same(t, p.delivery, p.fields.FocusedItem())
	p.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	require.Equal(t, 0, p.delivery.Index)
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Same(t, p.properties, p.fields.FocusedItem())
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.True(t, p.properties.Rows[0].Value.Focused())
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.True(t, p.properties.Rows[0].Key.Focused())
	p.Update(tea.MouseClickMsg{X: 80, Y: 5, Button: tea.MouseLeft})
	require.Equal(t, publishPayload, p.focus)
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	require.Equal(t, publishPayload, p.focus)
	p.Update(tea.PasteMsg{Content: "first\nsecond"})
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, 0, p.payload.Line())
	require.Same(t, p.properties, p.fields.FocusedItem())
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	require.Equal(t, publishFields, p.focus)
	require.Same(t, p.properties, p.fields.FocusedItem())
}

func TestPublishStandardSelectionKeys(t *testing.T) {
	t.Parallel()
	for _, mod := range []tea.KeyMod{tea.ModCtrl, tea.ModSuper, tea.ModMeta} {
		p := newTestPublisher()
		p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		p.payload.SetValue("one\ntwo")
		p.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
		require.Equal(t, "one\ntwo", p.payload.SelectedText())
		copyMod := mod
		if mod == tea.ModCtrl {
			copyMod |= tea.ModShift
		}
		require.NotNil(t, p.Update(tea.KeyPressMsg{Code: 'c', Mod: copyMod}))
		p.Update(tea.PasteMsg{Content: "replaced"})
		require.Equal(t, "replaced", p.payload.Value())
		p.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
		require.Equal(t, "d", p.payload.SelectedText())
		p.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
		require.Equal(t, "ed", p.payload.SelectedText())
		p.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		require.Equal(t, "replac", p.payload.Value())
	}
}

func TestPublishEncodingAndPublishShortcut(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.encoding.Index = 1
	p.payload.SetValue("invalid base64")
	require.Nil(t, p.submit())
	require.ErrorContains(t, p.err, "invalid base64")
	p.payload.SetValue("aGVsbG8=")
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, publishFields, p.focus)
	require.True(t, p.fields.Focused())
	require.Same(t, p.headers, p.fields.FocusedItem())
	cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	request := cmd().(PublishRequest)
	require.Equal(t, "base64", request.Options.PayloadEncoding)
	require.Equal(t, "aGVsbG8=", request.Options.Payload)
}

func TestPublishColumnsScrollIndependently(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetSize(120, 14)
	p.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	offset := p.fields.ScrollOffset()
	require.Greater(t, offset, 0)
	// Cursor ticks and rendering cannot pull the field viewport back to its cursor.
	p.Update(struct{}{})
	p.View()
	require.Equal(t, offset, p.fields.ScrollOffset())
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.PasteMsg{Content: strings.Repeat("payload line\n", 120) + "end"})
	p.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.Less(t, p.payload.Line(), 120)
	require.Equal(t, offset, p.fields.ScrollOffset())
	payloadLine := p.payload.Line()
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, payloadLine, p.payload.Line())
}

func TestPublishCommandShiftSelectionAcrossWrappedLines(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetSize(90, 15)
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	value := strings.Repeat("界 hello ", 15) + "\nsecond\nlast"
	p.payload.SetValue(value)
	p.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModSuper | tea.ModShift})
	require.Equal(t, "last", p.payload.SelectedText())
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModSuper | tea.ModShift})
	require.Equal(t, value, p.payload.SelectedText())
	p.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	require.Equal(t, strings.TrimPrefix(value, "界"), p.payload.SelectedText())
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModSuper | tea.ModShift})
	require.False(t, p.payload.HasSelection())
	p.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl})
	p.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModSuper | tea.ModShift})
	require.Equal(t, strings.Split(value, "\n")[0], p.payload.SelectedText())
	p.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModSuper | tea.ModShift})
	require.False(t, p.payload.HasSelection())
	p.Update(tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl | tea.ModShift})
	require.Equal(t, value, p.payload.SelectedText())
}

func TestPublishEditableTargetIsCapturedAtSubmission(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetVHosts([]string{"/other", "/later"}, nil)
	p.vhost.Next()
	p.exchange.SetValue("events")
	request := p.submit()().(PublishRequest)
	require.Equal(t, PublishTarget{VHost: "/other", Exchange: "events", RoutingKey: "orders"}, request.Target)
	p.vhost.Next()
	require.Equal(t, "/other", request.Target.VHost)
}

func TestPublishClickFieldsAndNestedValues(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetSize(160, 45)
	p.Update(tea.MouseClickMsg{X: 5, Y: 3, Button: tea.MouseLeft})
	require.Same(t, p.exchange, p.fields.FocusedItem())
	// Use the rendered row location so section spacing remains part of hit testing.
	lines := strings.Split(ansi.Strip(p.View().Content), "\n")
	for y, line := range lines {
		if strings.Contains(line, "Value:") {
			p.Update(tea.MouseClickMsg{X: 10, Y: y, Button: tea.MouseLeft})
			require.True(t, p.properties.Rows[0].Value.Focused())
			return
		}
	}
	t.Fatal("missing properties value")
}

func TestPublishKeyValuePairActions(t *testing.T) {
	t.Parallel()
	for _, section := range []string{"Properties", "Headers"} {
		t.Run(section, func(t *testing.T) {
			t.Parallel()
			p := newTestPublisher()
			pairs := p.properties
			if section == "Headers" {
				pairs = p.headers
			}
			for range p.fields.Fields {
				if p.fields.FocusedItem() == pairs {
					break
				}
				p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			}
			require.Same(t, pairs, p.fields.FocusedItem())
			hints := ansi.Strip(p.theme.RenderHelp(p))
			require.Contains(t, hints, "F2 add pair")
			require.Contains(t, hints, "F3 delete pair")

			p.Update(tea.PasteMsg{Content: "first"})
			require.Len(t, pairs.Rows, 1)
			p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			p.Update(tea.PasteMsg{Content: "first value"})
			p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
			require.Len(t, pairs.Rows, 2)
			require.True(t, pairs.Rows[1].Key.Focused())
			p.Update(tea.PasteMsg{Content: "second"})
			p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			p.Update(tea.PasteMsg{Content: "second value"})

			for _, size := range []tea.WindowSizeMsg{{Width: 90, Height: 12}, {Width: 120, Height: 30}} {
				p.SetSize(size.Width, size.Height)
				view := p.fields.View()
				require.NotNil(t, view.Cursor)
				lines := strings.Split(ansi.Strip(view.Content), "\n")
				require.Contains(t, lines[view.Cursor.Y], "Value:")
				// Mouse input must select the same value after scrolling and resizing.
				bounds := p.geometry().fields.content
				p.Update(tea.MouseClickMsg{X: bounds.Min.X + view.Cursor.X, Y: bounds.Min.Y + view.Cursor.Y, Button: tea.MouseLeft})
				require.True(t, pairs.Rows[1].Value.Focused())
			}

			p.Update(tea.KeyPressMsg{Code: tea.KeyF3})
			require.Len(t, pairs.Rows, 1)
			require.Equal(t, "first", pairs.Rows[0].Key.Value())
			require.Equal(t, "first value", pairs.Rows[0].Value.Value())
			p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			require.NotContains(t, ansi.Strip(p.theme.RenderHelp(p)), "add pair")
			p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
			p.Update(tea.KeyPressMsg{Code: tea.KeyF3})
			require.Len(t, pairs.Rows, 1, "pair actions only apply in the field column")
			p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			require.NotNil(t, p.submit())
			p.Update(tea.KeyPressMsg{Code: tea.KeyF2})
			p.Update(tea.KeyPressMsg{Code: tea.KeyF3})
			require.Len(t, pairs.Rows, 1, "busy forms retain their submitted draft")
		})
	}
}

func TestPublishUndoFormatCutAndRedo(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.PasteMsg{Content: `{"a":1}`})
	p.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	require.Contains(t, p.payload.Value(), "\n")
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, `{"a":1}`, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl | tea.ModShift})
	require.Contains(t, p.payload.Value(), "\n")
	p.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	p.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	require.Empty(t, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Contains(t, p.payload.Value(), `"a": 1`)
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, `{"a":1}`, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Empty(t, p.payload.Value())
}

func TestPublishWheelAndSelectionKeepPanelsContained(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	value := strings.Repeat("界 wide payload\n", 120) + "last"
	p.Update(tea.PasteMsg{Content: value})
	row, col := p.payload.Line(), p.payload.Column()
	beforeScroll := p.View().Content
	left := p.geometry().fields.bounds.Dx()
	p.Update(tea.MouseWheelMsg{X: left + 4, Y: 6, Button: tea.MouseWheelUp})
	require.NotEqual(t, beforeScroll, p.View().Content)
	scrolled := p.View().Content
	for _, event := range []tea.Msg{
		cursor.BlinkMsg{},
		tea.MouseMotionMsg{X: left + 4, Y: 6},
	} {
		p.Update(PublishEvent{Form: p, Value: event})
		require.Equal(t, scrolled, p.View().Content, "background events must preserve wheel browsing")
		require.Nil(t, p.View().Cursor)
	}
	require.Equal(t, row, p.payload.Line())
	require.Equal(t, col, p.payload.Column())
	p.Update(tea.MouseWheelMsg{X: left + 4, Y: 6, Button: tea.MouseWheelUp})
	require.Contains(t, p.View().Content, "▐")
	require.Nil(t, p.View().Cursor)
	for _, size := range []tea.WindowSizeMsg{{Width: 90, Height: 25}, {Width: 29, Height: 12}, {Width: 0, Height: 0}, {Width: 100, Height: 30}} {
		p.SetSize(size.Width, size.Height)
		if size.Width == 0 {
			require.Empty(t, p.View().Content)
			continue
		}
		p.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
		p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
		require.Equal(t, value, p.payload.Value())
		require.False(t, p.payload.HasSelection())
		view := ansi.Strip(p.View().Content)
		lines := strings.Split(view, "\n")
		require.Len(t, lines, size.Height)
		left := p.geometry().fields.bounds.Dx()
		for y, line := range lines {
			require.Equal(t, size.Width, lipgloss.Width(line))
			if y >= p.geometry().fields.content.Min.Y && y < p.geometry().fields.content.Max.Y {
				require.Equal(t, "││", ansi.Cut(line, left-1, left+1))
				require.True(t, strings.HasPrefix(line, "│"))
				require.True(t, strings.HasSuffix(line, "│"))
			}
		}
	}
}

func TestPublishNoticeAndSharedHints(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetSize(140, 30)
	request := p.submit()().(PublishRequest)
	p.Complete(request.ID, app.PublishResult{Routed: true}, nil)
	lines := strings.Split(ansi.Strip(p.View().Content), "\n")
	require.Contains(t, lines[0], "Published • routed")
	require.NotContains(t, lines[0], "Publish message")
	require.True(t, strings.HasPrefix(lines[0], "│"))
	require.True(t, strings.HasSuffix(lines[0], "│"))
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	hints := ansi.Strip(p.theme.RenderHelp(p))
	require.NotContains(t, hints, "ctrl+k")
	require.NotContains(t, hints, "ctrl+u")
	require.NotContains(t, hints, "panel")
	require.NotContains(t, hints, "exit")
	require.NotContains(t, hints, "ctrl+b")
	require.NotContains(t, hints, "ctrl+r")
	require.NotContains(t, hints, "undo")
	require.NotContains(t, hints, "redo")
	require.Contains(t, ansi.Strip(p.View().Content), "ctrl+g help")
	p.SetSize(90, 25)
	require.Contains(t, ansi.Strip(p.View().Content), "ctrl+g help")
	require.NotContains(t, hints, "copy")
	require.NotContains(t, hints, "Enter action")
}

func TestPublishNoticeLifecycle(t *testing.T) {
	t.Parallel()
	for _, outcome := range []struct {
		name   string
		routed bool
		err    error
		want   string
		kind   core.NoticeKind
	}{
		{name: "success", routed: true, want: "Published • routed", kind: core.NoticeSuccess},
		{name: "unrouted", want: "no queue matched the routing key.", kind: core.NoticeWarning},
		{name: "failure", err: errors.New("connection refused"), want: "Publish failed: connection refused", kind: core.NoticeError},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			t.Parallel()
			p := newTestPublisher()
			p.focus = publishPayload
			p.focusCurrent()
			before := p.geometry()
			request := p.submit()().(PublishRequest)
			require.Contains(t, ansi.Strip(p.View().Content), "Publishing...")
			p.Complete(request.ID, app.PublishResult{Routed: outcome.routed}, outcome.err)
			require.Contains(t, ansi.Strip(p.View().Content), outcome.want)
			require.Equal(t, before, p.geometry(), "notifications must not move controls")
			if outcome.err == nil {
				require.Equal(t, outcome.kind, p.statusKind)
				p.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
				require.Contains(t, ansi.Strip(p.View().Content), outcome.want, "editing must preserve the publication result")
			}
			for _, width := range []int{29, 90, 140} {
				p.SetSize(width, 25)
				view := p.View()
				require.Equal(t, 25, lipgloss.Height(view.Content))
				for _, line := range strings.Split(view.Content, "\n") {
					require.Equal(t, width, lipgloss.Width(line))
				}
			}
		})
	}
}

func TestPublishNoticeExpiresAndRejectsOldTimer(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	request := p.submit()().(PublishRequest)
	cmd := p.Complete(request.ID, app.PublishResult{Routed: true}, nil)
	expired := cmd()
	if commands, ok := expired.(tea.BatchMsg); ok {
		expired = commands[len(commands)-1]()
	}
	p.Update(expired)
	require.Empty(t, p.status)
	require.Contains(t, strings.Split(ansi.Strip(p.View().Content), "\n")[0], "Publish message")
	p.Update(expired)
	require.Empty(t, p.status, "duplicate expiry must be harmless")

	next := p.submit()().(PublishRequest)
	p.Complete(next.ID, app.PublishResult{Routed: true}, nil)
	p.Update(expired)
	require.NotEmpty(t, p.status, "an old expiry must not dismiss a newer result")
	p.Blur()
	require.Empty(t, p.status, "hidden forms must cancel the notice")
}

func TestPublishBeautifyDoesNotShowNotice(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue(`{"a":1}`)
	p.beautify()
	require.NoError(t, p.err)
	require.Contains(t, p.payload.Value(), "\n")
	require.Empty(t, p.status)
	require.Contains(t, strings.Split(ansi.Strip(p.View().Content), "\n")[0], "Publish message")
}

func TestPublishCompactFieldsAndExitFallback(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	lines := strings.Split(ansi.Strip(p.fields.View().Content), "\n")
	for row, label := range []string{"Vhost", "Exchange", "Routing key", "Delivery mode", "Encoding"} {
		require.Contains(t, lines[row], label)
	}

	for _, press := range []tea.KeyPressMsg{
		{Code: tea.KeyEscape},
		{Code: 'q', Mod: tea.ModCtrl},
	} {
		cmd := p.Update(press)
		require.NotNil(t, cmd)
		require.Equal(t, PublishExit{Form: p}, cmd())
	}

	require.Nil(t, p.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}))
	require.False(t, p.Busy())
}

func TestPublishClickUsesScrolledFieldPosition(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetSize(90, 12)
	p.fields.Scroll(1)
	offset := p.fields.ScrollOffset()
	require.Positive(t, offset)
	p.Update(tea.MouseClickMsg{X: 5, Y: 3, Button: tea.MouseLeft})
	require.Same(t, p.routing, p.fields.FocusedItem(), "row 1 of the scrolled column is routing key")
	p.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Equal(t, "ordersx", p.routing.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, "orders", p.routing.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl | tea.ModShift})
	require.Equal(t, "ordersx", p.routing.Value())
}

func TestPublishScrollbarsStayBesidePanelBorders(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue(strings.Repeat("界 payload\n", 100))
	for _, width := range []int{60, 90, 180, 60} {
		p.SetSize(width, 14)
		left := p.geometry().fields.bounds.Dx()
		for _, delta := range []int{0, 10000, -10000} {
			p.fields.Scroll(delta)
			p.payload.Scroll(delta)
			lines := strings.Split(ansi.Strip(p.View().Content), "\n")
			fields, payload := 0, 0
			for _, line := range lines {
				require.Equal(t, width, lipgloss.Width(line))
				for x := 0; x < width; x++ {
					if ansi.Cut(line, x, x+1) != "▐" {
						continue
					}
					require.Equal(t, "│", ansi.Cut(line, x+1, x+2))
					switch x {
					case left - 2:
						fields++
					case width - 2:
						payload++
					default:
						t.Fatalf("scrollbar in content column %d", x)
					}
				}
			}
			require.Positive(t, fields)
			require.Positive(t, payload)
		}
	}
}

func TestPublishShortcutsWithoutButtons(t *testing.T) {
	t.Parallel()
	for _, panel := range []struct {
		name  string
		focus int
	}{{"fields", publishFields}, {"payload", publishPayload}} {
		t.Run(panel.name, func(t *testing.T) {
			t.Parallel()
			for _, shortcut := range []struct {
				name string
				key  tea.KeyPressMsg
			}{
				{"f5", tea.KeyPressMsg{Code: tea.KeyF5}},
				{"ctrl+enter", tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl}},
			} {
				t.Run(shortcut.name, func(t *testing.T) {
					t.Parallel()
					p := newTestPublisher()
					p.focus = panel.focus
					p.focusCurrent()
					p.payload.SetValue(`{"a":1}`)

					p.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
					require.NoError(t, p.err)
					require.JSONEq(t, `{"a":1}`, p.payload.Value())
					if panel.focus == publishPayload {
						require.Contains(t, p.payload.Value(), "\n")
					}
					require.Equal(t, panel.focus, p.focus)
					hints := ansi.Strip(p.theme.RenderHelp(p))
					require.NotContains(t, hints, "alt+enter")
					require.Contains(t, hints, "F5 publish")
					require.NotContains(t, hints, "ctrl+enter")
					require.NotContains(t, hints, "super+enter")
					require.NotContains(t, hints, "ctrl+b")
					require.NotContains(t, hints, "ctrl+r")

					cmd := p.Update(shortcut.key)
					require.NotNil(t, cmd)
					request := cmd().(PublishRequest)
					require.Equal(t, p.payload.Value(), request.Options.Payload)
					require.True(t, p.Busy())
					require.Nil(t, p.Update(shortcut.key), "busy form must reject repeated publication")
				})
			}
		})
	}
}

func TestPublishRemovedButtonsHaveNoFocusOrClickActions(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Equal(t, publishFields, p.focus)
	require.True(t, p.fields.Focused())
	require.Same(t, p.headers, p.fields.FocusedItem())

	p.SetSize(120, 15)
	p.payload.SetValue(`{"a":1}`)
	body := p.geometry().fields.content.Dy()
	for _, y := range []int{body, body + 1} {
		p.Update(tea.MouseClickMsg{X: 5, Y: y, Button: tea.MouseLeft})
		require.Equal(t, publishFields, p.focus)
		require.False(t, p.Busy(), "former button rows must not publish")
		require.Equal(t, `{"a":1}`, p.payload.Value(), "former button rows must not format")
	}
	require.Equal(t, body, lipgloss.Height(p.fields.View().Content))
	view := ansi.Strip(p.View().Content)
	require.NotContains(t, view, "Beautify")
	require.NotContains(t, view, "\nPublish ")
}

func TestPublishInlineTargetsAndEncodingOrder(t *testing.T) {
	t.Parallel()
	p := NewPublishForm(core.DefaultTheme(), PublishTarget{VHost: "/team", Exchange: "events", RoutingKey: "orders"})
	p.SetSize(140, 32)
	p.Focus()
	require.Equal(t, publishPayload, p.focus)
	require.True(t, p.payload.Focused())
	p.Update(tea.PasteMsg{Content: "ready to publish"})
	require.Equal(t, "ready to publish", p.payload.Value(), "input must go directly to the payload on opening")
	lines := strings.Split(ansi.Strip(p.View().Content), "\n")
	for _, field := range []struct{ label, value string }{
		{"Vhost", "/team"}, {"Exchange", "events"}, {"Routing key", "orders"},
	} {
		found := false
		for _, line := range lines {
			if strings.Contains(line, field.label) {
				require.Contains(t, line, field.value, "label and preset must share a row")
				found = true
			}
		}
		require.True(t, found, "missing field %s", field.label)
	}
	view := ansi.Strip(p.View().Content)
	require.Less(t, strings.Index(view, "Encoding"), strings.Index(view, "Headers"))

	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	cursor := p.View().Cursor
	require.NotNil(t, cursor)
	require.Contains(t, lines[cursor.Y], "Routing key")
	require.Equal(t, "s", ansi.Cut(lines[cursor.Y], cursor.X-1, cursor.X))
}

func TestPublishVHostUsesFilterSelectorBehavior(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.SetVHosts([]string{"/", "/team", "/team"}, nil)
	require.Equal(t, []string{"/", "/team"}, p.vhost.Options)
	p.Update(tea.PasteMsg{Content: "free input"})
	p.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Equal(t, "/", p.Target().VHost)
	for _, press := range []tea.KeyPressMsg{
		{Code: tea.KeyRight}, {Code: tea.KeyLeft}, {Code: tea.KeyEnter}, {Code: tea.KeyEnter},
	} {
		before := p.Target().VHost
		p.Update(press)
		require.NotEqual(t, before, p.Target().VHost)
		require.Contains(t, p.vhost.Options, p.Target().VHost)
	}
	p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.Equal(t, "/team", p.Target().VHost)
	p.SetVHosts([]string{"/", "/other"}, nil)
	require.Equal(t, "/team", p.Target().VHost, "refresh must preserve selection")

	request := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})().(PublishRequest)
	require.Equal(t, "/team", request.Target.VHost)
	require.Equal(t, app.DefaultExchange, p.exchange.Value())
}

func TestPublishMouseMatchesRenderedEditorAfterResize(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue("first\nsecond\nthird")
	for _, size := range []tea.WindowSizeMsg{{Width: 90, Height: 25}, {Width: 29, Height: 12}, {Width: 180, Height: 30}} {
		p.SetSize(size.Width, size.Height)
		layout := p.geometry()
		y := layout.editor.Min.Y + 1
		lines := strings.Split(ansi.Strip(p.View().Content), "\n")
		x := lipgloss.Width(strings.SplitN(lines[y], "second", 2)[0])
		require.Equal(t, "s", ansi.Cut(lines[y], x, x+1), "mouse origin must match the rendered second line")

		p.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
		require.Equal(t, publishPayload, p.focus)
		require.Equal(t, 1, p.payload.Line())
		require.Zero(t, p.payload.Column())

		p.Update(tea.MouseClickMsg{X: layout.fields.bounds.Min.X, Y: y, Button: tea.MouseLeft})
		require.Equal(t, publishPayload, p.focus, "border clicks must not focus a field")
	}
}

func TestPublishHelpUsesEditorBindingsAndPreservesDraft(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.payload.SetValue("draft")
	cmd := p.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	open := cmd().(core.OpenDialog)
	require.Equal(t, "draft", p.payload.Value())
	open.Model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := ansi.Strip(open.Model.View().Content)
	require.Contains(t, view, "Ctrl+B / Cmd+B beautify")
	require.Contains(t, view, "Ctrl+R clear")
	require.Contains(t, view, "undo")
	require.Contains(t, view, "Shift+Cmd+Z")
	require.NotContains(t, view, "super+")
	require.NotContains(t, view, "meta+")
	require.NotContains(t, view, "publish")
	require.NotContains(t, view, "exit")
	require.NotContains(t, view, "ctrl+b character backward")
	require.Nil(t, p.Update(tea.KeyPressMsg{Code: '?', Text: "?"}))
	require.Contains(t, p.payload.Value(), "?", "question marks must be ordinary editor input")
	require.NotContains(t, ansi.Strip(p.theme.RenderHelp(p)), "undo")
}

func TestClearPayloadShortcutIsFocusedUndoableAndBlockedWhilePublishing(t *testing.T) {
	t.Parallel()
	p := newTestPublisher()
	p.payload.SetValue(strings.Repeat("payload line\n", 40))
	original := p.payload.Value()
	target := p.Target()

	// Other fields must not accidentally clear a payload that is out of focus.
	p.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.Equal(t, original, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	p.payload.Scroll(20)
	p.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.Empty(t, p.payload.Value())
	require.False(t, p.payload.HasSelection())
	require.Equal(t, target, p.Target())
	require.NotContains(t, ansi.Strip(p.theme.RenderHelp(p)), "ctrl+r")

	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	require.Equal(t, original, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl})
	require.Empty(t, p.payload.Value())
	p.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	request := p.submit()().(PublishRequest)
	p.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	require.True(t, p.Pending(request.ID))
	require.Equal(t, original, p.payload.Value())
}
