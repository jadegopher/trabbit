package client

import (
	"context"
	"net/http"

	rmqc "github.com/michaelklishin/rabbit-hole/v3"

	"github.com/jadegopher/trabbit/internal/domain"
)

func (s Client) ListVHosts(ctx context.Context) (domain.VHosts, error) {
	var vhosts []rmqc.VhostInfo
	if err := s.request(ctx, http.MethodGet, "/api/vhosts", nil, &vhosts); err != nil {
		return nil, err
	}
	return fromVHosts(vhosts), nil
}

func fromVHosts(v []rmqc.VhostInfo) domain.VHosts {
	result := make(domain.VHosts, 0, len(v))
	for _, elem := range v {
		result = append(result, fromVHost(elem))
	}

	return result
}

func fromVHost(v rmqc.VhostInfo) domain.VHost {
	return domain.VHost{
		Name:    v.Name,
		Tracing: v.Tracing,
		Tags:    v.Tags,
	}
}
