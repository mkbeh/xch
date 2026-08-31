package shard

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/mkbeh/xch"
)

// Topology is an immutable ordered set of logical shards.
//
// NewTopology takes ownership of all pools only after it returns successfully.
// Close closes every owned pool exactly once.
type Topology struct {
	shards     []Shard
	shardsByID map[ID]Shard

	closeOnce sync.Once
	closeErr  error
}

// NewTopology validates and creates an immutable topology.
//
// Shards retain pool registration order. Every pool must have a unique,
// non-empty name, which becomes the stable shard ID.
func NewTopology(pools ...*xch.Pool) (*Topology, error) {
	if len(pools) == 0 {
		return nil, errors.New("xch/topology/shard: topology must contain at least one shard")
	}

	shards := make([]Shard, len(pools))
	shardsByID := make(map[ID]Shard, len(pools))

	for index, pool := range pools {
		if pool == nil {
			return nil, fmt.Errorf("xch/topology/shard: shard %d: pool is nil", index)
		}

		id := ID(pool.Name())
		if id == "" {
			return nil, fmt.Errorf("xch/topology/shard: shard %d: pool name must not be empty", index)
		}

		if _, exists := shardsByID[id]; exists {
			return nil, fmt.Errorf("xch/topology/shard: duplicate shard ID %q", id)
		}

		current := Shard{pool: pool}
		shards[index] = current
		shardsByID[id] = current
	}

	return &Topology{
		shards:     shards,
		shardsByID: shardsByID,
	}, nil
}

// Len returns the number of registered shards.
func (t *Topology) Len() int {
	if t == nil {
		return 0
	}

	return len(t.shards)
}

// At returns the shard at index in registration order.
//
// At panics when index is outside the topology, matching ordinary slice
// indexing semantics.
func (t *Topology) At(index int) Shard {
	return t.shards[index]
}

// Shards returns a defensive copy of shards in registration order.
func (t *Topology) Shards() []Shard {
	if t == nil {
		return nil
	}

	return slices.Clone(t.shards)
}

// Shard returns one shard by stable ID.
func (t *Topology) Shard(id ID) (Shard, bool) {
	if t == nil {
		return Shard{}, false
	}

	resolved, ok := t.shardsByID[id]

	return resolved, ok
}

// Close closes owned pools in reverse registration order.
//
// Close is safe to call multiple times. All callers receive the result of the
// first close attempt.
func (t *Topology) Close() error {
	if t == nil {
		return nil
	}

	t.closeOnce.Do(func() {
		for _, current := range slices.Backward(t.shards) {
			if err := current.pool.Close(); err != nil {
				t.closeErr = errors.Join(
					t.closeErr,
					fmt.Errorf("xch/topology/shard: close shard %q: %w", current.ID(), err),
				)
			}
		}
	})

	return t.closeErr
}
