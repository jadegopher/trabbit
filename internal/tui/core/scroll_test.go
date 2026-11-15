package core

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestHandleScrollBoundsAndConsumesInput(t *testing.T) {
	t.Parallel()
	for _, input := range []struct {
		name string
		up   tea.Msg
		down tea.Msg
	}{
		{"keyboard", tea.KeyPressMsg{Code: tea.KeyUp}, tea.KeyPressMsg{Code: tea.KeyDown}},
		{"wheel", tea.MouseWheelMsg{Button: tea.MouseWheelUp}, tea.MouseWheelMsg{Button: tea.MouseWheelDown}},
	} {
		t.Run(input.name, func(t *testing.T) {
			t.Parallel()
			theme := DefaultTheme()
			const limit = 3
			offset := 0
			for range limit {
				require.True(t, theme.HandleScroll(input.up, &offset, limit))
				require.Zero(t, offset)
			}
			for expected := 1; expected <= limit; expected++ {
				require.True(t, theme.HandleScroll(input.down, &offset, limit))
				require.Equal(t, expected, offset)
			}
			for range limit {
				require.True(t, theme.HandleScroll(input.down, &offset, limit))
				require.Equal(t, limit, offset)
			}
			for expected := limit - 1; expected >= 0; expected-- {
				require.True(t, theme.HandleScroll(input.up, &offset, limit))
				require.Equal(t, expected, offset)
			}
			for _, bound := range []int{0, -1} {
				require.True(t, theme.HandleScroll(input.down, &offset, bound))
				require.Zero(t, offset)
			}
			require.False(t, theme.HandleScroll(tea.KeyPressMsg{Code: tea.KeyTab}, &offset, limit))
			require.Zero(t, offset)
		})
	}
}
