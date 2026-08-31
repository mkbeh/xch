<div align="center">

# ClickHouse toolkit for Go

**Lightweight ClickHouse wrapper for Go, built on top of [clickhouse-go](https://github.com/ClickHouse/clickhouse-go).**

[![Go Reference](https://pkg.go.dev/badge/github.com/mkbeh/xch.svg)](https://pkg.go.dev/github.com/mkbeh/xch)
[![Test](https://github.com/mkbeh/xch/actions/workflows/test.yml/badge.svg)](https://github.com/mkbeh/xch/actions/workflows/test.yml)
[![Coverage](https://codecov.io/gh/mkbeh/xch/graph/badge.svg)](https://codecov.io/gh/mkbeh/xch)

</div>

`xch` is a lightweight wrapper around `clickhouse-go` that keeps its native types and query model while adding pool
metadata and lifecycle management, managed batch inserts, error handling, shard routing, and optional OpenTelemetry
metrics.

## Features

* **Pool Management:** Named connection pools with labels, statistics, lifecycle management, and access to the
  underlying `clickhouse-go` client.
* **Native Query API:** Execute and scan queries using native `clickhouse-go` types.
* **Managed Batch Inserts:** Scoped batch inserts with automatic send and cleanup.
* **Error Handling:** Helpers for missing rows, ClickHouse exceptions, HTTP errors, and connection failures.
* **Shard Routing:** Rendezvous, range, time-range, and custom routing with colocation checks, key grouping,
  partitioning, and bounded fan-out operations.
* **Observability:** Client pool statistics with optional OpenTelemetry metrics integration.

## Installation

This repository contains the core `xch` module. The core module is released from the repository root:

```bash
go get github.com/mkbeh/xch
```

Optional integrations are released independently under `extra`:

```bash
go get github.com/mkbeh/xch/extra/otelxch
```

## Usage

Open a pool from a ClickHouse DSN and use the native `clickhouse-go` query model:

<!-- @formatter:off -->
```go
pool, err := xch.Open(
    os.Getenv("XCH_DATABASE_URL"),
    xch.WithName("example-pool"),
)
if err != nil {
    return fmt.Errorf("open pool: %w", err)
}
defer pool.Close()

var message string
if err := pool.QueryRow(ctx, "SELECT 'hello from xch'").Scan(&message); err != nil {
    return fmt.Errorf("query: %w", err)
}

fmt.Println(message) // hello from xch
```
<!-- @formatter:on -->

For full `clickhouse-go` configuration, create the pool from `clickhouse.Options`:

<!-- @formatter:off -->
```go
pool, err := xch.New(
    &clickhouse.Options{
        Addr:             []string{"clickhouse-1:9000", "clickhouse-2:9000"},
        ConnOpenStrategy: clickhouse.ConnOpenRoundRobin,
    },
    xch.WithName("analytics"),
)
```
<!-- @formatter:on -->

A pool may contain multiple replicas of the same logical ClickHouse shard.

### Batch Inserts

`InsertBatch` manages the batch lifecycle and sends the batch when the callback completes successfully.

<!-- @formatter:off -->
```go
err := pool.InsertBatch(
    ctx,
    "INSERT INTO users (id, name, active)",
    func(batch xch.BatchWriter) error {
        for _, user := range users {
            if err := batch.Append(user.ID, user.Name, user.Active); err != nil {
                return err
            }
        }

        return nil
    },
)
```
<!-- @formatter:on -->

If the callback returns an error, the batch is closed without being sent. Use `PrepareBatch` directly when full control
over the native `clickhouse-go` batch lifecycle is required.

### Error Handling

`xch` provides helpers for classifying common ClickHouse and transport errors while preserving the original error chain.

<!-- @formatter:off -->
```go
var name string
err := pool.QueryRow(ctx, "SELECT name FROM users WHERE id = ?", userID).Scan(&name)

switch {
case xch.IsNoRows(err):
    // Handle a missing row.

case xch.IsConnectionError(err):
    // Handle a connection failure.

case err != nil:
    if code, ok := xch.ExceptionCode(err); ok {
        log.Printf("ClickHouse exception %d", code)
    }

    return err
}
```
<!-- @formatter:on -->

`Exception` and `ExceptionCode` expose native ClickHouse exceptions, while `HTTPError` extracts errors returned by the
HTTP transport.

## Sharding

The `topology/shard` package provides application-level routing across an immutable set of logical ClickHouse shards.
Routing strategies live under `topology/shard/resolver`.

Each shard is backed by a single `xch.Pool`, which may contain multiple replicas of that shard.

<!-- @formatter:off -->

```go
shardA, err := xch.Open(
    shardADSN,
    xch.WithName("shard-a"),
)
if err != nil {
	panic(err)
}

shardB, err := xch.Open(
    shardBDSN,
    xch.WithName("shard-b"),
)
if err != nil {
	panic(err)
}

topology, err := shard.NewTopology(shardA, shardB)
if err != nil {
	panic(err)
}

defer topology.Close()

userResolver, _ := resolver.NewRange(
    topology,
    []resolver.Range[uint64]{
        {Start: 0, End: 100, ShardID: "shard-a"},
        {Start: 100, End: 200, ShardID: "shard-b"},
    },
)

// Resolve the physical shard for the user.
target, err := userResolver.Resolve(userID)
if err != nil {
	panic(err)
}

// Execute the operation directly on the resolved shard.
err = target.Pool().Exec(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", userID, name)
if err != nil {
	panic(err)
}
```

<!-- @formatter:on -->

After successful construction, the topology owns its pools and closes them when `Topology.Close` is called.

### Routing Strategies

Resolvers bind a placement strategy to an immutable shard topology. Every resolver implements the same `Resolve`
contract.

| Resolver             | Best Suited For                       | Routing Model                                                         |
| :------------------- | :------------------------------------ | :-------------------------------------------------------------------- |
| `RendezvousResolver` | Keys without natural ranges           | Deterministic Highest Random Weight (HRW) hashing within a namespace. |
| `RangeResolver`      | Ordered numeric or string keys        | Bounded, non-overlapping half-open intervals `[Start, End)`.          |
| `TimeRangeResolver`  | Time-series or partitioned event data | Bounded chronological intervals normalized to UTC.                    |
| `CustomResolver`     | Domain-specific placement rules       | Application-defined mapping from a key to `shard.ID`.                 |

<!-- @formatter:off -->

```go
// Rendezvous hashing distributes arbitrary keys deterministically.
usersByHash, _ := resolver.NewRendezvous(topology, "users", resolver.Uint64KeyEncoder())

// Ordered ranges provide explicit control over the keyspace.
usersByRange, _ := resolver.NewRange(
    topology,
    []resolver.Range[uint64]{
        {Start: 0, End: 100, ShardID: "shard-a"},
        {Start: 100, End: 200, ShardID: "shard-b"},
    },
)

// Time ranges route records through bounded chronological intervals.
eventsByTime, _ := resolver.NewTimeRange(
    topology,
    []resolver.TimeRange{
        {Start: start2025, End: start2026, ShardID: "shard-a"},
        {Start: start2026, End: start2027, ShardID: "shard-b"},
    },
)

// Custom routing keeps domain-specific placement rules in application code.
tenantsByRegion, _ := resolver.NewCustom(
    topology,
    func(region string) (shard.ID, error) {
        switch region {
        case "eu":
            return "shard-a", nil
        case "us":
            return "shard-b", nil
        default:
            return "", shard.ErrNoShard
        }
    },
)
```

<!-- @formatter:on -->

Regardless of the selected strategy, routing uses the same contract:

<!-- @formatter:off -->

```go
target, _ := usersByHash.Resolve(userID)
log.Printf("resolved shard: %s", target.ID())
```

<!-- @formatter:on -->

Range and time-range resolvers may contain intentional gaps in the configured keyspace. Keys that do not match any
configured range return `ErrNoShard`.

> [!IMPORTANT]
> For rendezvous routing, the namespace, key encoding, and stable shard IDs are part of the persistent placement
contract.

### Multi-Key Routing

For batch workloads, `xch` provides routing primitives for analyzing, grouping, and partitioning keys across the
topology.

**Strict Colocation**

`SameShard` verifies that all keys resolve to the same shard before an operation that must remain colocated.

<!-- @formatter:off -->

```go
target, err := shard.SameShard(userResolver, 42, 43)
if err != nil {
	panic(err)
}

log.Printf("resolved shard: %s", target.ID())
```

<!-- @formatter:on -->

**Strict Grouping**

`GroupByShard` groups keys by destination shard and fails if any key cannot be resolved.

<!-- @formatter:off -->

```go
keys := []uint64{42, 142, 43, 143}

groups, err := shard.GroupByShard(userResolver, keys)
if err != nil {
	panic(err)
}

for _, group := range groups {
    log.Printf("process shard=%s user_ids=%v", group.Shard.ID(), group.Keys)
}
```

<!-- @formatter:on -->

**Tolerant Partitioning**

`PartitionByShard` groups routable keys by shard while collecting keys that do not resolve to any shard separately.

<!-- @formatter:off -->

```go
keys := []uint64{42, 142, 250, 43, 143}

partition, err := shard.PartitionByShard(userResolver, keys)
if err != nil {
    panic(err)
}

for _, group := range partition.Groups {
    log.Printf("process shard=%s user_ids=%v", group.Shard.ID(), group.Keys)
}

if len(partition.Unresolved) != 0 {
    log.Printf("unresolved keys: %v", partition.Unresolved)
}
```

<!-- @formatter:on -->

### Fan-Out Operations

`ForEachShard` executes an operation across the entire topology with bounded concurrency. Setting the concurrency to `1`
makes execution sequential.

<!-- @formatter:off -->

```go
const maxConcurrency = 4

results, err := topology.ForEachShard(
    ctx,
    maxConcurrency,
    func(ctx context.Context, target shard.Shard) error {
        return target.Pool().Exec(ctx, "OPTIMIZE TABLE events FINAL")
    },
)
if err != nil {
    log.Printf("fan-out completed with errors: %v", err)
}

// Inspect individual shard failures when detailed handling is required.
for _, result := range results {
    if result.Err != nil {
        log.Printf("shard=%s failed: %v", result.ShardID, result.Err)
    }
}
```

<!-- @formatter:on -->

Results preserve topology registration order and retain individual shard failures, while the returned error aggregates
callback and context cancellation errors.

## Examples

See the [examples](examples) directory for runnable examples covering the main `xch` usage patterns.

## License

This project is licensed under the [MIT License](LICENSE).
