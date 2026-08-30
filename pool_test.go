package xch

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDoesNotDial(t *testing.T) {
	t.Parallel()

	var dialed atomic.Bool

	pool, err := New(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		DialContext: func(context.Context, string) (net.Conn, error) {
			dialed.Store(true)
			return nil, errors.New("unexpected dial")
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	assert.False(t, dialed.Load())
}

func TestOpenDoesNotConnect(t *testing.T) {
	t.Parallel()

	pool, err := Open("clickhouse://127.0.0.1:0/default?dial_timeout=1ms")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
}

func TestOpenWrapsDSNError(t *testing.T) {
	t.Parallel()

	_, err := Open("clickhouse://")
	require.Error(t, err)
	assert.ErrorContains(t, err, "xch: parse DSN:")
}

func TestClientInfo(t *testing.T) {
	t.Parallel()

	info := ClientInfo("analytics-service", "1.4.2")

	require.Len(t, info.Products, 1)
	assert.Equal(t, "analytics-service", info.Products[0].Name)
	assert.Equal(t, "1.4.2", info.Products[0].Version)
}
