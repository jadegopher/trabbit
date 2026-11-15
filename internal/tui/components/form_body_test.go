package components

import (
	"testing"

	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
)

func TestFormBodyNavigationBoundaries(t *testing.T) {
	t.Parallel()
	for _, navigation := range []struct {
		name string
		wrap bool
	}{{"stop", false}, {"wrap", true}} {
		t.Run(navigation.name, func(t *testing.T) {
			t.Parallel()
			theme := core.DefaultTheme()
			first := NewTextInput(theme)
			last := NewTextInput(theme)
			body := NewFormBody(theme, []FormField{{Item: first}, {Item: last}})
			body.WrapNavigation = navigation.wrap
			body.SetSize(40, 5)
			body.Focus()

			moved, _ := body.Move(true)
			require.Equal(t, navigation.wrap, moved)
			if navigation.wrap {
				require.Same(t, last, body.FocusedItem())
				require.False(t, first.Focused())
				body.Move(false)
			}
			require.Same(t, first, body.FocusedItem())
			require.True(t, first.Focused())

			body.Move(false)
			moved, _ = body.Move(false)
			require.Equal(t, navigation.wrap, moved)
			if navigation.wrap {
				require.Same(t, first, body.FocusedItem())
				require.True(t, first.Focused())
				require.False(t, last.Focused())
			} else {
				require.Same(t, last, body.FocusedItem())
				require.True(t, last.Focused())
			}
		})
	}
}
