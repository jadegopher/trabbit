package widgets

import (
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
	"github.com/jadegopher/trabbit/internal/domain"
)

const minimumCounterSamples = 2

// Broker samples are newest first. Counter resets contribute zero, and duplicate
// timestamps cannot form a rate. Emit points in chronological order for charts.
func counterRatePoints(samples []domain.Sample) []timeserieslinechart.TimePoint {
	if len(samples) < minimumCounterSamples {
		return nil
	}
	points := make([]timeserieslinechart.TimePoint, 0, len(samples)-1)
	for i := len(samples) - 1; i > 0; i-- {
		earlier, later := samples[i], samples[i-1]
		seconds := float64(later.Timestamp-earlier.Timestamp) / float64(time.Second/time.Millisecond)
		if seconds <= 0 {
			continue
		}
		points = append(points, timeserieslinechart.TimePoint{
			Time: time.UnixMilli(later.Timestamp), Value: max(0, float64(later.Value-earlier.Value)/seconds),
		})
	}
	return points
}
