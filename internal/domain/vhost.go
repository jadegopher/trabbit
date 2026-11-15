package domain

import "github.com/samber/lo"

type VHosts []VHost

func (v VHosts) Names() []string {
	return lo.Map(
		v, func(item VHost, index int) string {
			return item.Name
		},
	)
}

type VHost struct {
	Name        string
	Tracing     bool
	Description string
	Tags        []string
}
