package components

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

// NewPaginationModal validates the destination against the draft size, so both
// settings can be changed together without first fetching a different page.
func NewPaginationModal(theme core.Theme, pag domain.Pagination, maxPageSize int) *FormModal {
	page := NewTextInput(theme)
	page.SetValue(strconv.Itoa(max(1, pag.Page)))
	size := NewTextInput(theme)
	size.SetValue(strconv.Itoa(max(1, pag.PageSize)))

	m := NewFormModal(theme, "GoTo page", []FormField{
		{Label: "Page: ", Item: page},
		{Label: "Page size: ", Item: size},
	})
	m.CloseOnSubmit = true
	m.ExtraView = func() string {
		pageSize, err := paginationPageSize(size.Value(), maxPageSize)
		if err != nil {
			return fmt.Sprintf("Page size: 1–%d", maxPageSize)
		}
		return fmt.Sprintf("Page: 1–%d • Page size: 1–%d", paginationPageCount(pag.FilteredCount, pageSize), maxPageSize)
	}
	m.OnSubmit = func() (tea.Msg, error) {
		pageSize, err := paginationPageSize(size.Value(), maxPageSize)
		if err != nil {
			return nil, err
		}
		pageCount := paginationPageCount(pag.FilteredCount, pageSize)
		destination, err := strconv.Atoi(strings.TrimSpace(page.Value()))
		if err != nil || destination < 1 || destination > pageCount {
			return nil, fmt.Errorf("page must be a whole number from 1 to %d", pageCount)
		}
		return ActionPageChange{Page: destination, PageSize: pageSize}, nil
	}
	return m
}

func paginationPageSize(value string, maxPageSize int) (int, error) {
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || size < 1 || size > maxPageSize {
		return 0, fmt.Errorf("page size must be a whole number from 1 to %d", maxPageSize)
	}
	return size, nil
}

func paginationPageCount(count, size int) int {
	// Empty results still have page 1. Divide before rounding to avoid overflow.
	count = max(0, count)
	pages := count / size
	if count%size != 0 {
		pages++
	}
	return max(1, pages)
}
