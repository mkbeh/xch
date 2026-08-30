# Examples

This directory contains runnable examples demonstrating the main features and usage patterns of `xch`.

| Example                          | Demonstrates                                                                                      |
|----------------------------------|---------------------------------------------------------------------------------------------------|
| [`basic`](basic)                 | Pool creation, connectivity checks, managed batch inserts, struct scanning, and error helpers     |
| [`observability`](observability) | Structured driver logging and OpenTelemetry client connection-pool metrics exported to Prometheus |

## Running the examples

The examples use Docker Compose to start ClickHouse and any required supporting services.

From the `examples` directory, start the topology required by the example:

```bash
docker compose up -d
```
