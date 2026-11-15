package widgets

import (
	"encoding/json"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

const (
	minColumnWidth      = 30
	columnSpacing       = 2
	chartTimeLabelStep  = 6
	chartValueLabelStep = 2
	minimumRateTimeSpan = time.Second
)

type QueueDetails struct {
	theme           core.Theme
	queuedMsgsChart timeserieslinechart.Model
	msgRatesChart   timeserieslinechart.Model
	panel           components.ChartPanel
	chartSettings   settings.ChartSettings

	focused      bool
	scrollOffset int
	lastStats    domain.QueueDetailedStats

	loadingVHost string
	loadingName  string
}

type MsgChartSettings struct{ Settings settings.ChartSettings }

func NewQueueCharts(theme core.Theme) *QueueDetails {
	panel := components.NewChartPanel(theme)
	qmc := panel.NewChart(timeserieslinechart.WithXYSteps(3, 10))
	mrc := panel.NewChart(timeserieslinechart.WithXYSteps(3, 10))

	// Configure styles
	readyStyle := lipgloss.NewStyle().Foreground(theme.ChartReady)
	unackedStyle := lipgloss.NewStyle().Foreground(theme.ChartUnacked)
	totalStyle := lipgloss.NewStyle().Foreground(theme.ChartTotal)

	qmc.SetDataSetStyle("Ready", readyStyle)
	qmc.SetDataSetStyle("Unacked", unackedStyle)
	qmc.SetDataSetStyle("Total", totalStyle)

	return &QueueDetails{
		theme:           theme,
		panel:           panel,
		chartSettings:   settings.DefaultChartSettings(),
		queuedMsgsChart: qmc,
		msgRatesChart:   mrc,
	}
}

func (q *QueueDetails) Init() tea.Cmd { return nil }

func (q *QueueDetails) SetLoading(vhost, name string) tea.Cmd {
	if q.loadingVHost != vhost || q.loadingName != name {
		q.lastStats = domain.QueueDetailedStats{}
		q.updateData()
		q.scrollOffset = 0
	}
	q.loadingVHost = vhost
	q.loadingName = name
	return nil
}

func (q *QueueDetails) Clear() {
	q.lastStats = domain.QueueDetailedStats{}
	q.loadingName = ""
	q.loadingVHost = ""
	q.updateData()
	q.scrollOffset = 0
}

func (q *QueueDetails) SetFocused(focused bool) {
	q.focused = focused
}

func (q *QueueDetails) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer func() {
		q.scrollOffset = q.panel.ClampOffset(q.scrollOffset, q.content())
	}()

	switch m := msg.(type) {
	case MsgChartSettings:
		q.chartSettings = m.Settings.Clone()
		q.panel.SetChartAppearance(m.Settings.Size, m.Settings.Style == "lines")
		q.updateData()

	case tea.WindowSizeMsg:
		q.setSize(m.Width, m.Height)

	case MsgQueueDetailedStats:

		q.lastStats = m.Stats
		q.updateData()

	case tea.KeyPressMsg, tea.MouseWheelMsg:
		if q.focused {
			if handled := q.panel.HandleScroll(m, &q.scrollOffset, q.content()); handled {
				return q, nil
			}
		}
	}
	return q, nil
}

func (q *QueueDetails) updateData() {
	q.updateQueuedMsgsChart()
	q.updateMsgRatesChart()
}

func (q *QueueDetails) updateQueuedMsgsChart() {
	minTime, maxTime, ok := q.queuedTimeRange()
	if !ok {
		minTime, maxTime = q.emptyChartTimeRange()
	}

	minTime = q.rangeStart(minTime, maxTime)
	chart := q.panel.NewChart(
		timeserieslinechart.WithXYSteps(chartTimeLabelStep, chartValueLabelStep),
		timeserieslinechart.WithTimeRange(minTime, maxTime),
	)
	chart.SetDataSetStyle("Ready", lipgloss.NewStyle().Foreground(q.theme.ChartReady))
	chart.SetDataSetStyle("Unacked", lipgloss.NewStyle().Foreground(q.theme.ChartUnacked))
	chart.SetDataSetStyle("Total", lipgloss.NewStyle().Foreground(q.theme.ChartTotal))

	for _, name := range settings.MessageSeries() {
		if q.chartSettings.MessageVisible(name) {
			q.pushSortedSamples(&chart, name, q.messageSamples(name))
		}
	}

	q.panel.DrawChart(&chart)
	q.queuedMsgsChart = chart
}

func (q *QueueDetails) updateMsgRatesChart() {
	type series struct {
		name   string
		points []timeserieslinechart.TimePoint
	}
	var data []series
	var minTime, maxTime time.Time

	for _, name := range settings.ThroughputSeries() {
		if !q.chartSettings.ThroughputVisible(name) {
			continue
		}

		rd := q.getRateDetails(name)
		points := q.calculateRatePoints(rd.Samples)
		if len(points) == 0 {
			continue
		}

		if minTime.IsZero() || points[0].Time.Before(minTime) {
			minTime = points[0].Time
		}
		if maxTime.IsZero() || points[len(points)-1].Time.After(maxTime) {
			maxTime = points[len(points)-1].Time
		}

		data = append(data, series{name: name, points: points})
	}

	if len(data) == 0 {
		minTime, maxTime = q.emptyChartTimeRange()
	}
	// Two counter samples produce a single rate point. Keep a nonzero time
	// range so the chart can scale that point while history is warming up.
	if minTime.Equal(maxTime) {
		minTime = maxTime.Add(-minimumRateTimeSpan)
	}
	minTime = q.rangeStart(minTime, maxTime)
	chart := q.panel.NewChart(
		timeserieslinechart.WithXYSteps(chartTimeLabelStep, chartValueLabelStep),
		timeserieslinechart.WithTimeRange(minTime, maxTime),
		timeserieslinechart.WithYLabelFormatter(
			func(i int, v float64) string {
				return core.FormatRateValue(v)
			},
		),
	)

	for _, s := range data {
		style := lipgloss.NewStyle().Foreground(q.getRateColor(s.name))
		chart.SetDataSetStyle(s.name, style)
		for _, p := range s.points {
			if p.Time.Before(minTime) {
				continue
			}
			chart.PushDataSet(s.name, p)
		}
	}

	q.panel.DrawChart(&chart)
	q.msgRatesChart = chart
}

func (q *QueueDetails) getTimeRange(samples []domain.Sample) (time.Time, time.Time, bool) {
	if len(samples) < 2 {
		return time.Time{}, time.Time{}, false
	}

	// Samples are newest first.
	// The range will be from the oldest sample to the newest.
	start := time.UnixMilli(samples[len(samples)-1].Timestamp)
	end := time.UnixMilli(samples[0].Timestamp)

	return start, end, true
}

func (q *QueueDetails) pushSortedSamples(chart *timeserieslinechart.Model, name string, samples []domain.Sample) {
	if len(samples) == 0 {
		return
	}

	// Samples are newest first, so we iterate from oldest (len-1) to newest (0).
	for i := len(samples) - 1; i >= 0; i-- {
		s := samples[i]
		if float64(s.Timestamp)/1000 < chart.ViewMinX() {
			continue
		}
		chart.PushDataSet(
			name, timeserieslinechart.TimePoint{
				Time:  time.UnixMilli(s.Timestamp),
				Value: float64(s.Value),
			},
		)
	}
}

func (q *QueueDetails) calculateRatePoints(samples []domain.Sample) []timeserieslinechart.TimePoint {
	return counterRatePoints(samples)
}

func (q *QueueDetails) getRateDetails(name string) domain.RateDetails {
	switch name {
	case "Publish":
		return q.lastStats.MessageStats.PublishDetails
	case "Deliver":
		return q.lastStats.MessageStats.DeliverDetails
	case "DeliverNoAck":
		return q.lastStats.MessageStats.DeliverNoAckDetails
	case "DeliverGet":
		return q.lastStats.MessageStats.DeliverGetDetails
	case "Redeliver":
		return q.lastStats.MessageStats.RedeliverDetails
	case "Get":
		return q.lastStats.MessageStats.GetDetails
	case "GetNoAck":
		return q.lastStats.MessageStats.GetNoAckDetails
	case "Ack":
		return q.lastStats.MessageStats.AckDetails
	case "ReturnUnroutable":
		return q.lastStats.MessageStats.ReturnUnroutableDetails
	case "DropUnroutable":
		return q.lastStats.MessageStats.DropUnroutableDetails
	default:
		return domain.RateDetails{}
	}
}

func (q *QueueDetails) getRateColor(name string) color.Color {
	switch name {
	case "Publish":
		return q.theme.ChartPublish
	case "DeliverGet":
		return q.theme.ChartDeliverGet
	case "Ack":
		return q.theme.ChartAck
	case "Redeliver":
		return q.theme.ChartRedeliver
	case "ReturnUnroutable":
		return q.theme.ChartReturnUnroutable
	case "DropUnroutable":
		return q.theme.Warning
	case "Deliver":
		return q.theme.ChartDeliver
	case "Get":
		return q.theme.ChartGet
	case "DeliverNoAck":
		return q.theme.ChartDeliverNoAck
	case "GetNoAck":
		return q.theme.ChartGetNoAck
	default:
		return q.theme.Inactive
	}
}

func (q *QueueDetails) setSize(width, height int) {
	if !q.panel.SetSize(width, height) {
		return
	}

	q.updateData()
}

func (q *QueueDetails) View() tea.View {
	return tea.NewView(q.panel.Render(q.content(), q.scrollOffset, q.focused))
}

func (q *QueueDetails) renderExtraInfo() string {
	features := q.renderFeatures()
	messages := q.renderMessages()
	policies := q.renderPolicies()
	consumers := q.renderConsumers()

	// Calculate column width for 4 columns
	colWidth := (q.panel.ContentWidth() - (3 * columnSpacing)) / 4
	if colWidth < minColumnWidth {
		// Not enough space for 4 columns, try 2x2
		colWidth = (q.panel.ContentWidth() - columnSpacing) / 2
		if colWidth < minColumnWidth {
			// Stack them
			return features + "\n\n" + messages + "\n\n" + policies + "\n\n" + consumers
		}
		row1 := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(colWidth).Render(features),
			strings.Repeat(" ", columnSpacing),
			lipgloss.NewStyle().Width(colWidth).Render(messages),
		)
		row2 := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(colWidth).Render(policies),
			strings.Repeat(" ", columnSpacing),
			lipgloss.NewStyle().Width(colWidth).Render(consumers),
		)
		return row1 + "\n\n" + row2
	}

	fCol := lipgloss.NewStyle().Width(colWidth).Render(features)
	mCol := lipgloss.NewStyle().Width(colWidth).Render(messages)
	pCol := lipgloss.NewStyle().Width(colWidth).Render(policies)
	cCol := lipgloss.NewStyle().Width(colWidth).Render(consumers)

	spacing := strings.Repeat(" ", columnSpacing)
	return lipgloss.JoinHorizontal(lipgloss.Top, fCol, spacing, mCol, spacing, pCol, spacing, cCol)
}

func (q *QueueDetails) renderFeatures() string {
	title := q.theme.HeaderStyle.Render("General")

	var sb strings.Builder
	sb.WriteString("Process Memory: " + core.FormatBytes(q.lastStats.Memory) + "\n")
	sb.WriteString("Storage version: " + strconv.Itoa(q.lastStats.StorageVersion) + "\n")

	// Mentioned fields: hide if default (false/empty)
	sb.WriteString(q.renderFeatureBool("Durable", q.lastStats.Durable))
	sb.WriteString(q.renderFeatureBool("Exclusive", q.lastStats.Exclusive))
	sb.WriteString(q.renderFeatureBool("Internal", q.lastStats.Internal))
	sb.WriteString(q.renderFeatureBool("Internal Owner", q.lastStats.InternalOwner))
	sb.WriteString(q.renderFeatureString("Single Active Consumer Tag", q.lastStats.SingleActiveConsumerTag))
	sb.WriteString(q.renderFeatureString("Exclusive Consumer Tag", q.lastStats.ExclusiveConsumerTag))
	sb.WriteString(q.renderFeatureString("Idle Since", core.FormatTime(q.lastStats.IdleSince)))

	args := q.formatMap(q.lastStats.Arguments)
	sb.WriteString("Arguments:\n" + q.indent(args, 2))

	return title + "\n" + sb.String()
}

func (q *QueueDetails) renderMessages() string {
	title := q.theme.HeaderStyle.Render("Messages")

	headerStyle := q.theme.TableHeaderStyle.PaddingRight(1)
	cellStyle := q.theme.TableStyle.PaddingRight(1)

	rows := [][]string{
		{
			"Total",
			core.FormatValue(float64(q.lastStats.Messages.Total)),
			core.FormatBytes(q.lastStats.Messages.TotalBytes),
		},
		{
			"Ready",
			core.FormatValue(float64(q.lastStats.Messages.Ready)),
			core.FormatBytes(q.lastStats.Messages.ReadyBytes),
		},
		{
			"Unacked",
			core.FormatValue(float64(q.lastStats.Messages.Unacked)),
			core.FormatBytes(q.lastStats.Messages.UnackedBytes),
		},
		{
			"In Memory",
			core.FormatValue(float64(q.lastStats.Messages.RAM)),
			core.FormatBytes(q.lastStats.Messages.RAMBytes),
		},
		{
			"Persistent",
			core.FormatValue(float64(q.lastStats.Messages.Persistent)),
			core.FormatBytes(q.lastStats.Messages.PersistentBytes),
		},
	}

	headers := []string{"", "Count", "Size"}

	// Calculate widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(title + "\n")

	// Render header
	for i, h := range headers {
		sb.WriteString(headerStyle.Width(widths[i] + 1).Render(h))
	}
	sb.WriteString("\n")

	// Render rows
	for _, row := range rows {
		for i, cell := range row {
			sb.WriteString(cellStyle.Width(widths[i] + 1).Render(cell))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (q *QueueDetails) renderFeatureBool(label string, val bool) string {
	if !val {
		return ""
	}
	return fmt.Sprintf("%s: true\n", label)
}

func (q *QueueDetails) renderFeatureString(label string, val string) string {
	if val == "" {
		return ""
	}
	return fmt.Sprintf("%s: %s\n", label, val)
}

func (q *QueueDetails) renderPolicies() string {
	title := q.theme.HeaderStyle.Render("Policies")

	policy := q.lastStats.Policy
	if policy == "" {
		policy = "none"
	}

	opPolicy := q.lastStats.OperatorPolicy
	if opPolicy == "" {
		opPolicy = "none"
	}

	effPolicy := q.formatMap(q.lastStats.EffectivePolicyDefinition)

	return title + "\n" +
		fmt.Sprintf("Policy: %s\n", policy) +
		fmt.Sprintf("Operator Policy: %s\n", opPolicy) +
		"Effective policy definition:\n" + q.indent(effPolicy, 2)
}

func (q *QueueDetails) renderConsumers() string {
	title := q.theme.HeaderStyle.Render("Consumers")

	count := fmt.Sprintf("%d", q.lastStats.Consumers)
	capacity := fmt.Sprintf("%.2f%%", q.lastStats.ConsumerCapacity*100)
	utilisation := fmt.Sprintf("%.2f%%", q.lastStats.ConsumerUtilisation*100)

	return title + "\n" +
		"Count: " + count + "\n" +
		"Capacity: " + capacity + "\n" +
		"Utilization: " + utilisation
}

func (q *QueueDetails) indent(s string, n int) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", n) + line
	}
	return strings.Join(lines, "\n")
}

func (q *QueueDetails) formatMap(m map[string]interface{}) string {
	if len(m) == 0 {
		return "none"
	}
	var lines []string
	for k, v := range m {
		b, _ := json.Marshal(v)
		lines = append(lines, fmt.Sprintf("%s: %s", k, string(b)))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func (q *QueueDetails) renderQueuedMsgs() string {
	title := q.theme.HeaderStyle.Render("Queued messages")

	readyVal := core.FormatValue(float64(q.lastStats.Messages.Ready))
	unackedVal := core.FormatValue(float64(q.lastStats.Messages.Unacked))
	totalVal := core.FormatValue(float64(q.lastStats.Messages.Total))

	legend := strings.Join(
		[]string{
			q.theme.RenderLegendItem(q.theme.ChartReady, "Ready", readyVal, q.chartSettings.MessageVisible("Ready")),
			q.theme.RenderLegendItem(q.theme.ChartUnacked, "Unacked", unackedVal, q.chartSettings.MessageVisible("Unacked")),
			q.theme.RenderLegendItem(q.theme.ChartTotal, "Total", totalVal, q.chartSettings.MessageVisible("Total")),
		}, " ",
	)

	legend = lipgloss.NewStyle().Width(q.panel.ContentWidth()).Render(legend)

	return title + "\n" + q.queuedMsgsChart.View() + "\n" + legend
}

func (q *QueueDetails) renderMsgRates() string {
	title := q.theme.HeaderStyle.Render("Message Throughput")
	legend := q.renderRatesLegend()

	return title + "\n" + q.msgRatesChart.View() + "\n" + legend
}

func (q *QueueDetails) renderRatesLegend() string {
	var items []string
	for _, name := range settings.ThroughputSeries() {
		rd := q.getRateDetails(name)
		val := core.FormatValue(rd.Rate) + "/s"
		items = append(items, q.theme.RenderLegendItem(q.getRateColor(name), name, val, q.chartSettings.ThroughputVisible(name)))
	}

	legendContent := strings.Join(items, " ")
	return lipgloss.NewStyle().Width(q.panel.ContentWidth()).Render(legendContent)
}

func (q *QueueDetails) content() string {
	return q.renderQueueInfo() + "\n\n" + q.renderQueuedMsgs() + "\n" + q.renderMsgRates() + "\n\n" + q.renderExtraInfo()
}

func (q *QueueDetails) renderQueueInfo() string {
	name, vhost := q.lastStats.Name, q.lastStats.VHost
	if name == "" {
		name = q.loadingName
	}
	if vhost == "" {
		vhost = q.loadingVHost
	}

	return q.theme.HeaderStyle.Foreground(q.theme.TextColor).Render("Queue: "+name) + "\n" +
		fmt.Sprintf("Vhost: %s  Type: %s  State: %s", vhost, q.lastStats.Type, q.lastStats.State)
}

// Use queue history to align an empty throughput chart with the chart above it.
// Without history, show the selected time window without inventing samples.
func (q *QueueDetails) emptyChartTimeRange() (time.Time, time.Time) {
	if start, end, ok := q.queuedTimeRange(); ok {
		return q.rangeStart(start, end), end
	}

	end := time.Now()
	return end.Add(-q.chartSettings.Duration()), end
}

// queuedTimeRange excludes throughput counters: their history can extend beyond
// the queue-length samples actually drawn in this chart.
func (q *QueueDetails) queuedTimeRange() (time.Time, time.Time, bool) {
	var min, max int64
	include := func(samples []domain.Sample) {
		for _, sample := range samples {
			if min == 0 || sample.Timestamp < min {
				min = sample.Timestamp
			}
			if max == 0 || sample.Timestamp > max {
				max = sample.Timestamp
			}
		}
	}
	for _, name := range settings.MessageSeries() {
		if q.chartSettings.MessageVisible(name) {
			include(q.messageSamples(name))
		}
	}
	if min == max {
		return time.Time{}, time.Time{}, false
	}
	return time.UnixMilli(min), time.UnixMilli(max), true
}

func (q *QueueDetails) messageSamples(name string) []domain.Sample {
	switch name {
	case "Ready":
		return q.lastStats.MessagesReady
	case "Unacked":
		return q.lastStats.MessagesUnacked
	case "Total":
		return q.lastStats.MessagesTotal
	default:
		return nil
	}
}

// rangeStart clips older samples without inventing history the broker did not return.
func (q *QueueDetails) rangeStart(start, end time.Time) time.Time {
	if earliest := end.Add(-q.chartSettings.Duration()); start.Before(earliest) {
		return earliest
	}
	return start
}
