package directory

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type countingProvider struct {
	calls int
	err   error
}

func (p *countingProvider) Resolve(_ context.Context, siren string) (Entry, error) {
	p.calls++
	if p.err != nil {
		return Entry{}, p.err
	}
	return Entry{SIREN: siren, PAID: "chorus", Resolved: true, Source: "primary"}, nil
}

func TestResolverCachesWithinTTL(t *testing.T) {
	primary := &countingProvider{}
	resolver := NewResolver(time.Minute, primary, FallbackProvider{PAID: "mock", Source: "fallback"})

	first, err := resolver.Resolve(context.Background(), "123456789")
	require.NoError(t, err)
	second, err := resolver.Resolve(context.Background(), "123456789")
	require.NoError(t, err)

	require.Equal(t, "chorus", first.PAID)
	require.Equal(t, "chorus", second.PAID)
	require.False(t, first.Cached)
	require.True(t, second.Cached)
	require.Equal(t, 1, primary.calls)
}

func TestResolverCachedLookupP95Under100ms(t *testing.T) {
	primary := &countingProvider{}
	resolver := NewResolver(time.Minute, primary, FallbackProvider{PAID: "mock", Source: "fallback"})
	ctx := context.Background()
	_, err := resolver.Resolve(ctx, "123456789")
	require.NoError(t, err)

	durations := make([]time.Duration, 1000)
	for i := range durations {
		start := time.Now()
		entry, err := resolver.Resolve(ctx, "123456789")
		durations[i] = time.Since(start)
		require.NoError(t, err)
		require.True(t, entry.Cached)
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95 := durations[int(float64(len(durations))*0.95)-1]
	require.Less(t, p95, 100*time.Millisecond)
	require.Equal(t, 1, primary.calls)
}

func TestResolverFallsBackWhenPrimaryFails(t *testing.T) {
	primary := &countingProvider{err: errors.New("primary down")}
	resolver := NewResolver(time.Minute, primary, FallbackProvider{PAID: "mock", Source: "fallback"})

	entry, err := resolver.Resolve(context.Background(), "123456789")

	require.NoError(t, err)
	require.Equal(t, "mock", entry.PAID)
	require.False(t, entry.Resolved)
	require.Equal(t, "fallback", entry.Source)
}

func TestStaticProviderNotFound(t *testing.T) {
	provider := StaticProvider{Source: "static", Entries: map[string]string{"123456789": "chorus"}}

	_, err := provider.Resolve(context.Background(), "000000000")

	require.ErrorIs(t, err, ErrNotFound)
}
