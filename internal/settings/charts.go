package settings

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

// ChartSettings applies the same presets to every chart in a view.
type ChartSettings struct {
	Style      string          `json:"style"`
	Size       string          `json:"size"`
	TimeRange  string          `json:"time_range"`
	Messages   map[string]bool `json:"messages,omitempty"`
	Throughput map[string]bool `json:"throughput,omitempty"`
}

// Series names are stable settings keys and chart dataset names, in display order.
func MessageSeries() []string {
	return []string{"Ready", "Unacked", "Total"}
}

func ThroughputSeries() []string {
	return []string{
		"Publish", "DeliverGet", "Ack", "Redeliver",
		"ReturnUnroutable", "DropUnroutable", "Deliver", "Get",
		"DeliverNoAck", "GetNoAck",
	}
}

// Missing overrides retain the choices used before series were configurable.
func (c ChartSettings) MessageVisible(name string) bool {
	if visible, ok := c.Messages[name]; ok {
		return visible
	}
	return true
}

func (c ChartSettings) ThroughputVisible(name string) bool {
	if visible, ok := c.Throughput[name]; ok {
		return visible
	}
	return name == "Publish" || name == "DeliverGet" || name == "Ack"
}

func (c ChartSettings) Clone() ChartSettings {
	c.Messages = maps.Clone(c.Messages)
	c.Throughput = maps.Clone(c.Throughput)
	return c
}

func DefaultChartSettings() ChartSettings {
	return ChartSettings{Style: "dotted", Size: "medium", TimeRange: "1m"}
}

func (c ChartSettings) Duration() time.Duration {
	duration, _ := time.ParseDuration(c.TimeRange)
	return duration
}

func (c ChartSettings) Validate() error {
	for name := range c.Messages {
		if !slices.Contains(MessageSeries(), name) {
			return fmt.Errorf("unknown messages series %q", name)
		}
	}
	for name := range c.Throughput {
		if !slices.Contains(ThroughputSeries(), name) {
			return fmt.Errorf("unknown throughput series %q", name)
		}
	}
	if c.Style != "dotted" && c.Style != "lines" {
		return fmt.Errorf("chart style must be dotted or lines")
	}
	if c.Size != "small" && c.Size != "medium" && c.Size != "large" {
		return fmt.Errorf("chart size must be small, medium or large")
	}
	switch c.TimeRange {
	case "1m", "10m", "30m", "1h":
		return nil
	default:
		return fmt.Errorf("chart time range must be 1m, 10m, 30m or 1h")
	}
}
