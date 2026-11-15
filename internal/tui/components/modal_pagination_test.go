package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

func TestPaginationModalValidatesDraftSettings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, page, size string
		count            int
		want             ActionPageChange
		err              string
	}{
		{name: "current size", page: "3", size: "3", count: 10, want: ActionPageChange{Page: 3, PageSize: 3}},
		{name: "smaller size adds pages", page: "10", size: "1", count: 10, want: ActionPageChange{Page: 10, PageSize: 1}},
		{name: "larger size removes pages", page: "3", size: "5", count: 10, err: "page must be a whole number from 1 to 2"},
		{name: "partial last page", page: "4", size: "3", count: 10, want: ActionPageChange{Page: 4, PageSize: 3}},
		{name: "maximum size", page: "1", size: "1000", count: 10, want: ActionPageChange{Page: 1, PageSize: 1000}},
		{name: "empty results", page: "1", size: "3", want: ActionPageChange{Page: 1, PageSize: 3}},
		{name: "empty second page", page: "2", size: "3", err: "page must"},
		{name: "zero page", page: "0", size: "3", count: 10, err: "page must"},
		{name: "negative page", page: "-1", size: "3", count: 10, err: "page must"},
		{name: "fractional page", page: "1.5", size: "3", count: 10, err: "page must"},
		{name: "blank page", size: "3", count: 10, err: "page must"},
		{name: "oversized page", page: "999999999999999999999", size: "3", count: 10, err: "page must"},
		{name: "zero size", page: "1", size: "0", count: 10, err: "page size must"},
		{name: "negative size", page: "1", size: "-1", count: 10, err: "page size must"},
		{name: "excessive size", page: "1", size: "1001", count: 10, err: "page size must"},
		{name: "fractional size", page: "1", size: "3.5", count: 10, err: "page size must"},
		{name: "blank size", page: "1", count: 10, err: "page size must"},
		{name: "text size", page: "1", size: "abc", count: 10, err: "page size must"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewPaginationModal(core.DefaultTheme(), domain.Pagination{
				Page: 3, PageSize: 3, PageCount: 4, FilteredCount: tc.count, TotalCount: 1000,
			}, 1000)
			m.fields[0].Item.(*TextInput).SetValue(tc.page)
			m.fields[1].Item.(*TextInput).SetValue(tc.size)
			value, err := m.OnSubmit()
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				require.Nil(t, value)
				_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
				require.Nil(t, cmd, "invalid drafts stay open")
				require.Error(t, m.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, value)
		})
	}
}

func TestPaginationModalInitialValuesAndRange(t *testing.T) {
	t.Parallel()

	m := NewPaginationModal(core.DefaultTheme(), domain.Pagination{Page: 3, PageSize: 3, FilteredCount: 10}, 1000)
	require.Equal(t, "3", m.fields[0].Item.(*TextInput).Value())
	require.Equal(t, "3", m.fields[1].Item.(*TextInput).Value())
	require.Contains(t, m.ExtraView(), "Page: 1–4")
	m.fields[1].Item.(*TextInput).SetValue("5")
	require.Contains(t, m.ExtraView(), "Page: 1–2")
	require.True(t, m.CloseOnSubmit)
}

func TestPaginationGoToShortcut(t *testing.T) {
	t.Parallel()

	p := NewPagination(core.DefaultTheme(), 1000)
	p.Update(domain.Pagination{Page: 2, PageSize: 3, FilteredCount: 10})
	_, cmd := p.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	require.NotNil(t, cmd)
	dialog := cmd().(core.OpenDialog).Model.(*FormModal)
	value, err := dialog.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, ActionPageChange{Page: 2, PageSize: 3}, value)

	_, cmd = p.Update(tea.KeyPressMsg{Code: 'G', Text: "G"})
	require.Nil(t, cmd, "GoTo uses lowercase g only")
	require.Contains(t, ansi.Strip(p.View().Content), "g GoTo")
	require.NotContains(t, p.View().Content, "Per 3")
	p.Update(ActionPageChange{Page: 3})
	_, cmd = p.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	require.Nil(t, cmd, "pending navigation cannot open a dialog with old settings")
}

func TestPaginationModalConfiguredLimit(t *testing.T) {
	t.Parallel()

	p := NewPagination(core.DefaultTheme(), 25)
	p.Update(domain.Pagination{Page: 1, PageSize: 3, FilteredCount: 100})
	_, cmd := p.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	m := cmd().(core.OpenDialog).Model.(*FormModal)
	require.Contains(t, m.ExtraView(), "Page size: 1–25")
	m.fields[1].Item.(*TextInput).SetValue("26")
	_, err := m.OnSubmit()
	require.ErrorContains(t, err, "from 1 to 25")
	m.fields[1].Item.(*TextInput).SetValue("25")
	value, err := m.OnSubmit()
	require.NoError(t, err)
	require.Equal(t, ActionPageChange{Page: 1, PageSize: 25}, value)
}
