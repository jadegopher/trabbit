package refresher

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestRefreshWidget_UI(t *testing.T) {
	t.Parallel()

	theme := core.DefaultTheme()

	tests := []struct {
		name          string
		status        RefreshStatus
		interval      time.Duration
		lastUpdatedAt time.Time
	}{
		{
			name:          "refresh_widget_success",
			status:        RefreshStatusSuccess,
			interval:      5 * time.Second,
			lastUpdatedAt: time.Now().Add(-2 * time.Second),
		},
		{
			name:          "refresh_widget_error",
			status:        RefreshStatusError,
			interval:      5 * time.Second,
			lastUpdatedAt: time.Now().Add(-2 * time.Second),
		},
		{
			name:          "refresh_widget_error_past",
			status:        RefreshStatusError,
			interval:      5 * time.Second,
			lastUpdatedAt: time.Now().Add(-10 * time.Second),
		},
		{
			name:          "refresh_widget_error_zero",
			status:        RefreshStatusError,
			interval:      5 * time.Second,
			lastUpdatedAt: time.Time{},
		},
		{
			name:          "refresh_widget_pending",
			status:        RefreshStatusPending,
			interval:      5 * time.Second,
			lastUpdatedAt: time.Now().Add(-2 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				widget := NewWidget(theme, tt.status, time.Now, tt.interval)

				var msg tea.Msg
				switch tt.status {
				case RefreshStatusSuccess:
					msg = MsgRefreshComplete{}
				case RefreshStatusError:
					msg = MsgRefreshFailed{
						Err: fmt.Errorf("refresh failed"),
					}
				case RefreshStatusPending:
					msg = MsgRefreshRequested{}
				}

				widget.Update(msg)

				widget.lastUpdatedAt = tt.lastUpdatedAt

				view := widget.View()
				if got := lipgloss.Height(view.Content); got != 1 {
					t.Fatalf("refresh widget must fit the dashboard header: got %d rows", got)
				}
				core.AssertGolden(t, tt.name, view.Content)
			},
		)
	}
}
