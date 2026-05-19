package webhooks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignDeterministic(t *testing.T) {
	a := sign([]byte("secret"), []byte("hello"))
	b := sign([]byte("secret"), []byte("hello"))
	require.Equal(t, a, b)
	require.NotEqual(t, a, sign([]byte("other"), []byte("hello")))
}
