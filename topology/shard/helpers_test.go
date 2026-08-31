package shard

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mkbeh/xch"
)

func newTestPool(
	t *testing.T,
	id ID,
	labels map[string]string,
) *xch.Pool {
	t.Helper()

	options := []xch.Option{
		xch.WithName(string(id)),
	}
	if labels != nil {
		options = append(
			options,
			xch.WithLabels(labels),
		)
	}

	pool, err := xch.New(
		&clickhouse.Options{},
		options...,
	)
	if err != nil {
		t.Fatalf("xch.New() error = %v", err)
	}

	t.Cleanup(func() {
		_ = pool.Close()
	})

	return pool
}

func newTestTopology(t *testing.T, ids ...ID) *Topology {
	t.Helper()

	pools := make([]*xch.Pool, len(ids))
	for index, id := range ids {
		pools[index] = newTestPool(t, id, nil)
	}

	topology, err := NewTopology(pools...)
	if err != nil {
		t.Fatalf("NewTopology() error = %v", err)
	}

	t.Cleanup(func() {
		closeTopology(t, topology)
	})

	return topology
}

func closeTopology(t *testing.T, topology *Topology) {
	t.Helper()

	if err := topology.Close(); err != nil {
		t.Errorf("Topology.Close() error = %v", err)
	}
}

type testResolverFunc[K any] func(K) (Shard, error)

func (resolve testResolverFunc[K]) Resolve(key K) (Shard, error) {
	return resolve(key)
}
