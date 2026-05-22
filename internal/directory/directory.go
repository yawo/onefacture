package directory

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("directory entry not found")

type Entry struct {
	SIREN     string    `json:"siren"`
	PAID      string    `json:"pa_id"`
	Resolved  bool      `json:"resolved"`
	Source    string    `json:"source"`
	Cached    bool      `json:"cached"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type Provider interface {
	Resolve(ctx context.Context, siren string) (Entry, error)
}

type StaticProvider struct {
	Source  string
	Entries map[string]string
}

func (p StaticProvider) Resolve(_ context.Context, siren string) (Entry, error) {
	paID, ok := p.Entries[siren]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return Entry{SIREN: siren, PAID: paID, Resolved: true, Source: p.Source}, nil
}

type FallbackProvider struct {
	PAID   string
	Source string
}

func (p FallbackProvider) Resolve(_ context.Context, siren string) (Entry, error) {
	return Entry{SIREN: siren, PAID: p.PAID, Resolved: false, Source: p.Source}, nil
}

type Resolver struct {
	ttl      time.Duration
	primary  Provider
	fallback Provider
	now      func() time.Time

	mu    sync.Mutex
	cache map[string]Entry
}

func NewResolver(ttl time.Duration, primary, fallback Provider) *Resolver {
	return &Resolver{
		ttl:      ttl,
		primary:  primary,
		fallback: fallback,
		now:      time.Now,
		cache:    map[string]Entry{},
	}
}

func (r *Resolver) Resolve(ctx context.Context, siren string) (Entry, error) {
	now := r.now()
	r.mu.Lock()
	if cached, ok := r.cache[siren]; ok && now.Before(cached.ExpiresAt) {
		cached.Cached = true
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	entry, err := r.primary.Resolve(ctx, siren)
	if err != nil && r.fallback != nil {
		entry, err = r.fallback.Resolve(ctx, siren)
	}
	if err != nil {
		return Entry{}, err
	}
	if r.ttl > 0 {
		entry.ExpiresAt = now.Add(r.ttl)
		r.mu.Lock()
		r.cache[siren] = entry
		r.mu.Unlock()
	}
	return entry, nil
}
