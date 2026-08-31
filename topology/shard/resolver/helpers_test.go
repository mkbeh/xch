package resolver

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mkbeh/xch"
	"github.com/mkbeh/xch/topology/shard"
)

func newTestTopology(
	t *testing.T,
	ids ...shard.ID,
) *shard.Topology {
	t.Helper()

	pools := make([]*xch.Pool, len(ids))

	for index, id := range ids {
		pool, err := xch.New(
			&clickhouse.Options{},
			xch.WithName(string(id)),
		)
		if err != nil {
			t.Fatalf("xch.New() error = %v", err)
		}

		pools[index] = pool
	}

	topology, err := shard.NewTopology(pools...)
	if err != nil {
		for _, pool := range pools {
			_ = pool.Close()
		}

		t.Fatalf("shard.NewTopology() error = %v", err)
	}

	t.Cleanup(func() {
		_ = topology.Close()
	})

	return topology
}
