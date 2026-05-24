package reliability

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/metrics"
)

var ErrCircuitOpen = errors.New("adapter circuit breaker open")

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type CircuitBreaker struct {
	mu          sync.Mutex
	threshold   int
	cooldown    time.Duration
	failures    int
	openedUntil time.Time
	now         func() time.Time
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
	}
}

func (b *CircuitBreaker) before() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.threshold <= 0 || b.openedUntil.IsZero() || b.now().After(b.openedUntil) {
		return nil
	}
	return ErrCircuitOpen
}

func (b *CircuitBreaker) after(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		b.openedUntil = time.Time{}
		return
	}
	if b.threshold <= 0 {
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openedUntil = b.now().Add(b.cooldown)
	}
}

type Adapter struct {
	inner   adapters.PAAdapter
	breaker *CircuitBreaker
	policy  RetryPolicy
}

func WrapAdapter(inner adapters.PAAdapter) *Adapter {
	return &Adapter{
		inner:   inner,
		breaker: NewCircuitBreaker(3, 30*time.Second),
		policy:  RetryPolicy{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: 2 * time.Second},
	}
}

func NewAdapter(inner adapters.PAAdapter, breaker *CircuitBreaker, policy RetryPolicy) *Adapter {
	return &Adapter{inner: inner, breaker: breaker, policy: policy}
}

func (a *Adapter) Name() string { return a.inner.Name() }

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	start := time.Now()
	defer func() {
		metrics.AdapterCallDuration.WithLabelValues(a.inner.Name(), "submit").Observe(time.Since(start).Seconds())
	}()
	attempts := a.policy.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		if err := a.breaker.before(); err != nil {
			return nil, err
		}
		res, err := a.inner.Submit(ctx, inv)
		a.breaker.after(err)
		if err == nil {
			metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "submit", "success").Inc()
			return res, nil
		}
		if errors.Is(err, adapters.ErrNotImplemented) {
			return nil, err
		}
		last = err
		metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "submit", "error").Inc()
		if i < attempts-1 {
			if err := sleep(ctx, jitteredDelay(a.policy, i)); err != nil {
				return nil, err
			}
		}
	}
	return nil, last
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	start := time.Now()
	ev, err := a.inner.GetStatus(ctx, paRef)
	metrics.AdapterCallDuration.WithLabelValues(a.inner.Name(), "get_status").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "get_status", "error").Inc()
		return nil, err
	}
	metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "get_status", "success").Inc()
	return ev, nil
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	start := time.Now()
	ev, err := a.inner.Webhook(ctx, payload)
	metrics.AdapterCallDuration.WithLabelValues(a.inner.Name(), "webhook").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "webhook", "error").Inc()
		return nil, err
	}
	metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "webhook", "success").Inc()
	return ev, nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	start := time.Now()
	err := a.inner.HealthCheck(ctx)
	metrics.AdapterCallDuration.WithLabelValues(a.inner.Name(), "health").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "health", "error").Inc()
		return err
	}
	metrics.AdapterCallsTotal.WithLabelValues(a.inner.Name(), "health", "success").Inc()
	return nil
}

func jitteredDelay(policy RetryPolicy, attempt int) time.Duration {
	delay := policy.BaseDelay
	if delay <= 0 {
		return 0
	}
	for i := 0; i < attempt; i++ {
		delay *= 2
		if policy.MaxDelay > 0 && delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}
	jitterMax := delay / 4
	if jitterMax <= 0 {
		return delay
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(jitterMax)))
	if err != nil {
		return delay
	}
	return delay + time.Duration(n.Int64())
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
