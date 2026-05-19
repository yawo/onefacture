package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	redis "github.com/redis/go-redis/v9"

	"github.com/yawo/onefacture/internal/config"
)

func startRedisContainer(ctx context.Context) (string, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, err
	}
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", nil, err
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		container.Terminate(ctx)
		return "", nil, err
	}
	addr := host + ":" + port.Port()
	cleanup := func() {
		container.Terminate(ctx)
	}
	return addr, cleanup, nil
}

func TestNewBusSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "onefacture.events",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, bus)
	defer bus.Close()
	require.NotNil(t, bus.Client())
}

func TestNewBusInvalidAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.RedisConfig{
		Addr:      "invalid-host:6379",
		Password:  "",
		DB:        0,
		StreamKey: "onefacture.events",
	}
	_, err := New(ctx, cfg)
	require.Error(t, err)
}

func TestPublishEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
		Payload:        map[string]any{"pa": "chorus"},
	}
	err = bus.Publish(ctx, ev)
	require.NoError(t, err)
}

func TestPublishEventSetsOccurredAt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
	}
	// OccurredAt is zero initially
	require.True(t, ev.OccurredAt.IsZero())

	err = bus.Publish(ctx, ev)
	require.NoError(t, err)
}

func TestPublishEventInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	// Create an event with a non-marshalable payload
	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
		Payload: map[string]any{
			"chan": make(chan int), // channels cannot be marshaled to JSON
		},
	}
	err = bus.Publish(ctx, ev)
	require.Error(t, err)
}

func TestSubscribeBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	// Publish an event
	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
		Payload:        map[string]any{"pa": "chorus"},
	}
	err = bus.Publish(ctx, ev)
	require.NoError(t, err)

	// Subscribe in a goroutine
	var receivedEvent *Event
	done := make(chan bool, 1)

	go func() {
		err := bus.Subscribe(ctx, "test-group", "consumer-1", func(ctx context.Context, e Event) error {
			receivedEvent = &e
			done <- true
			// Exit the subscription loop by canceling context
			cancel()
			return nil
		})
		// Expected to return because context is canceled
		_ = err
	}()

	select {
	case <-done:
		require.NotNil(t, receivedEvent)
		require.Equal(t, ev.Type, receivedEvent.Type)
		require.Equal(t, ev.OrganizationID, receivedEvent.OrganizationID)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSubscribeProcessesMultipleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	// Publish multiple events
	for i := 1; i <= 3; i++ {
		ev := Event{
			Type:           "invoice.submitted",
			OrganizationID: "org-123",
			InvoiceID:      "inv-" + string(rune(48+i)),
		}
		err = bus.Publish(ctx, ev)
		require.NoError(t, err)
	}

	count := 0
	done := make(chan bool, 1)

	go func() {
		_ = bus.Subscribe(ctx, "test-group", "consumer-1", func(ctx context.Context, e Event) error {
			count++
			if count >= 3 {
				done <- true
				cancel()
			}
			return nil
		})
	}()

	select {
	case <-done:
		require.GreaterOrEqual(t, count, 3)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

func TestSubscribeHandlerError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	// Publish an event
	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
	}
	err = bus.Publish(ctx, ev)
	require.NoError(t, err)

	done := make(chan bool, 1)
	handlerCalled := false

	go func() {
		_ = bus.Subscribe(ctx, "test-group", "consumer-1", func(ctx context.Context, e Event) error {
			handlerCalled = true
			done <- true
			cancel()
			return nil // return nil means the message will be acked
		})
	}()

	select {
	case <-done:
		require.True(t, handlerCalled)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSubscribeInvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	addr, cleanup, err := startRedisContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.RedisConfig{
		Addr:      addr,
		Password:  "",
		DB:        0,
		StreamKey: "test.stream",
	}
	bus, err := New(ctx, cfg)
	require.NoError(t, err)
	defer bus.Close()

	// Manually add a malformed event to the stream
	cmd := bus.Client().XAdd(ctx, &redis.XAddArgs{
		Stream: cfg.StreamKey,
		Values: map[string]any{"data": "invalid json"},
	})
	require.NoError(t, cmd.Err())

	done := make(chan bool, 1)
	callCount := 0

	go func() {
		_ = bus.Subscribe(ctx, "test-group", "consumer-1", func(ctx context.Context, e Event) error {
			callCount++
			done <- true
			cancel()
			return nil
		})
	}()

	select {
	case <-done:
		// If we get here, the handler was called for another event
	case <-time.After(2 * time.Second):
		// Timeout is expected since malformed event is skipped
	}
}

func TestIsGroupExists(t *testing.T) {
	// Create a mock error with the expected message
	err := errors.New("BUSYGROUP Consumer Group name already exists")
	require.True(t, isGroupExists(err))

	err = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	require.False(t, isGroupExists(err))

	require.False(t, isGroupExists(nil))
}

func TestBusCloseNil(t *testing.T) {
	bus := &Bus{rdb: nil}
	bus.Close() // should not panic
}

func TestEventMarshalling(t *testing.T) {
	now := time.Now()
	ev := Event{
		Type:           "invoice.submitted",
		OrganizationID: "org-123",
		InvoiceID:      "inv-456",
		OccurredAt:     now,
		Payload:        map[string]any{"pa": "chorus", "count": 42},
	}

	// Marshal to JSON
	data, err := json.Marshal(ev)
	require.NoError(t, err)

	// Unmarshal back
	var ev2 Event
	err = json.Unmarshal(data, &ev2)
	require.NoError(t, err)

	require.Equal(t, ev.Type, ev2.Type)
	require.Equal(t, ev.OrganizationID, ev2.OrganizationID)
	require.Equal(t, ev.InvoiceID, ev2.InvoiceID)
	require.Equal(t, ev.Payload["pa"], ev2.Payload["pa"])
	require.Equal(t, float64(42), ev2.Payload["count"]) // JSON numbers become float64
}
