package refresher

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"time"

	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type RefreshIntervalOption struct {
	Label    string
	Duration time.Duration
}

func (o RefreshIntervalOption) String() string {
	return o.Label
}

func NewSetRefreshModal(theme core.Theme) tea.Model {
	m := components.NewSelection("Set Refresh Interval", RefreshIntervalOptions(), theme)
	m.OnSelect = func(i int) tea.Msg {
		return IntervalSelected{Duration: RefreshIntervalOptions()[i].(RefreshIntervalOption).Duration}
	}
	return m
}

func RefreshIntervalOptions() []fmt.Stringer {
	return []fmt.Stringer{
		RefreshIntervalOption{Label: "5 sec", Duration: 5 * time.Second},
		RefreshIntervalOption{Label: "10 sec", Duration: 10 * time.Second},
		RefreshIntervalOption{Label: "30 sec", Duration: 30 * time.Second},
		RefreshIntervalOption{Label: "1 min", Duration: 1 * time.Minute},
		RefreshIntervalOption{Label: "Disabled", Duration: 0},
	}
}

type IntervalSelected struct{ Duration time.Duration }
