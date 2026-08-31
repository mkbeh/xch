# Examples

This directory contains runnable examples demonstrating the main features and usage patterns of `xch`.

| Example                          | Demonstrates                                                                                  |
|----------------------------------|-----------------------------------------------------------------------------------------------|
| [`basic`](basic)                 | Pool creation, connectivity checks, managed batch inserts, struct scanning, and error helpers |
| [`observability`](observability) | OpenTelemetry client connection-pool metrics                                                  |
| [`shard`](shard)                 | Direct routing across ClickHouse shards                                                       |
| [`shard_group`](shard_group)     | Key grouping, colocation checks, partitioning, and fan-out operations                         |
| [`shard_geo`](shard_geo)         | Region-based routing using shard metadata and a custom resolver                               |

## Running the examples

The examples use Docker Compose where local ClickHouse instances are required.

Refer to each example README for setup, configuration, expected output, and cleanup instructions.
