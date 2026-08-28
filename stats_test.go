package xch

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolStats(t *testing.T) {
	t.Parallel()

	pool, err := New(&clickhouse.Options{
		MaxOpenConns: 12,
		MaxIdleConns: 4,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })

	assert.Equal(t, PoolStats{
		OpenConns:    0,
		IdleConns:    0,
		MaxOpenConns: 12,
		MaxIdleConns: 4,
	}, pool.Stats())
}
