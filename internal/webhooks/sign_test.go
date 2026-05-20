package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignDeterministic(t *testing.T) {
	a := sign([]byte("secret"), []byte("hello"))
	b := sign([]byte("secret"), []byte("hello"))
	require.Equal(t, a, b)
	require.NotEqual(t, a, sign([]byte("other"), []byte("hello")))
}

func TestSignCorrectFormat(t *testing.T) {
	result := sign([]byte("secret"), []byte("message"))
	// Should be hex encoded
	_, err := hex.DecodeString(result)
	require.NoError(t, err)
	// Should match expected HMAC-SHA256
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("message"))
	expected := hex.EncodeToString(mac.Sum(nil))
	require.Equal(t, expected, result)
}

func TestSignEmptySecret(t *testing.T) {
	result := sign([]byte(""), []byte("message"))
	mac := hmac.New(sha256.New, []byte(""))
	mac.Write([]byte("message"))
	expected := hex.EncodeToString(mac.Sum(nil))
	require.Equal(t, expected, result)
}

func TestSignEmptyMessage(t *testing.T) {
	result := sign([]byte("secret"), []byte(""))
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(""))
	expected := hex.EncodeToString(mac.Sum(nil))
	require.Equal(t, expected, result)
}

func TestBackoffCalculation(t *testing.T) {
	before := time.Now().UTC()
	result := backoff(0)
	after := time.Now().UTC()

	// For attempt 0: 2^0 = 1 second
	expectedDuration := 1 * time.Second
	require.True(t, result.After(before.Add(expectedDuration).Add(-100*time.Millisecond)))
	require.True(t, result.Before(after.Add(expectedDuration).Add(100*time.Millisecond)))
}

func TestBackoffExponential(t *testing.T) {
	t1 := backoff(0)
	t2 := backoff(1)
	t3 := backoff(2)
	t4 := backoff(3)

	// Each should be further in the future
	now := time.Now().UTC()
	dur1 := t1.Sub(now)
	dur2 := t2.Sub(now)
	dur3 := t3.Sub(now)
	dur4 := t4.Sub(now)

	// Allow for some clock drift
	require.Less(t, dur1, dur2)
	require.Less(t, dur2, dur3)
	require.Less(t, dur3, dur4)
}

func TestBackoffCappedAt1Hour(t *testing.T) {
	// With 20 attempts, 2^20 seconds would be way over an hour
	result := backoff(20)
	now := time.Now().UTC()
	dur := result.Sub(now)

	// Should be capped at 1 hour
	require.LessOrEqual(t, dur, 1*time.Hour+1*time.Second) // +1s for clock drift
	require.Greater(t, dur, 59*time.Minute)
}
