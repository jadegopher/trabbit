package components

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/core"
)

type ActionPageChange struct {
	// Owner identifies independent pagers hosted by the same scene.
	Owner string
	Page  int
	// Zero keeps the current page size when navigating with the arrow keys.
	PageSize int
}

type ActionPageChangeComplete struct{}

type paginationShortcuts struct {
	NextPage key.Binding
	PrevPage key.Binding
	GoTo     key.Binding
}

func (s paginationShortcuts) ShortHelp() []key.Binding {
	return []key.Binding{s.PrevPage, s.GoTo, s.NextPage}
}

func (s paginationShortcuts) FullHelp() [][]key.Binding {
	return [][]key.Binding{s.ShortHelp()}
}

type Pagination struct {
	theme       core.Theme
	shortcuts   paginationShortcuts
	pagination  domain.Pagination
	targetPage  int
	maxPageSize int
	status      string
	owner       string
}

const ComponentName = "pagination"

func NewPagination(theme core.Theme, maxPageSize int) *Pagination {
	return &Pagination{
		theme:       theme,
		maxPageSize: max(1, maxPageSize),
		shortcuts: paginationShortcuts{
			NextPage: core.NextPage(),
			PrevPage: core.PreviousPage(),
			GoTo:     core.GoToPage(),
		},
	}
}

func (p *Pagination) Init() tea.Cmd { return nil }

func (p *Pagination) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case domain.Pagination:
		if p.targetPage == 0 || msg.Page == p.targetPage {
			p.pagination = msg
			p.targetPage = 0
		}
	case ActionPageChange:
		p.targetPage = msg.Page
		return p, nil
	case ActionPageChangeComplete:
		p.targetPage = 0
	case tea.KeyPressMsg:
		if p.targetPage != 0 {
			return p, nil
		}
		switch {
		case key.Matches(msg, p.shortcuts.GoTo):
			return p, core.Cmd(core.OpenDialog{Model: p.SettingsModal()})
		case key.Matches(msg, p.shortcuts.NextPage):
			if p.pagination.Page < p.pagination.PageCount {
				return p, core.Cmd(ActionPageChange{Owner: p.owner, Page: p.pagination.Page + 1})
			}
		case key.Matches(msg, p.shortcuts.PrevPage):
			if p.pagination.Page > 1 {
				return p, core.Cmd(ActionPageChange{Owner: p.owner, Page: p.pagination.Page - 1})
			}
		}
	}
	return p, nil
}

func (p *Pagination) SetOwner(owner string) { p.owner = owner }

// SettingsModal captures the pager owner so a later submit cannot target another list.
func (p *Pagination) SettingsModal() *FormModal {
	modal := NewPaginationModal(p.theme, p.pagination, p.maxPageSize)
	submit, owner := modal.OnSubmit, p.owner
	modal.OnSubmit = func() (tea.Msg, error) {
		value, err := submit()
		if change, ok := value.(ActionPageChange); ok {
			change.Owner = owner
			value = change
		}
		return value, err
	}
	return modal
}

func (p *Pagination) View() tea.View {
	return tea.NewView(p.render(p.theme.PlainStyle))
}

func (p *Pagination) SetPagination(pag domain.Pagination) {
	p.pagination = pag
}

// SetStatus adds context beside the page count, above the navigation controls.
func (p *Pagination) SetStatus(status string) {
	p.status = status
}

func (p *Pagination) render(style lipgloss.Style) string {
	pag := p.pagination
	if pag.PageCount == 0 {
		pag.PageCount = 1
	}
	if pag.Page == 0 {
		pag.Page = 1
	}

	normalInfo := fmt.Sprintf("%d of %d", pag.Page, pag.PageCount)
	infoStyle := p.theme.FooterStyle

	info := infoStyle.Render(normalInfo)

	if p.status != "" {
		info += p.theme.RenderHelpSeparator() + infoStyle.Render(p.status)
	}
	footer := info + "\n" + p.theme.RenderHelp(p.shortcuts)

	return style.Render(footer)
}

func (p *Pagination) ShortHelp() []key.Binding {
	return p.shortcuts.ShortHelp()
}
