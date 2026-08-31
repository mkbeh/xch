CREATE DATABASE IF NOT EXISTS xch_shard_group_example;

CREATE TABLE IF NOT EXISTS xch_shard_group_example.users
(
    id Int64,
    name String,
    active Bool
)
ENGINE = MergeTree
ORDER BY id;
