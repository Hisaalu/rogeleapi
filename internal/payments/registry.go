package payments

import (
	"fmt"
	"strings"

	"github.com/Hisaalu/rogele/internal/config"
)

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(momoCfg config.MomoConfig, airtelCfg config.AirtelConfig) *Registry {
	r := &Registry{providers: make(map[string]Provider)}
	r.Register(NewMTNProvider(momoCfg))
	r.Register(NewAirtelProvider(airtelCfg))
	return r
}

func (r *Registry) Register(p Provider) {
	r.providers[strings.ToLower(p.Name())] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown payment provider: %s", name)
	}
	return p, nil
}

func (r *Registry) Enabled() []string {
	var out []string
	for name, p := range r.providers {
		if p.Enabled() {
			out = append(out, name)
		}
	}
	return out
}
