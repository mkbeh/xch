# Changelog

All notable changes to this project will be documented in this file.

## v0.2.0

Initial release of the redesigned `xch` client.

### Added

* **Pool Management:** Named ClickHouse pools with labels, statistics, lifecycle management, connectivity checks, server
  version access, and an escape hatch to the underlying `clickhouse-go` client.
* **Query API:** `Exec`, `Query`, `QueryRow`, `Select`, and `PrepareBatch` built directly on native `clickhouse-go`
  types and semantics.
* **Batch Inserts:** Managed batch inserts with automatic send and cleanup through `InsertBatch` and `BatchWriter`.
* **Error Handling:** Helpers for missing rows, ClickHouse exceptions and exception codes, HTTP errors, and connection
  failures.
* **Shard Routing:** Immutable shard topologies with rendezvous, range, time-range, and custom routing strategies.
* **Multi-Key Routing:** Colocation checks, shard grouping, tolerant partitioning, and bounded fan-out operations.
* **Observability:** Optional OpenTelemetry client pool metrics through the independently versioned `extra/otelxch`
  module.
* **Examples:** Runnable examples for basic usage, OpenTelemetry metrics, range routing, geographic routing, and
  multi-key shard operations.

---

## extra/otelxch/v0.1.0

Initial release of the `otelxch` integration.

### Added

* **Pool Metrics:** OpenTelemetry gauges for open, idle, maximum open, and maximum idle client connections.
* **Metric Attributes:** Pool names and custom labels exported as metric attributes.
* **Meter Provider:** Support for application-managed OpenTelemetry meter providers and exporters.
* **Lifecycle:** Automatic metrics registration and cleanup with the associated `xch.Pool`.
