package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/domain"
)

type bindingInfo struct {
	Source        string                 `json:"source"`
	RoutingKey    string                 `json:"routing_key"`
	PropertiesKey string                 `json:"properties_key"`
	Arguments     map[string]interface{} `json:"arguments"`
}

func bindingPath(vhost, source, queue string) string {
	return "/api/bindings/" + url.PathEscape(vhost) + "/e/" + url.PathEscape(source) + "/q/" + url.PathEscape(queue)
}

func (s Client) ListQueueBindings(ctx context.Context, vhost, queue string) ([]domain.QueueBinding, error) {
	var result []bindingInfo
	if err := s.request(ctx, http.MethodGet, queuePath(vhost, queue)+"/bindings", nil, &result); err != nil {
		return nil, err
	}
	bindings := make([]domain.QueueBinding, len(result))
	for i, b := range result {
		bindings[i] = domain.QueueBinding{
			VHost: vhost, Queue: queue, Source: b.Source, RoutingKey: b.RoutingKey,
			PropertiesKey: b.PropertiesKey, Arguments: b.Arguments,
		}
	}
	return bindings, nil
}

func (s Client) CreateQueueBinding(ctx context.Context, p app.CreateBindingParams) error {
	body := struct {
		RoutingKey string                 `json:"routing_key"`
		Arguments  map[string]interface{} `json:"arguments"`
	}{RoutingKey: p.RoutingKey, Arguments: p.Arguments}
	if body.Arguments == nil {
		body.Arguments = map[string]interface{}{}
	}
	return s.request(ctx, http.MethodPost, bindingPath(p.VHost, p.Source, p.Queue), body, nil)
}

func (s Client) DeleteQueueBinding(ctx context.Context, b domain.QueueBinding) error {
	return s.request(ctx, http.MethodDelete, bindingPath(b.VHost, b.Source, b.Queue)+"/"+url.PathEscape(b.PropertiesKey), nil, nil)
}
