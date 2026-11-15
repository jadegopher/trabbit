package widgets

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/settings"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestConsumersPageSnapshotWithoutCursor(t *testing.T) {
	t.Parallel()

	c := NewQueueConsumers(core.DefaultTheme())
	c.SetSize(260, 12)
	c.SetFocused(true)
	c.SetLoading("/", "orders")
	data := make([]domain.QueueConsumer, 120)
	active := true
	timeout := int64(1800000)
	for i := range data {
		data[i] = domain.QueueConsumer{Channel: "172.19.0.1:51754 (4)", Tag: fmt.Sprintf("consumer-%03d", i), Active: &active, TimeoutMillis: &timeout, AckRequired: true, PrefetchCount: 10000, ActivityStatus: "up"}
	}
	c.SetData(data)
	view := ansi.Strip(c.View().Content)
	for _, column := range []string{"Channel", "Consumer tag", "Ack required", "Exclusive", "Prefetch count", "Active", "Activity status", "Timeout (seconds)", "Arguments"} {
		require.Contains(t, view, column)
	}
	require.Contains(t, view, "consumer-000")
	require.Contains(t, view, "1800")
	require.Contains(t, view, "●")
	require.Contains(t, view, "○")
	for step := 0; step < 10; step++ {
		c.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	require.Equal(t, -1, c.table.Cursor())
	c.Update(components.ActionPageChange{Owner: c.PageOwner(), Page: 2})
	require.Contains(t, c.View().Content, "consumer-100")
	require.Contains(t, c.View().Content, "2 of 2")
	c.SetData(data[:1])
	require.Equal(t, 1, c.page)
	require.Contains(t, c.View().Content, "consumer-000")
	oldOwner := c.PageOwner()
	c.SetLoading("/", "other")
	c.Update(components.ActionPageChange{Owner: oldOwner, Page: 2, PageSize: 2})
	require.Equal(t, settings.DefaultDetailPageSize, c.pageSize)
}

func TestBindingSelectionPreservesIdentityAndProtectsDefault(t *testing.T) {
	t.Parallel()

	b := NewQueueBindings(core.DefaultTheme())
	b.SetSize(150, 10)
	b.SetFocused(true)
	b.SetLoading("/", "orders")
	bindings := []domain.QueueBinding{
		{VHost: "/", Queue: "orders", RoutingKey: "orders"},
		{VHost: "/", Queue: "orders", Source: "first", PropertiesKey: "one"},
		{VHost: "/", Queue: "orders", Source: "second", PropertiesKey: "two"},
	}
	b.SetData(bindings[:1])
	require.Nil(t, b.ActiveBinding())
	require.Contains(t, b.View().Content, "Default exchange binding")
	b.SetData(bindings)
	require.Equal(t, "first", b.ActiveBinding().Source)
	b.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, "second", b.ActiveBinding().Source)
	b.SetData([]domain.QueueBinding{bindings[2], bindings[0], bindings[1]})
	require.Equal(t, "second", b.ActiveBinding().Source)
	b.SetLoading("/", "other")
	require.Nil(t, b.ActiveBinding(), "a new queue cannot act on old bindings during debounce")
}

func TestAddBindingFormUsesCapturedTargetAndTypedArguments(t *testing.T) {
	t.Parallel()

	m := NewModalAddBinding(core.DefaultTheme(), "/team", "orders")
	m.Init()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, err := m.OnSubmit()
	require.Error(t, err)
	typeText := func(value string) {
		for _, r := range value {
			m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
	}
	typeText("events")
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	typeText("enabled")
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	typeText("true")
	result, err := m.OnSubmit()
	require.NoError(t, err)
	p := result.(ActionCreateBinding).Params
	require.Equal(t, "/team", p.VHost)
	require.Equal(t, "orders", p.Queue)
	require.Equal(t, "events", p.Source)
	require.Empty(t, p.RoutingKey)
	require.Equal(t, map[string]interface{}{"enabled": true}, p.Arguments)
}

func TestRuntimeChartContainmentAndMissingMetrics(t *testing.T) {
	t.Parallel()

	r := NewQueueRuntime(core.DefaultTheme())
	r.SetFocused(true)
	zero, heap := int64(0), int64(46422)
	data := domain.QueueRuntime{
		Reductions:       &domain.RateDetails{Rate: 288181, Samples: []domain.Sample{{Timestamp: 70000, Value: 600000}, {Timestamp: 40000, Value: 300000}, {Timestamp: 10000, Value: 0}}},
		MinBinVHeapWords: &heap, MinorGCs: &zero,
	}
	for _, width := range []int{100, 28, 0, 100} {
		r.SetSize(width, 10)
		r.SetData(data)
		if width == 0 {
			require.Empty(t, r.View().Content)
			continue
		}
		for step := 0; step < 40; step++ {
			r.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			before := r.offset
			view := ansi.Strip(r.View().Content)
			require.Equal(t, before, r.offset)
			require.Equal(t, width+core.BorderOverhead, lipgloss.Width(view))
			require.Equal(t, 10+core.BorderOverhead, lipgloss.Height(view))
			require.Contains(t, view, "▐│")
			for _, line := range strings.Split(view, "\n") {
				require.Equal(t, width+core.BorderOverhead, lipgloss.Width(line))
			}
		}
	}
	require.Contains(t, ansi.Strip(r.content()), "Minor GCs")
	require.Contains(t, ansi.Strip(r.content()), "0")
	require.Contains(t, ansi.Strip(r.content()), "Minimum heap (words)")
	require.Contains(t, ansi.Strip(r.content()), "—")
	r.SetData(domain.QueueRuntime{})
	require.Contains(t, r.content(), "Reductions: unavailable")
}

func TestCounterRatesSkipDuplicateTimesAndClampResets(t *testing.T) {
	t.Parallel()

	points := counterRatePoints([]domain.Sample{{Timestamp: 10000, Value: 10}, {Timestamp: 5000, Value: 100}, {Timestamp: 5000, Value: 50}, {Timestamp: 0, Value: 0}})
	require.Len(t, points, 2)
	require.Equal(t, float64(10), points[0].Value)
	require.Zero(t, points[1].Value)
}

func TestBindingPaginationSelectionAndStaleOwner(t *testing.T) {
	t.Parallel()
	b := NewQueueBindings(core.DefaultTheme())
	b.SetSize(100, 12)
	b.SetFocused(true)
	b.SetLoading("/", "orders")
	data := make([]domain.QueueBinding, 120)
	for i := range data {
		data[i] = domain.QueueBinding{VHost: "/", Queue: "orders", Source: fmt.Sprintf("exchange-%03d", i), PropertiesKey: fmt.Sprint(i)}
	}
	data[0].Source = ""
	b.SetData(data)
	require.Equal(t, "exchange-001", b.ActiveBinding().Source)
	command := b.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.NotNil(t, command)
	b.Update(command())
	require.Equal(t, "exchange-100", b.ActiveBinding().Source)
	require.Contains(t, ansi.Strip(b.View().Content), "2 of 2")
	require.Contains(t, ansi.Strip(b.View().Content), "Total: 120")
	b.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Equal(t, "exchange-101", b.ActiveBinding().Source)
	b.SetData(data)
	require.Equal(t, "exchange-101", b.ActiveBinding().Source)
	b.SetData(data[:2])
	require.Equal(t, 1, b.page)
	require.Equal(t, "exchange-001", b.ActiveBinding().Source)
	owner := b.PageOwner()
	b.SetLoading("/", "other")
	b.Update(components.ActionPageChange{Owner: owner, Page: 2, PageSize: 2})
	require.Equal(t, 1, b.page)
	require.Equal(t, settings.DefaultDetailPageSize, b.pageSize)
	require.Nil(t, b.ActiveBinding())
}

func TestConsumersNarrowLayoutIncludesTimeoutAndArguments(t *testing.T) {
	t.Parallel()
	c := NewQueueConsumers(core.DefaultTheme())
	c.SetSize(60, 14)
	c.SetFocused(true)
	timeout := int64(1800001)
	c.SetData([]domain.QueueConsumer{{Channel: "channel-1", Tag: "consumer-1", TimeoutMillis: &timeout, Arguments: map[string]interface{}{"format": "json"}}})
	seen := ""
	for step := 0; step < 30; step++ {
		view := ansi.Strip(c.View().Content)
		seen += view
		require.Equal(t, 62, lipgloss.Width(view))
		require.Equal(t, 16, lipgloss.Height(view))
		c.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	require.Contains(t, seen, "Timeout (seconds)")
	require.Contains(t, seen, "1800.001")
	require.Contains(t, seen, `{"format":"json"}`)
	require.Contains(t, seen, "Total: 1")
	require.NotContains(t, seen, "Consumers:")
	require.Equal(t, "—", consumerTimeoutSeconds(nil))
	zero := int64(0)
	require.Equal(t, "0", consumerTimeoutSeconds(&zero))
}
