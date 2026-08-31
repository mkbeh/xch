CREATE DATABASE IF NOT EXISTS xch_basic_example;

CREATE TABLE IF NOT EXISTS xch_basic_example.users
(
    id UInt64,
    name String,
    email String,
    active Bool
)
ENGINE = MergeTree
ORDER BY id;
