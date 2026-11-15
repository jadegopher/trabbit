package publish

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/tui/components"
	"github.com/jadegopher/trabbit/internal/tui/core"
	"github.com/jadegopher/trabbit/internal/tui/scenes/publish/widgets"
)

type result struct {
	id    uint64
	value app.PublishResult
	err   error
}

type vhostsResult struct {
	values domain.VHosts
	err    error
}

// Scene owns request lifetime; the reusable form owns draft and editing state.
type Scene struct {
	theme        core.Theme
	form         *widgets.PublishForm
	service      *app.Service
	cancel       context.CancelFunc
	vhostsCancel context.CancelFunc
	requestID    uint64
	closed       bool
}

func New(theme core.Theme, service *app.Service, target widgets.PublishTarget) *Scene {
	return &Scene{theme: theme, form: widgets.NewPublishForm(theme, target), service: service}
}

func (s *Scene) Init() tea.Cmd {
	if s.closed {
		return nil
	}
	focus := s.form.Focus()
	if s.vhostsCancel != nil {
		return focus
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.vhostsCancel = cancel
	service := s.service
	return tea.Batch(focus, func() tea.Msg {
		values, err := service.ListVHosts(ctx)
		return vhostsResult{values: values, err: err}
	})
}

func (s *Scene) Close() {
	s.closed = true
	s.form.Blur()
	if s.vhostsCancel != nil {
		s.vhostsCancel()
		s.vhostsCancel = nil
	}
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *Scene) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.closed {
		return s, nil
	}
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		s.form.SetSize(m.Width, m.Height)
		return s, nil
	case core.VisibilityMsg:
		if !m.Visible {
			s.form.Blur()
			return s, nil
		}
		return s, s.form.Focus()
	case vhostsResult:
		if s.vhostsCancel != nil {
			s.vhostsCancel()
			s.vhostsCancel = nil
			s.form.SetVHosts(m.values.Names(), m.err)
		}
		return s, nil
	case core.DialogResult:
		if back, ok := m.Value.(widgets.PublishBack); ok && back.Form == s.form {
			return s, core.Cmd(core.CloseScreen{Model: s})
		}
		return s, nil
	case widgets.PublishExit:
		if m.Form == s.form {
			return s, core.Cmd(core.OpenDialog{Model: components.NewConfirm("Discard draft and exit publish?", s.theme).WithResult(widgets.PublishBack{Form: s.form})})
		}
		return s, nil
	case widgets.PublishBack:
		if m.Form == s.form {
			return s, core.Cmd(core.CloseScreen{Model: s})
		}
		return s, nil
	case widgets.PublishRequest:
		if m.Form != s.form || !s.form.Pending(m.ID) || s.cancel != nil {
			return s, nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.requestID = m.ID
		service := s.service
		return s, func() tea.Msg {
			value, err := service.Publish(ctx, m.Target.VHost, m.Target.Exchange, m.Options)
			return result{id: m.ID, value: value, err: err}
		}
	case result:
		if m.id != s.requestID || s.cancel == nil || !s.form.Pending(m.id) {
			return s, nil
		}
		s.cancel()
		s.cancel = nil
		return s, s.form.Complete(m.id, m.value, m.err)
	}
	return s, s.form.Update(msg)
}

func (s *Scene) View() tea.View { return s.form.View() }
