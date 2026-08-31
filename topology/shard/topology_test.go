package shard

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mkbeh/xch"
)

func TestNewTopologyRequiresShard(t *testing.T) {
	t.Parallel()

	topology, err := NewTopology()
	if err == nil {
		closeTopology(t, topology)
		t.Fatal("expected error")
	}

	if got, want := err.Error(),
		"xch/topology/shard: topology must contain at least one shard"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNewTopologyRejectsNilPool(t *testing.T) {
	t.Parallel()

	topology, err := NewTopology(nil)
	if err == nil {
		closeTopology(t, topology)
		t.Fatal("expected error")
	}

	if got, want := err.Error(),
		"xch/topology/shard: shard 0: pool is nil"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNewTopologyRejectsEmptyPoolName(t *testing.T) {
	t.Parallel()

	pool, err := xch.New(&clickhouse.Options{})
	if err != nil {
		t.Fatalf("xch.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = pool.Close()
	})

	topology, err := NewTopology(pool)
	if err == nil {
		closeTopology(t, topology)
		t.Fatal("expected error")
	}

	if got, want := err.Error(),
		"xch/topology/shard: shard 0: pool name must not be empty"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNewTopologyRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	first := newTestPool(t, "shard-a", nil)
	second := newTestPool(t, "shard-a", nil)

	topology, err := NewTopology(first, second)
	if err == nil {
		closeTopology(t, topology)
		t.Fatal("expected error")
	}

	if got, want := err.Error(),
		`xch/topology/shard: duplicate shard ID "shard-a"`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestTopologyPreservesRegistrationOrder(t *testing.T) {
	t.Parallel()

	topology := newTestTopology(
		t,
		"shard-b",
		"shard-a",
		"shard-c",
	)

	if got, want := topology.Len(), 3; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	want := []ID{
		"shard-b",
		"shard-a",
		"shard-c",
	}

	for index, wantID := range want {
		if got := topology.At(index).ID(); got != wantID {
			t.Fatalf(
				"At(%d).ID() = %q, want %q",
				index,
				got,
				wantID,
			)
		}

		resolved, ok := topology.Shard(wantID)
		if !ok {
			t.Fatalf("Shard(%q) not found", wantID)
		}

		if got := resolved.ID(); got != wantID {
			t.Fatalf(
				"Shard(%q).ID() = %q, want %q",
				wantID,
				got,
				wantID,
			)
		}
	}
}

func TestTopologyShardsReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	topology := newTestTopology(
		t,
		"shard-a",
		"shard-b",
	)

	shards := topology.Shards()
	shards[0] = Shard{}

	if got, want := topology.At(0).ID(), ID("shard-a"); got != want {
		t.Fatalf(
			"At(0).ID() = %q, want %q",
			got,
			want,
		)
	}
}

func TestTopologyShardUnknownID(t *testing.T) {
	t.Parallel()

	topology := newTestTopology(t, "shard-a")

	resolved, ok := topology.Shard("missing")
	if ok {
		t.Fatalf(
			"Shard() = %+v, true; want false",
			resolved,
		)
	}

	if got := resolved.ID(); got != "" {
		t.Fatalf(
			"Shard().ID() = %q, want empty",
			got,
		)
	}
}

func TestTopologyAtPanicsOutOfRange(t *testing.T) {
	t.Parallel()

	topology := newTestTopology(t, "shard-a")

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = topology.At(1)
}

func TestTopologyNilReceiver(t *testing.T) {
	t.Parallel()

	var topology *Topology

	if got := topology.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}

	if shards := topology.Shards(); shards != nil {
		t.Fatalf(
			"Shards() = %#v, want nil",
			shards,
		)
	}

	if resolved, ok := topology.Shard("shard-a"); ok || resolved.ID() != "" {
		t.Fatalf(
			"Shard() = %+v, %v; want zero, false",
			resolved,
			ok,
		)
	}

	closeTopology(t, topology)
}

func TestTopologyCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	topology := newTestTopology(
		t,
		"shard-a",
		"shard-b",
	)

	closeTopology(t, topology)
	closeTopology(t, topology)
}
