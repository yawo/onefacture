// Package registry resolves which PA adapter to use for a given organization.
package registry

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/chorus"
	"github.com/yawo/onefacture/internal/adapters/docaposte"
	"github.com/yawo/onefacture/internal/adapters/mock"
	"github.com/yawo/onefacture/internal/adapters/pennylane"
	"github.com/yawo/onefacture/internal/reliability"
)

// Registry holds named adapters and resolves them by org configuration.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]adapters.PAAdapter
	logger   *slog.Logger
}

// NewDefault returns a registry pre-populated with the bundled adapters.
func NewDefault(logger *slog.Logger) *Registry {
	r := &Registry{
		adapters: map[string]adapters.PAAdapter{},
		logger:   logger,
	}
	r.Register(reliability.WrapAdapter(mock.New()))
	r.Register(reliability.WrapAdapter(chorus.New()))
	r.Register(reliability.WrapAdapter(pennylane.New()))
	r.Register(reliability.WrapAdapter(docaposte.New()))
	return r
}

// Register adds an adapter; later calls override earlier ones with the same name.
func (r *Registry) Register(a adapters.PAAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.Name()] = a
}

// Get resolves an adapter by name.
func (r *Registry) Get(name string) (adapters.PAAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if name == "" {
		name = "mock"
	}
	a, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("unknown PA adapter %q", name)
	}
	return a, nil
}

// Names returns all registered adapter names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.adapters))
	for n := range r.adapters {
		out = append(out, n)
	}
	return out
}
