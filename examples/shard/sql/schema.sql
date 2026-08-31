CREATE DATABASE IF NOT EXISTS xch_shard_example;

CREATE TABLE IF NOT EXISTS xch_shard_example.users
(
    id UInt64,
    name String
)
ENGINE = MergeTree
ORDER BY id;
