# Shard grouping

This example demonstrates multi-key routing across ClickHouse shards:

* Group keys by shard
* Verify key colocation
* Detect cross-shard mismatches
* Separate routed and unroutable keys
* Execute one batch operation per shard

The topology contains two shards:

```text
[0, 100)   -> shard-a
[100, 200) -> shard-b
```

## Local setup

From this directory, start both ClickHouse shards:

```shell
docker compose up -d
```

Apply the schema to both shards:

```shell
docker compose exec -T clickhouse-shard-a \
  clickhouse-client --user clickhouse --password clickhouse --multiquery \
  < sql/schema.sql

docker compose exec -T clickhouse-shard-b \
  clickhouse-client --user clickhouse --password clickhouse --multiquery \
  < sql/schema.sql
```

The shards are available at:

```text
Shard A: localhost:58431
Shard B: localhost:58432
```

## Configuration

Set `XCH_SHARD_A_DATABASE_URL` or `XCH_SHARD_B_DATABASE_URL` to use different endpoints.

## Run

From this directory:

```shell
go run .
```

Or from the repository root:

```shell
go run ./examples/shard_group
```

## Expected output

```text
batch insert:
- shard=shard-a user_ids=[42 43]
- shard=shard-b user_ids=[142 143]

colocation:
- user_ids=[42 43] shard=shard-a
- user_ids=[42 142] mismatch=shard-a->shard-b

batch select:
- unresolved user_ids=[250]
- shard=shard-a user_ids=[42 43]
  user_id=42 name=Alice active=true
  user_id=43 name=Carol active=true
- shard=shard-b user_ids=[142 143]
  user_id=142 name=Bob active=true
  user_id=143 name=Dave active=true
```

## Cleanup

Stop ClickHouse:

```shell
docker compose down
```

To also remove both data volumes:

```shell
docker compose down -v
```
