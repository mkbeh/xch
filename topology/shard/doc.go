// Package shard provides application-level routing across ClickHouse shards.
//
// A Topology owns an immutable set of logical shards backed by xch.Pool values.
// Resolvers map application keys to shards using rendezvous hashing, ordered
// ranges, time ranges, or custom routing logic.
//
// The package also provides shard grouping, colocation checks, and bounded
// parallel operations across shards.
package shard
