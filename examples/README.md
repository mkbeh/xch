# Examples

This directory contains runnable examples demonstrating the main features and usage patterns of `xch`.

| Example          | Demonstrates                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------- |
| [`basic`](basic) | Pool creation, connectivity checks, native batch inserts, single-row reads, and multi-row queries |

## Running the examples

The examples use Docker Compose to start ClickHouse and any required supporting services.

From the `examples` directory, start the topology required by the example:

```bash
docker compose up -d
```

> [!NOTE]
> Some examples may require additional services or configuration. Refer to the README in the corresponding example directory for the exact startup command and setup instructions.
