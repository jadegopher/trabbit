package components

import (
	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPaginationRequestsAndRecovery(t *testing.T) {
	t.Parallel()

	p := NewPagination(core.DefaultTheme(), 1000)
	p.Update(domain.Pagination{Page: 1, PageCount: 5})
	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.NotNil(t, cmd)
	require.Equal(t, ActionPageChange{Page: 2}, cmd())
	_, tick := p.Update(ActionPageChange{Page: 2})
	require.Nil(t, tick)
	require.Contains(t, p.View().Content, "1 of 5")
	require.NotContains(t, p.View().Content, "Loading")
	_, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.Nil(t, cmd)
	p.Update(ActionPageChangeComplete{})
	_, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.NotNil(t, cmd)
	p.Update(domain.Pagination{Page: 5, PageCount: 5})
	_, cmd = p.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	require.Nil(t, cmd)
}
