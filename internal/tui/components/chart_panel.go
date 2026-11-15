package components

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

const chartHeight = 9

// ChartPanel owns chart geometry and the final clipping boundary. Its size is
// the area inside the border; callers subtract the border exactly once.
// The owning widget keeps data, focus and scroll state and changes them in Update.
type ChartPanel struct {
	theme         core.Theme
	width, height int
	plotHeight    int
	lines         bool
}

func NewChartPanel(theme core.Theme) ChartPanel {
	return ChartPanel{theme: theme}
}

// SetSize reports whether charts need rebuilding with the new content width.
func (p *ChartPanel) SetSize(width, height int) bool {
	width, height = max(0, width), max(0, height)
	if p.width == width && p.height == height {
		return false
	}

	p.width, p.height = width, height
	return true
}

func (p ChartPanel) ContentWidth() int {
	// One column on each side of the content, plus an interior scrollbar.
	return max(0, p.width-3)
}

func (p ChartPanel) NewChart(options ...timeserieslinechart.Option) timeserieslinechart.Model {
	return p.theme.NewChart(p.ContentWidth(), p.chartHeight(), options...)
}

// SetChartAppearance keeps plot sizing and drawing inside the panel boundary.
func (p *ChartPanel) SetChartAppearance(size string, lines bool) {
	p.plotHeight, p.lines = chartHeight, lines
	switch size {
	case "small":
		p.plotHeight = 6
	case "large":
		p.plotHeight = 15
	}
}

func (p ChartPanel) chartHeight() int {
	if p.plotHeight > 0 {
		return p.plotHeight
	}
	return chartHeight
}

// DrawChart applies the panel's dimensions before allocating the braille grid.
// Axis labels can leave no plot area in a hidden or narrow panel.
func (p ChartPanel) DrawChart(chart *timeserieslinechart.Model) {
	chart.Resize(p.ContentWidth(), p.chartHeight())
	if chart.GraphWidth() <= 0 || chart.GraphHeight() <= 0 {
		return
	}

	// Empty datasets can skip drawing in the chart library. Keep the axes
	// visible even when the broker has no history to plot.
	chart.DrawXYAxisAndLabel()
	if p.lines {
		chart.DrawAll()
	} else {
		chart.DrawBrailleAll()
	}
}

func (p ChartPanel) constrain(content string) string {
	if p.ContentWidth() == 0 {
		return ""
	}

	return lipgloss.NewStyle().Width(p.ContentWidth()).MaxWidth(p.ContentWidth()).Render(content)
}

// ClampOffset uses the same wrapped content as Render. Call it in Update after
// resizing, changing content or scrolling; rendering only clamps a local copy.
func (p ChartPanel) ClampOffset(offset int, content string) int {
	return max(0, min(offset, p.maxOffset(content)))
}

func (p ChartPanel) maxOffset(content string) int {
	if p.width <= 0 || p.height <= 0 {
		return 0
	}

	return max(0, lipgloss.Height(p.constrain(content))-p.height)
}

// HandleScroll uses the same wrapped bounds as rendering and resizing.
func (p ChartPanel) HandleScroll(msg tea.Msg, offset *int, content string) bool {
	return p.theme.HandleScroll(msg, offset, p.maxOffset(content))
}

func (p ChartPanel) Center(content string) string {
	return lipgloss.Place(p.ContentWidth(), p.height, lipgloss.Center, lipgloss.Center, content)
}

// Render keeps content, padding and scrollbar inside an uninterrupted border,
// including oversized styled text. It never changes the owner's scroll state.
func (p ChartPanel) Render(content string, offset int, focused bool) string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}

	content = lipgloss.NewStyle().Padding(0, 1).Render(p.constrain(content))
	return p.theme.RenderInsetScrollContainer(p.width, p.height, focused, &offset, content, 0)
}
