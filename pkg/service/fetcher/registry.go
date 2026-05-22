// Package fetcher hosts the provider registry and the shared
// SSRF-protected HTTP client. There is intentionally NO global registry
// and NO init()-time side effects — providers are constructed by name
// in pkg/service/providers and handed to a *Registry at startup. The
// CLI / usecase layer carries the Registry through usecase.Deps.
package fetcher

import (
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Registry maps a Source.Type identifier to a concrete Provider. It is
// safe for concurrent reads after Register has finished (typical usage:
// build at startup, read for the life of the process).
type Registry struct {
	providers map[string]interfaces.Provider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{providers: map[string]interfaces.Provider{}}
}

// Register installs p under typeID. Returns an error (tagged Conflict)
// on duplicate type ids rather than panicking — duplicate registrations
// are configuration mistakes, not programmer mistakes.
func (r *Registry) Register(typeID string, p interfaces.Provider) error {
	if typeID == "" {
		return goerr.New("provider type id is empty", goerr.T(errutil.TagInvalidInput))
	}
	if _, ok := r.providers[typeID]; ok {
		return goerr.New("duplicate provider type",
			goerr.V("type", typeID),
			goerr.T(errutil.TagConflict))
	}
	r.providers[typeID] = p
	return nil
}

// MustRegister panics on conflict. Useful in tests where the typeID is
// generated and uniqueness is guaranteed.
func (r *Registry) MustRegister(typeID string, p interfaces.Provider) {
	if err := r.Register(typeID, p); err != nil {
		panic(err)
	}
}

// Resolve returns the provider for typeID, or a tagged not-found error
// when none is registered. Used by the usecase layer when dispatching a
// Source.
func (r *Registry) Resolve(typeID string) (interfaces.Provider, error) {
	if r == nil {
		return nil, goerr.New("registry is nil")
	}
	p, ok := r.providers[typeID]
	if !ok {
		return nil, goerr.New("unknown provider type",
			goerr.V("type", typeID),
			goerr.T(errutil.TagNotFound))
	}
	return p, nil
}

// Types returns every registered provider type in stable sorted order.
// Used by `beehive validate` for the did-you-mean hint.
func (r *Registry) Types() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Len reports the number of registered providers.
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.providers)
}
