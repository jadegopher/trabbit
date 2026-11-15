package widgets

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const runtimeHistory = time.Minute

type QueueRuntime struct {
	theme               core.Theme
	panel               components.ChartPanel
	chart               timeserieslinechart.Model
	data                domain.QueueRuntime
	vhost, name, status string
	offset              int
	focused             bool
	hasData             bool
}

func NewQueueRuntime(theme core.Theme) *QueueRuntime {
	return &QueueRuntime{theme: theme, panel: components.NewChartPanel(theme), status: "Select a queue"}
}

func (r *QueueRuntime) SetLoading(vhost, name string) {
	if r.vhost == vhost && r.name == name {
		return
	}
	r.Clear()
	r.vhost, r.name, r.status = vhost, name, ""
}

func (r *QueueRuntime) Clear() {
	r.vhost, r.name, r.status = "", "", "Select a queue"
	r.data = domain.QueueRuntime{}
	r.hasData = false
	r.offset = 0
}

func (r *QueueRuntime) SetSize(width, height int) {
	if r.panel.SetSize(width, height) {
		r.rebuild()
	}
	r.clamp()
}

func (r *QueueRuntime) SetFocused(focused bool) { r.focused = focused }

func (r *QueueRuntime) SetData(data domain.QueueRuntime) {
	r.data, r.status = data, ""
	r.hasData = true
	r.rebuild()
	r.clamp()
}

func (r *QueueRuntime) SetError(err error) {
	r.status = "Runtime metrics unavailable: " + err.Error()
	r.clamp()
}

func (r *QueueRuntime) rebuild() {
	var points []timeserieslinechart.TimePoint
	if r.data.Reductions != nil {
		points = counterRatePoints(r.data.Reductions.Samples)
	}
	end := time.Now()
	if len(points) > 0 {
		end = points[len(points)-1].Time
	}
	start := end.Add(-runtimeHistory)
	chart := r.panel.NewChart(
		timeserieslinechart.WithTimeRange(start, end),
		timeserieslinechart.WithXYSteps(chartTimeLabelStep, chartValueLabelStep),
		timeserieslinechart.WithYLabelFormatter(func(_ int, value float64) string { return core.FormatRateValue(value) }),
	)
	chart.SetDataSetStyle("Reductions", lipgloss.NewStyle().Foreground(r.theme.ChartTotal))
	for _, point := range points {
		if !point.Time.Before(start) {
			chart.PushDataSet("Reductions", point)
		}
	}
	r.panel.DrawChart(&chart)
	r.chart = chart
}

func (r *QueueRuntime) content() string {
	if r.status != "" {
		return r.status
	}
	// The parent owns delayed loading; missing metrics only describe a fetched snapshot.
	if !r.hasData {
		return ""
	}

	parts := []string{r.theme.HeaderStyle.Render("Reductions (per second) — last minute")}
	if r.data.Reductions == nil {
		parts = append(parts, "Reductions: unavailable")
	} else {
		parts = append(parts, r.chart.View(), r.theme.RenderLegendItem(r.theme.ChartTotal, "Reductions", core.FormatValue(r.data.Reductions.Rate)+"/s", true))
		if len(r.data.Reductions.Samples) < minimumCounterSamples {
			parts = append(parts, "Waiting for reductions history")
		}
	}
	parts = append(parts, "", r.theme.RenderProperties([]core.Property{
		{Label: "Minimum binary virtual heap (words)", Value: optionalMetric(r.data.MinBinVHeapWords)},
		{Label: "Minimum heap (words)", Value: optionalMetric(r.data.MinHeapWords)},
		{Label: "Collections before full sweep", Value: optionalMetric(r.data.FullSweepAfter)},
		{Label: "Minor GCs", Value: optionalMetric(r.data.MinorGCs)},
	}, r.panel.ContentWidth()))
	return strings.Join(parts, "\n")
}

func (r *QueueRuntime) clamp() { r.offset = r.panel.ClampOffset(r.offset, r.content()) }

func (r *QueueRuntime) Update(msg tea.Msg) tea.Cmd {
	if r.focused {
		r.panel.HandleScroll(msg, &r.offset, r.content())
	}
	return nil
}

func (r *QueueRuntime) View() tea.View {
	return tea.NewView(r.panel.Render(r.content(), r.offset, r.focused))
}
