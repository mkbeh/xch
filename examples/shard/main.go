package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/mkbeh/xch/topology/shard"
	"github.com/mkbeh/xch/topology/shard/resolver"
)

const (
	shardARangeStart uint64 = 0
	shardBoundary    uint64 = 100
	shardBRangeEnd   uint64 = 200

	insertUserSQL = "INSERT INTO xch_shard_example.users (id, name) VALUES (?, ?)"
)

type user struct {
	ID   uint64
	Name string
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) (err error) {
	topology, err := openTopology(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := topology.Close(); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf("close topology: %w", closeErr),
			)
		}
	}()

	userResolver, err := resolver.NewRange(
		topology,
		[]resolver.Range[uint64]{
			{
				Start:   shardARangeStart,
				End:     shardBoundary,
				ShardID: shardAID,
			},
			{
				Start:   shardBoundary,
				End:     shardBRangeEnd,
				ShardID: shardBID,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("create user resolver: %w", err)
	}

	users := []user{
		{ID: 42, Name: "Alice"},
		{ID: 142, Name: "Bob"},
	}

	fmt.Println("range routing:")

	for _, current := range users {
		if err := insertUser(ctx, userResolver, current); err != nil {
			return err
		}
	}

	return nil
}

func insertUser(
	ctx context.Context,
	userResolver shard.Resolver[uint64],
	current user,
) error {
	target, err := userResolver.Resolve(current.ID)
	if err != nil {
		return fmt.Errorf("resolve user %d: %w", current.ID, err)
	}

	if err := target.Pool().Exec(ctx, insertUserSQL, current.ID, current.Name); err != nil {
		return fmt.Errorf("write user %d to shard %s: %w", current.ID, target.ID(), err)
	}

	fmt.Printf(
		"- user_id=%d shard=%s pool=%s\n",
		current.ID,
		target.ID(),
		target.Pool().Name(),
	)

	return nil
}
