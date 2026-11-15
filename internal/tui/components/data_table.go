package components

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	dataTableCellPaddingColumns    = 1
	dataTableHeaderRows            = 1
	dataTableHorizontalStepColumns = 8
	dataRecordRailColumns          = 1
	dataRecordRailGapColumns       = 1
	dataRecordInsetColumns         = dataRecordRailColumns + dataRecordRailGapColumns
)

type DataColumn struct {
	Title string
	Width int
	// Grow shares spare viewport columns by weight.
	Grow int
}

type DataRow struct {
	ID         string
	Cells      []string
	Selectable bool
}

type dataRecordLine struct {
	content string
	row     int
	rail    string
}

type dataRecordBounds struct {
	start, end int // Half-open line bounds in the wrapped viewport.
}

// DataTable owns a clipped table viewport. Read-only mode scrolls without a
// cursor; selection mode skips protected rows. Hosts own data and pagination.
type DataTable struct {
	theme                      core.Theme
	columns                    []DataColumn
	rows                       []DataRow
	selection                  bool
	wrapNavigation             bool
	truncateCells              bool
	responsive                 bool
	recordLayout               bool
	records                    []dataRecordLine
	recordBounds               []dataRecordBounds
	cursor, offset, horizontal int
	width, height              int
	focused                    bool
	footer, status             string
}

func NewDataTable(theme core.Theme, columns []DataColumn, selection bool) *DataTable {
	return &DataTable{theme: theme, columns: columns, selection: selection, cursor: -1}
}

// Sizes are inside the border, matching ChartPanel's coordinate system.
func (t *DataTable) SetSize(width, height int) {
	t.width, t.height = max(0, width), max(0, height)
	t.rebuildRecords()
	t.clamp()
}

// SetTruncateCells sizes columns from their configured widths and growth weights,
// truncating long values with an ellipsis instead of scrolling horizontally.
func (t *DataTable) SetTruncateCells(enabled bool) {
	t.truncateCells = enabled
	t.rebuildRecords()
	t.clamp()
}

// SetResponsive uses grouped, wrapped records when the columns cannot fit.
// Selectable records retain row identity for keyboard and mouse actions.
func (t *DataTable) SetResponsive(enabled bool) {
	t.responsive = enabled
	t.rebuildRecords()
	t.clamp()
}

// SetRecordLayout keeps grouped fields at every width, without switching to columns.
func (t *DataTable) SetRecordLayout(enabled bool) {
	t.recordLayout = enabled
	t.rebuildRecords()
	t.clamp()
}

func (t *DataTable) recordMode() bool {
	return t.recordLayout || (t.responsive && t.naturalWidth() > t.contentWidth())
}

func (t *DataTable) headerRows() int {
	if t.recordMode() {
		return 0
	}
	return dataTableHeaderRows
}

// Cache wrapped records when data or geometry changes, keeping scrolling and
// rendering independent of the number of offscreen fields.
func (t *DataTable) rebuildRecords() {
	t.records = nil
	t.recordBounds = nil
	if !t.recordMode() || t.contentWidth() <= dataRecordInsetColumns {
		return
	}
	for rowIndex, row := range t.rows {
		properties := make([]core.Property, len(t.columns))
		for i, column := range t.columns {
			value := ""
			if i < len(row.Cells) {
				value = tableCell(row.Cells[i])
			}
			properties[i] = core.Property{Label: column.Title, Value: value}
		}
		if rowIndex > 0 {
			t.records = append(t.records, dataRecordLine{row: -1})
		}
		start := len(t.records)
		lines := strings.Split(t.theme.RenderProperties(properties, t.contentWidth()-dataRecordInsetColumns), "\n")
		for i, line := range lines {
			rail := "│"
			if i == 0 {
				rail = "╭"
			} else if i == len(lines)-1 {
				rail = "╰"
			}
			t.records = append(t.records, dataRecordLine{content: line, row: rowIndex, rail: rail})
		}
		t.recordBounds = append(t.recordBounds, dataRecordBounds{start: start, end: len(t.records)})
	}
}

func (t *DataTable) scrollRows() int {
	if t.recordMode() {
		return len(t.records)
	}
	return len(t.rows)
}

// SetWrapNavigation wraps arrow-key selection; mouse wheels still stop at the ends.
func (t *DataTable) SetWrapNavigation(enabled bool) { t.wrapNavigation = enabled }

// MinimumWidth includes column padding, the reserved scrollbar and outer borders.
func (t *DataTable) MinimumWidth() int {
	width := core.ScrollbarColumns + core.BorderOverhead
	for _, column := range t.columns {
		width += max(column.Width, lipgloss.Width(column.Title)) + dataTableCellPaddingColumns*2
	}
	return width
}

func (t *DataTable) SetFocused(focused bool) { t.focused = focused }

func (t *DataTable) SetFooter(footer string) {
	t.footer = footer
	t.clamp()
}

func (t *DataTable) SetStatus(status string) { t.status = status }

func (t *DataTable) Cursor() int { return t.cursor }

func (t *DataTable) ResetScroll() { t.offset, t.horizontal = 0, 0 }

func (t *DataTable) SetRows(rows []DataRow) {
	selected := ""
	if t.cursor >= 0 && t.cursor < len(t.rows) {
		selected = t.rows[t.cursor].ID
	}
	t.rows, t.cursor = rows, -1
	if t.selection {
		for i, row := range rows {
			if !row.Selectable {
				continue
			}
			if t.cursor < 0 || row.ID == selected {
				t.cursor = i
			}
			if row.ID == selected {
				break
			}
		}
	}
	t.rebuildRecords()
	t.clamp()
}

func (t *DataTable) bodyHeight() int {
	footerRows := 0
	if t.footer != "" {
		footerRows = lipgloss.Height(t.footer)
	}
	return max(0, t.height-footerRows-t.headerRows())
}

func (t *DataTable) contentWidth() int { return max(0, t.width-core.ScrollbarColumns) }

func (t *DataTable) naturalWidth() int {
	width := 0
	for _, colWidth := range t.columnWidths() {
		width += colWidth + dataTableCellPaddingColumns*2
	}
	return width
}

// Unless truncating, grow columns to their data so horizontal scrolling can
// reveal complete values. Widths are terminal cells, including styled text.
func (t *DataTable) columnWidths() []int {
	widths := make([]int, len(t.columns))
	for i, col := range t.columns {
		widths[i] = max(col.Width, lipgloss.Width(col.Title))
		if t.truncateCells {
			continue
		}
		for _, row := range t.rows {
			if i < len(row.Cells) {
				widths[i] = max(widths[i], lipgloss.Width(tableCell(row.Cells[i])))
			}
		}
	}
	return widths
}

// Stretch only into unused space. The unstretched widths govern overflow and
// responsive records, so growth never creates horizontal scroll.
func (t *DataTable) viewportColumnWidths() []int {
	widths := t.columnWidths()
	remainingWidth := t.contentWidth()
	remainingWeight := 0
	for i, column := range t.columns {
		remainingWidth -= widths[i] + dataTableCellPaddingColumns*2
		remainingWeight += max(0, column.Grow)
	}
	if remainingWidth <= 0 || remainingWeight == 0 {
		return widths
	}

	for i, column := range t.columns {
		if column.Grow <= 0 {
			continue
		}
		extra := remainingWidth * column.Grow / remainingWeight
		widths[i] += extra
		remainingWidth -= extra
		remainingWeight -= column.Grow
	}
	return widths
}

func tableCell(value string) string {
	return strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(value)
}

func (t *DataTable) clamp() {
	t.offset = max(0, min(t.offset, t.scrollRows()-t.bodyHeight()))
	t.horizontal = max(0, min(t.horizontal, t.naturalWidth()-t.contentWidth()))
	if t.truncateCells {
		t.horizontal = 0
	}
	if t.selection && t.cursor >= 0 && t.bodyHeight() > 0 {
		start, end := t.cursor, t.cursor+1
		if t.recordMode() {
			if t.cursor >= len(t.recordBounds) {
				return
			}
			bounds := t.recordBounds[t.cursor]
			start, end = bounds.start, bounds.end
			// A long selected record may be scrolled within. Refreshes must
			// preserve that position while any part of it remains visible.
			if start < t.offset+t.bodyHeight() && end > t.offset {
				return
			}
		}
		if start < t.offset {
			t.offset = start
		}
		if end > t.offset+t.bodyHeight() {
			t.offset = min(start, end-t.bodyHeight())
		}
	}
}

func (t *DataTable) move(delta int) {
	if !t.selection {
		t.offset += delta
		return
	}
	if len(t.rows) == 0 {
		return
	}
	if t.cursor < 0 {
		for i, row := range t.rows {
			if row.Selectable {
				t.cursor = i
				return
			}
		}
		return
	}
	direction := 1
	if delta < 0 {
		direction = -1
	}
	for step := 0; step < max(delta, -delta); step++ {
		for next := t.cursor + direction; next >= 0 && next < len(t.rows); next += direction {
			if t.rows[next].Selectable {
				t.cursor = next
				break
			}
		}
	}
	if t.recordMode() && t.cursor < len(t.recordBounds) {
		bounds := t.recordBounds[t.cursor]
		if bounds.start < t.offset {
			t.offset = bounds.start
		} else if bounds.end > t.offset+t.bodyHeight() {
			t.offset = min(bounds.start, max(0, bounds.end-t.bodyHeight()))
		}
	}
}

// Wrapped keyboard navigation still skips rows that cannot be selected.
func (t *DataTable) moveWrapped(delta int) {
	if t.cursor < 0 || t.cursor >= len(t.rows) {
		t.move(delta)
		return
	}

	for step := 1; step <= len(t.rows); step++ {
		next := (t.cursor + step*delta + len(t.rows)) % len(t.rows)
		if t.rows[next].Selectable {
			t.cursor = next
			return
		}
	}
}

func (t *DataTable) scrollRecords(delta int) {
	next := max(0, min(t.offset+delta, len(t.records)-t.bodyHeight()))
	if next == t.offset {
		return
	}
	t.offset = next
	if !t.selection {
		return
	}
	if t.cursor >= 0 && t.cursor < len(t.recordBounds) {
		bounds := t.recordBounds[t.cursor]
		if bounds.start < t.offset+t.bodyHeight() && bounds.end > t.offset {
			return
		}
	}
	t.cursor = -1
	for i := t.offset; i < min(len(t.records), t.offset+t.bodyHeight()); i++ {
		row := t.records[i].row
		if row >= 0 && t.rows[row].Selectable {
			t.cursor = row
			return
		}
	}
}

func (t *DataTable) Update(msg tea.Msg) tea.Cmd {
	if !t.focused {
		return nil
	}
	if t.recordMode() {
		switch m := msg.(type) {
		case tea.MouseWheelMsg:
			if m.Button == tea.MouseWheelUp {
				t.scrollRecords(-1)
			} else if m.Button == tea.MouseWheelDown {
				t.scrollRecords(1)
			}
			return nil
		case tea.KeyPressMsg:
			if key.Matches(m, core.TablePageScroll()) {
				delta := max(1, t.bodyHeight())
				if m.String() == core.KeyPgUp {
					delta = -delta
				}
				t.scrollRecords(delta)
				return nil
			}
		}
	}
	defer t.clamp()
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(m, core.TableHorizontalScroll()) && !t.recordMode() && !t.truncateCells:
			if m.String() == core.KeyShiftLeft {
				t.horizontal -= dataTableHorizontalStepColumns
			} else {
				t.horizontal += dataTableHorizontalStepColumns
			}
		case key.Matches(m, core.TableScroll()):
			delta := 1
			if m.String() == core.KeyUp {
				delta = -1
			}
			if t.wrapNavigation && t.selection {
				t.moveWrapped(delta)
			} else {
				t.move(delta)
			}
		case key.Matches(m, core.TablePageScroll()):
			if m.String() == core.KeyPgUp {
				t.move(-max(1, t.bodyHeight()))
			} else {
				t.move(max(1, t.bodyHeight()))
			}
		}
	case tea.MouseWheelMsg:
		if m.Button == tea.MouseWheelUp {
			t.move(-1)
		} else if m.Button == tea.MouseWheelDown {
			t.move(1)
		}
	case tea.MouseClickMsg:
		// Host supplies coordinates relative to the panel, including its border.
		row := m.Y - t.theme.BaseStyle.GetBorderTopSize() - t.headerRows() + t.offset
		if t.recordMode() {
			if row < 0 || row >= len(t.records) {
				return nil
			}
			row = t.records[row].row
		}
		visible := row >= t.offset && row < min(len(t.rows), t.offset+t.bodyHeight())
		if t.recordMode() {
			y := m.Y - t.theme.BaseStyle.GetBorderTopSize()
			visible = y >= 0 && y < t.bodyHeight()
		}
		if t.selection && m.Button == tea.MouseLeft && m.X > 0 && m.X <= t.contentWidth() && visible && row >= 0 && row < len(t.rows) && t.rows[row].Selectable {
			t.cursor = row
		}
	}
	return nil
}

func (t *DataTable) line(cells []string, widths []int) string {
	parts := make([]string, len(t.columns))
	padding := strings.Repeat(" ", dataTableCellPaddingColumns)
	for i, width := range widths {
		value := ""
		if i < len(cells) {
			value = tableCell(cells[i])
		}
		parts[i] = padding + lipgloss.PlaceHorizontal(width, lipgloss.Left, ansi.Truncate(value, width, "…")) + padding
	}
	return ansi.Cut(strings.Join(parts, ""), t.horizontal, t.horizontal+t.contentWidth())
}

func (t *DataTable) View() tea.View {
	if t.width <= 0 || t.height <= 0 {
		return tea.NewView("")
	}
	titles := make([]string, len(t.columns))
	for i, col := range t.columns {
		titles[i] = col.Title
	}
	widths := t.viewportColumnWidths()
	lines := []string{t.theme.TableHeaderStyle.Render(t.line(titles, widths))}
	recordMode := t.recordMode()
	for i := t.offset; !recordMode && i < min(len(t.rows), t.offset+t.bodyHeight()); i++ {
		line := t.line(t.rows[i].Cells, widths)
		if t.selection && i == t.cursor {
			line = t.theme.SelectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	if recordMode {
		lines = nil
		for _, record := range t.records[t.offset:min(len(t.records), t.offset+t.bodyHeight())] {
			if record.row < 0 {
				lines = append(lines, "")
				continue
			}
			railStyle := t.theme.RecordRailStyle
			if t.selection && record.row == t.cursor && t.focused {
				railStyle = t.theme.RecordRailSelectedStyle
			}
			lines = append(lines, railStyle.Render(record.rail)+strings.Repeat(" ", dataRecordRailGapColumns)+record.content)
		}
	}
	if t.status != "" {
		lines = []string{t.theme.BlurredText.Render(ansi.Truncate(t.status, t.contentWidth(), "…"))}
	}
	footer := t.footer
	if footer != "" {
		footerLines := strings.Split(footer, "\n")
		for i, line := range footerLines {
			footerLines[i] = lipgloss.PlaceHorizontal(t.contentWidth(), lipgloss.Center, ansi.Truncate(line, t.contentWidth(), ""))
		}
		footer = strings.Join(footerLines, "\n")
	}
	bodyRows := min(t.height, t.bodyHeight()+t.headerRows())
	if footer != "" && t.height <= lipgloss.Height(footer) {
		footer = ""
		bodyRows = t.height
	}
	// The header is pinned. The shared renderer owns the reserved scrollbar column.
	offset := t.offset
	total := t.scrollRows() + t.headerRows()
	if t.status != "" {
		total = bodyRows
		offset = 0
	}
	content := strings.Join(lines, "\n")
	// Pad the clipped viewport so RenderInsetScrollContainer never scrolls it again.
	content = lipgloss.NewStyle().Height(bodyRows).MaxHeight(bodyRows).Render(content)
	return tea.NewView(t.theme.RenderInsetScrollContainer(t.width, bodyRows, t.focused, &offset, content, total, footer))
}
