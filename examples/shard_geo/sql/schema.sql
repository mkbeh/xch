CREATE DATABASE IF NOT EXISTS xch_shard_geo_example;

CREATE TABLE IF NOT EXISTS xch_shard_geo_example.tenants
(
    id String,
    name String,
    region LowCardinality(String),
    last_seen_at DateTime64(3)
)
ENGINE = ReplacingMergeTree(last_seen_at)
ORDER BY id;
