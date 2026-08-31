package main

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/mkbeh/xch"
	"github.com/mkbeh/xch/topology/shard"
)

const (
	defaultShardADSN = "clickhouse://clickhouse:clickhouse@localhost:57431/default?dial_timeout=5s"
	defaultShardBDSN = "clickhouse://clickhouse:clickhouse@localhost:57432/default?dial_timeout=5s"

	shardAID shard.ID = "shard-eu"
	shardBID shard.ID = "shard-us"
)

func openTopology(ctx context.Context) (*shard.Topology, error) {
	configs := [...]struct {
		id     shard.ID
		env    string
		dsn    string
		labels map[string]string
	}{
		{
			id:     shardAID,
			env:    "XCH_SHARD_EU_DATABASE_URL",
			dsn:    defaultShardADSN,
			labels: map[string]string{"region": "eu"},
		},
		{
			id:     shardBID,
			env:    "XCH_SHARD_US_DATABASE_URL",
			dsn:    defaultShardBDSN,
			labels: map[string]string{"region": "us"},
		},
	}

	pools := make([]*xch.Pool, 0, len(configs))

	ownsPools := true
	defer func() {
		if ownsPools {
			closePools(pools)
		}
	}()

	for _, config := range configs {
		pool, err := openShard(
			ctx,
			config.id,
			environment(config.env, config.dsn),
			config.labels,
		)
		if err != nil {
			return nil, fmt.Errorf("open shard %s: %w", config.id, err)
		}

		pools = append(pools, pool)
	}

	topology, err := shard.NewTopology(pools...)
	if err != nil {
		return nil, fmt.Errorf("create topology: %w", err)
	}

	ownsPools = false

	return topology, nil
}

func openShard(
	ctx context.Context,
	id shard.ID,
	dsn string,
	labels map[string]string,
) (*xch.Pool, error) {
	options := []xch.Option{
		xch.WithName(string(id)),
	}

	for key, value := range labels {
		options = append(
			options,
			xch.WithLabel(key, value),
		)
	}

	pool, err := xch.Open(dsn, options...)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		_ = pool.Close()

		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return pool, nil
}

func closePools(pools []*xch.Pool) {
	for _, pool := range slices.Backward(pools) {
		_ = pool.Close()
	}
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
