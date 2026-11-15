package core

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

const (
	DefaultModalWidth = 60
	BorderOverhead    = 2
)

// Palette values live here; renderers use Theme fields and semantic styles.
const (
	paletteOrange                = "#FF6600"
	paletteCyan                  = "#00ADD8"
	paletteRed                   = "#FF0000"
	paletteWhite                 = "#FFFFFF"
	paletteBlack                 = "#000000"
	paletteInactiveLight         = "244"
	paletteInactiveDark          = "240"
	palettePlaceholderLight      = "248"
	palettePlaceholderDark       = paletteInactiveLight
	paletteSuccessLight          = "#008000"
	paletteSuccessDark           = "#00FF00"
	palettePendingLight          = "#8B8000"
	palettePendingDark           = "#FFFF00"
	paletteChartReady            = "#edc240"
	paletteChartUnacked          = "#afd8f8"
	paletteChartTotal            = "#cb4b4b"
	paletteChartPublish          = "#4bb2c5"
	paletteChartDeliverGet       = "#eaa228"
	paletteChartAck              = "#579575"
	paletteChartRedeliver        = "#c5b47f"
	paletteChartReturnUnroutable = "#953579"
	paletteChartDeliver          = "#4b5de4"
	paletteChartGet              = "#d8b83f"
	paletteChartDeliverNoAck     = "#ff5800"
	paletteChartGetNoAck         = "#0085cc"
)

type Theme struct {
	Primary   color.Color
	Secondary color.Color
	Warning   color.Color
	Inactive  color.Color
	Success   color.Color
	Error     color.Color
	Pending   color.Color

	BaseStyle    lipgloss.Style
	FocusedStyle lipgloss.Style

	FocusedText lipgloss.Style
	BlurredText lipgloss.Style
	KeyStyle    lipgloss.Style
	DescStyle   lipgloss.Style

	HeaderStyle    lipgloss.Style
	SubHeaderStyle lipgloss.Style
	LabelStyle     lipgloss.Style
	LabelFocused   lipgloss.Style
	FooterStyle    lipgloss.Style

	PlaceholderStyle lipgloss.Style

	SelectedStyle           lipgloss.Style
	RecordRailStyle         lipgloss.Style
	RecordRailSelectedStyle lipgloss.Style

	TabLabelStyle    lipgloss.Style
	TabKeyStyle      lipgloss.Style
	TabSelectedStyle lipgloss.Style

	TableHeaderStyle lipgloss.Style
	RefreshTextStyle lipgloss.Style

	PlainStyle lipgloss.Style
	TableStyle lipgloss.Style

	SuccessStyle lipgloss.Style
	ErrorStyle   lipgloss.Style

	TextColor   color.Color
	BorderColor color.Color

	ChartReady   color.Color
	ChartUnacked color.Color
	ChartTotal   color.Color

	ChartPublish          color.Color
	ChartDeliverGet       color.Color
	ChartAck              color.Color
	ChartRedeliver        color.Color
	ChartReturnUnroutable color.Color
	ChartDeliver          color.Color
	ChartGet              color.Color
	ChartDeliverNoAck     color.Color
	ChartGetNoAck         color.Color
}

func (t Theme) NewChart(width, height int, options ...timeserieslinechart.Option) timeserieslinechart.Model {
	axisStyle := lipgloss.NewStyle().Foreground(t.Inactive)
	defaultOptions := []timeserieslinechart.Option{
		timeserieslinechart.WithXLabelFormatter(
			func(i int, v float64) string {
				return time.Unix(int64(v), 0).Format("15:04:05")
			},
		),
		timeserieslinechart.WithYLabelFormatter(
			func(i int, v float64) string {
				return FormatValue(v)
			},
		),
		timeserieslinechart.WithAxesStyles(axisStyle, axisStyle),
	}
	return timeserieslinechart.New(width, height, append(defaultOptions, options...)...)
}

func FormatValue(v float64) string {
	if v == 0 {
		return "0"
	}
	if v < 1 {
		return fmt.Sprintf("%.2f", v)
	}
	if v < 1000 {
		return formatSuffix(v, "")
	}
	if v < 1000000 {
		return formatSuffix(v/1000, "k")
	}
	if v < 1000000000 {
		return formatSuffix(v/1000000, "m")
	}
	return formatSuffix(v/1000000000, "b")
}

func FormatBytes(v int64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div, exp := int64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(v)/float64(div), "KMGTPE"[exp])
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func FormatRateValue(v float64) string {
	if v == 0 {
		return "0/s"
	}
	return FormatValue(v) + "/s"
}

func formatSuffix(v float64, suffix string) string {
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	return s + suffix
}

func (t Theme) RenderLegendItem(dotColor color.Color, name, value string, enabled bool) string {
	dotStyle := lipgloss.NewStyle().Foreground(dotColor)
	textStyle := lipgloss.NewStyle()
	if !enabled {
		dotStyle = lipgloss.NewStyle().Foreground(t.Inactive)
		textStyle = textStyle.Foreground(t.Inactive)
	}

	return fmt.Sprintf("%s\u00a0%s\u00a0%s", dotStyle.Render("●"), textStyle.Render(name+":"), textStyle.Render(value))
}

func (t Theme) TextInputStyles() textinput.Styles {
	s := textinput.DefaultStyles(true)

	s.Focused.Prompt = t.LabelStyle
	s.Focused.Text = lipgloss.NewStyle().Foreground(t.TextColor)
	s.Focused.Placeholder = t.PlaceholderStyle

	s.Blurred.Prompt = t.LabelStyle
	s.Blurred.Text = t.BlurredText
	s.Blurred.Placeholder = t.PlaceholderStyle

	return s
}

// OptionalFieldStyle keeps empty optional groups quiet, even when focused.
func (t Theme) OptionalFieldStyle(populated bool) lipgloss.Style {
	if populated {
		return t.PlainStyle.Foreground(t.TextColor)
	}
	return t.BlurredText
}

// TextAreaStyles keeps document text legible when focus moves to another panel.
func (t Theme) TextAreaStyles() textarea.Styles {
	s := textarea.DefaultStyles(true)
	state := textarea.StyleState{
		Base: t.PlainStyle, Text: t.PlainStyle.Foreground(t.TextColor),
		CursorLine: t.PlainStyle.Foreground(t.TextColor),
		LineNumber: t.BlurredText, CursorLineNumber: t.BlurredText,
		EndOfBuffer: t.BlurredText, Placeholder: t.PlaceholderStyle,
		Prompt: t.BlurredText, Selection: t.SelectedStyle,
	}
	s.Focused, s.Blurred = state, state
	s.Cursor.Color = t.TextColor
	return s
}

func (t Theme) NewTextInput() textinput.Model {
	ti := textinput.New()
	ti.SetWidth(30)
	ti.CharLimit = 1000000
	ti.SetStyles(t.TextInputStyles())
	ti.Prompt = ""
	ti.SetVirtualCursor(false)
	return ti
}

func (t Theme) PlaceInModal(v tea.View, modalWidth int, borderColor color.Color) tea.View {
	style := t.ModalStyle(modalWidth, borderColor)
	v.Content = style.Render(v.Content)

	if v.Cursor != nil {
		c := *v.Cursor
		c.X += style.GetPaddingLeft() + style.GetBorderLeftSize()
		c.Y += style.GetPaddingTop() + style.GetBorderTopSize()
		v.Cursor = &c
	}

	return v
}

func (t Theme) Center(v tea.View, width, height, modalWidth int, borderColor color.Color) tea.View {
	v = t.PlaceInModal(v, modalWidth, borderColor)

	if width > 0 && height > 0 {
		contentWidth, contentHeight := lipgloss.Size(v.Content)
		x := (width - contentWidth) / 2
		y := (height - contentHeight) / 2

		v.Content = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, v.Content)
		if v.Cursor != nil {
			c := *v.Cursor
			c.X += x
			c.Y += y
			v.Cursor = &c
		}
	}

	return v
}

func (t Theme) ModalStyle(width int, borderColor color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(width)
}

func (t Theme) Help() help.Model {
	h := help.New()
	h.Styles.ShortKey = t.KeyStyle
	h.Styles.ShortDesc = t.DescStyle
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.Inactive)
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(t.Inactive).SetString("…")

	h.ShortSeparator = " ⚬ "
	h.FullSeparator = "  "

	return h
}

type HelpMode int

const (
	HelpWithDescriptions HelpMode = iota
	HelpKeysOnly
)

// RenderHelp can omit descriptions for controls whose context already explains the action.
func (t Theme) RenderHelp(km help.KeyMap, modes ...HelpMode) string {
	if km == nil {
		return ""
	}

	if len(modes) > 0 && modes[0] == HelpKeysOnly {
		var hints []string
		for _, binding := range prepareHelp(km.ShortHelp()...) {
			if binding.Enabled() {
				hints = append(hints, t.KeyStyle.Render("["+binding.Help().Key+"]"))
			}
		}
		return strings.Join(hints, t.RenderHelpSeparator())
	}

	return t.Help().View(&shortcutWrapper{km: km})
}

// RenderHelpSeparator shares the muted separator style with adjacent status text.
func (t Theme) RenderHelpSeparator() string {
	help := t.Help()
	return help.Styles.ShortSeparator.Render(help.ShortSeparator)
}

const scrollStepRows = 1

// HandleScroll consumes scroll input even at a boundary. Keep the offset bounded
// in Update so a blocked scroll cannot change the next frame or fall through.
func (t Theme) HandleScroll(msg tea.Msg, scrollOffset *int, maxOffset int) bool {
	delta := 0
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		switch m.String() {
		case KeyUp:
			delta = -scrollStepRows
		case KeyDown:
			delta = scrollStepRows
		}
	case tea.MouseWheelMsg:
		switch m.Button {
		case tea.MouseWheelUp:
			delta = -scrollStepRows
		case tea.MouseWheelDown:
			delta = scrollStepRows
		}
	}
	if delta == 0 {
		return false
	}

	*scrollOffset = max(0, min(*scrollOffset+delta, maxOffset))
	return true
}

func (t Theme) RenderScrollContainer(
	innerContainerWidth, innerContainerHeight int,
	focused bool,
	scrollOffset *int,
	content string,
	totalLines int,
) string {
	return t.renderScrollContainer(
		innerContainerWidth,
		innerContainerHeight,
		focused,
		scrollOffset,
		content,
		totalLines,
		false,
		"",
	)
}

// RenderScrollbarViewport draws an already clipped viewport with a reserved
// scrollbar column. It neither scrolls content nor changes the owner's offset.
func (t Theme) RenderScrollbarViewport(width, height int, content string, total, offset int, focused bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	rows := make([]string, height)
	for y := range rows {
		line := ""
		if y < len(lines) {
			line = ansi.Truncate(lines[y], max(0, width-ScrollbarColumns), "")
		}
		rows[y] = lipgloss.PlaceHorizontal(max(0, width-ScrollbarColumns), lipgloss.Left, line) + strings.Repeat(" ", ScrollbarColumns)
	}
	style := t.BaseStyle
	if focused {
		style = t.FocusedStyle
	}
	return t.ApplyScrollbar(strings.Join(rows, "\n"), 0, height, total, offset, style.GetBorderTopForeground())
}

// RenderInsetScrollContainer reserves a scrollbar column inside the border.
// Optional footer rows stay within that border and do not scroll with the body.
func (t Theme) RenderInsetScrollContainer(
	innerContainerWidth, innerContainerHeight int,
	focused bool,
	scrollOffset *int,
	content string,
	totalLines int,
	footer ...string,
) string {
	return t.renderScrollContainer(
		innerContainerWidth,
		innerContainerHeight,
		focused,
		scrollOffset,
		content,
		totalLines,
		true,
		strings.Join(footer, "\n"),
	)
}

func (t Theme) renderScrollContainer(
	innerContainerWidth, innerContainerHeight int,
	focused bool,
	scrollOffset *int,
	content string,
	totalLines int,
	insetScrollbar bool,
	footer string,
) string {
	var style lipgloss.Style
	if focused {
		style = t.FocusedStyle
	} else {
		style = t.BaseStyle
	}

	borderColor := style.GetBorderTopForeground()
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	lines := strings.Split(content, "\n")
	if totalLines <= 0 {
		totalLines = len(lines)
	}

	maxOffset := totalLines - innerContainerHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	// Clamp scrollOffset
	if *scrollOffset > maxOffset {
		*scrollOffset = maxOffset
	}
	if *scrollOffset < 0 {
		*scrollOffset = 0
	}

	visibleLines := lines
	// If we are using RenderScrollContainer for its own scrolling, slice it.
	if totalLines == len(lines) && len(lines) > innerContainerHeight {
		visibleLines = lines[*scrollOffset:]
		if len(visibleLines) > innerContainerHeight {
			visibleLines = visibleLines[:innerContainerHeight]
		}
	}

	if insetScrollbar {
		visibleLines = strings.Split(
			t.RenderScrollbarViewport(
				innerContainerWidth, innerContainerHeight,
				strings.Join(visibleLines, "\n"), totalLines, *scrollOffset, focused,
			), "\n",
		)
	}

	panelHeight := innerContainerHeight
	if footer != "" {
		footerHeight := lipgloss.Height(footer)
		footerView := CropView(tea.NewView(footer), innerContainerWidth, footerHeight, 0)
		bodyRows := make([]string, innerContainerHeight)
		copy(bodyRows, visibleLines)
		visibleLines = append(bodyRows, strings.Split(footerView.Content, "\n")...)
		panelHeight += footerHeight
	}

	var sb strings.Builder

	// Top border
	sb.WriteString(borderStyle.Render("╭"+strings.Repeat("─", innerContainerWidth)+"╮") + "\n")

	for i := 0; i < panelHeight; i++ {
		// Left border
		sb.WriteString(borderStyle.Render("│"))

		line := ""
		if i < len(visibleLines) {
			line = visibleLines[i]
		}

		// Content: Truncate if too long, pad if too short
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerContainerWidth {
			line = ansi.Truncate(line, innerContainerWidth, "")
		}
		sb.WriteString(lipgloss.PlaceHorizontal(innerContainerWidth, lipgloss.Left, line))

		// Right border
		sb.WriteString(borderStyle.Render("│"))

		if i < panelHeight-1 {
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString("\n" + borderStyle.Render("╰"+strings.Repeat("─", innerContainerWidth)+"╯"))

	if !insetScrollbar {
		return t.ApplyScrollbar(sb.String(), 1, innerContainerHeight, totalLines, *scrollOffset, borderColor)
	}
	return sb.String()
}

type Scrollbar struct {
	HandleStart int
	HandleEnd   int
}

func CalculateScrollbar(containerHeight, totalLines, scrollOffset int) Scrollbar {
	if totalLines <= containerHeight || containerHeight <= 0 {
		return Scrollbar{HandleStart: -1, HandleEnd: -1}
	}

	maxOffset := totalLines - containerHeight
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Proportional handle size
	handleSize := int(float64(containerHeight) * float64(containerHeight) / float64(totalLines))
	if handleSize < 1 {
		handleSize = 1
	}
	// If content is just barely overflowing, ensure handle doesn't take up the WHOLE bar
	if handleSize >= containerHeight {
		handleSize = containerHeight - 1
	}

	// Handle position
	percentage := 0.0
	if maxOffset > 0 {
		percentage = float64(scrollOffset) / float64(maxOffset)
	}
	handleStart := int(percentage * float64(containerHeight-handleSize))
	handleEnd := handleStart + handleSize

	return Scrollbar{
		HandleStart: handleStart,
		HandleEnd:   handleEnd,
	}
}

type shortcutWrapper struct {
	km help.KeyMap
}

func (s shortcutWrapper) ShortHelp() []key.Binding {
	return prepareHelp(s.km.ShortHelp()...)
}

func (s shortcutWrapper) FullHelp() [][]key.Binding {
	fullHelp := s.km.FullHelp()
	result := make([][]key.Binding, 0, len(fullHelp))
	for _, elem := range fullHelp {
		result = append(result, prepareHelp(elem...))
	}

	return result
}

func (t Theme) noticeStyle(kind NoticeKind) lipgloss.Style {
	background := t.ChartUnacked
	switch kind {
	case NoticeSuccess:
		background = t.Secondary
	case NoticeWarning:
		background = t.Primary
	case NoticeError:
		return t.PlainStyle.Foreground(lipgloss.Color(paletteWhite)).Background(t.Error)
	}
	return t.PlainStyle.Foreground(t.TextColor).Background(background)
}

func DefaultTheme() Theme {
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	ld := lipgloss.LightDark(isDark)

	primary := lipgloss.Color(paletteOrange)
	secondary := lipgloss.Color(paletteCyan)
	warning := lipgloss.Color(paletteRed)

	inactive := ld(lipgloss.Color(paletteInactiveLight), lipgloss.Color(paletteInactiveDark))
	placeholder := ld(lipgloss.Color(palettePlaceholderLight), lipgloss.Color(palettePlaceholderDark))
	textColor := ld(lipgloss.Color(paletteBlack), lipgloss.Color(paletteWhite))

	success := ld(lipgloss.Color(paletteSuccessLight), lipgloss.Color(paletteSuccessDark))
	errColor := lipgloss.Color(paletteRed)
	pending := ld(lipgloss.Color(palettePendingLight), lipgloss.Color(palettePendingDark))

	base := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 1)

	return Theme{
		Primary:   primary,
		Secondary: secondary,
		Warning:   warning,
		Inactive:  inactive,
		Success:   success,
		Error:     errColor,
		Pending:   pending,

		BaseStyle:    base.BorderForeground(inactive),
		FocusedStyle: base.BorderForeground(secondary),

		FocusedText: lipgloss.NewStyle().Foreground(primary),
		BlurredText: lipgloss.NewStyle().Foreground(inactive),
		KeyStyle:    lipgloss.NewStyle().Foreground(primary),
		DescStyle:   lipgloss.Style{}.Foreground(inactive),

		HeaderStyle:    lipgloss.NewStyle().Foreground(primary).Bold(true),
		SubHeaderStyle: lipgloss.NewStyle().Foreground(secondary).Bold(true),
		LabelStyle:     lipgloss.Style{}.Foreground(textColor).Bold(true),
		LabelFocused:   lipgloss.Style{}.Foreground(primary).Bold(true),
		FooterStyle:    lipgloss.NewStyle().Foreground(inactive),

		PlaceholderStyle: lipgloss.Style{}.Foreground(placeholder),

		SelectedStyle: lipgloss.Style{}.
			Background(secondary).
			Foreground(textColor),

		RecordRailStyle:         lipgloss.NewStyle().Foreground(inactive),
		RecordRailSelectedStyle: lipgloss.NewStyle().Foreground(secondary),

		TabLabelStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color(paletteWhite)),
		TabKeyStyle:      lipgloss.NewStyle().Foreground(primary),
		TabSelectedStyle: lipgloss.NewStyle().Background(secondary),

		TableHeaderStyle: lipgloss.NewStyle().Foreground(textColor).Bold(true),
		RefreshTextStyle: lipgloss.NewStyle().Foreground(lipgloss.Color(paletteWhite)),

		PlainStyle: lipgloss.NewStyle(),
		TableStyle: lipgloss.NewStyle().Foreground(textColor),

		SuccessStyle: lipgloss.NewStyle().Foreground(success),
		ErrorStyle:   lipgloss.NewStyle().Foreground(errColor),

		TextColor:   textColor,
		BorderColor: inactive,

		ChartReady:   lipgloss.Color(paletteChartReady),
		ChartUnacked: lipgloss.Color(paletteChartUnacked),
		ChartTotal:   lipgloss.Color(paletteChartTotal),

		ChartPublish:          lipgloss.Color(paletteChartPublish),
		ChartDeliverGet:       lipgloss.Color(paletteChartDeliverGet),
		ChartAck:              lipgloss.Color(paletteChartAck),
		ChartRedeliver:        lipgloss.Color(paletteChartRedeliver),
		ChartReturnUnroutable: lipgloss.Color(paletteChartReturnUnroutable),
		ChartDeliver:          lipgloss.Color(paletteChartDeliver),
		ChartGet:              lipgloss.Color(paletteChartGet),
		ChartDeliverNoAck:     lipgloss.Color(paletteChartDeliverNoAck),
		ChartGetNoAck:         lipgloss.Color(paletteChartGetNoAck),
	}
}

func (t Theme) ApplyScrollbar(content string, top, height, total, offset int, borderColor color.Color) string {
	thumb := CalculateScrollbar(height, total, offset)
	if thumb.HandleStart < 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	style := t.PlainStyle.Foreground(borderColor)
	for i := thumb.HandleStart; i < thumb.HandleEnd; i++ {
		y := top + i
		if y >= 0 && y < len(lines) {
			w := lipgloss.Width(lines[y])
			lines[y] = ansi.Cut(lines[y], 0, max(0, w-ScrollbarColumns)) + style.Render("▐")
		}
	}
	return strings.Join(lines, "\n")
}
