package shard

import "github.com/mkbeh/xch"

// ID identifies one logical ClickHouse shard.
type ID string

// Shard is an immutable view of one Pool registered in a Topology.
//
// A Shard does not own the underlying Pool and is valid only for the lifetime
// of its owning Topology. The borrowed Pool must not be closed separately.
type Shard struct {
	pool *xch.Pool
}

// ID returns the stable logical shard ID inherited from the pool name.
func (s Shard) ID() ID {
	if s.pool == nil {
		return ""
	}

	return ID(s.pool.Name())
}

// Label returns one shard label without allocating a copy of all labels.
func (s Shard) Label(key string) (string, bool) {
	if s.pool == nil {
		return "", false
	}

	return s.pool.Label(key)
}

// Labels returns a defensive copy of shard labels.
func (s Shard) Labels() map[string]string {
	if s.pool == nil {
		return nil
	}

	return s.pool.Labels()
}

// Pool returns the pool backing this shard.
//
// The returned pool is borrowed and must not be closed separately.
func (s Shard) Pool() *xch.Pool {
	return s.pool
}
