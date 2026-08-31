package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/mkbeh/xch"
	"github.com/mkbeh/xch/topology/shard"
	"github.com/mkbeh/xch/topology/shard/resolver"
)

const (
	shardARangeStart int64 = 0
	shardBoundary    int64 = 100
	shardBRangeEnd   int64 = 200

	truncateUsersSQL = "TRUNCATE TABLE xch_shard_group_example.users"
	insertUsersSQL   = "INSERT INTO xch_shard_group_example.users (id, name, active)"
	selectUsersSQL   = "SELECT id, name, active FROM xch_shard_group_example.users WHERE id IN (%s) ORDER BY id"
)

type user struct {
	ID     int64  `ch:"id"`
	Name   string `ch:"name"`
	Active bool   `ch:"active"`
}

var usersByID = map[int64]user{
	42: {
		ID:     42,
		Name:   "Alice",
		Active: true,
	},
	43: {
		ID:     43,
		Name:   "Carol",
		Active: true,
	},
	142: {
		ID:     142,
		Name:   "Bob",
		Active: true,
	},
	143: {
		ID:     143,
		Name:   "Dave",
		Active: true,
	},
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
		[]resolver.Range[int64]{
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

	if err := resetUsers(ctx, topology); err != nil {
		return err
	}

	if err := batchInsertUsers(ctx, userResolver, []int64{42, 142, 43, 143}); err != nil {
		return err
	}

	if err := showColocation(userResolver, 42, 43); err != nil {
		return err
	}

	if err := showShardMismatch(userResolver, 42, 142); err != nil {
		return err
	}

	if err := batchLoadUsers(ctx, userResolver, []int64{42, 142, 250, 43, 143}); err != nil {
		return err
	}

	return nil
}

func resetUsers(ctx context.Context, topology *shard.Topology) error {
	_, err := topology.ForEachShard(
		ctx,
		topology.Len(),
		func(ctx context.Context, current shard.Shard) error {
			return current.Pool().Exec(ctx, truncateUsersSQL)
		},
	)
	if err != nil {
		return fmt.Errorf("reset users: %w", err)
	}

	return nil
}

func batchInsertUsers(ctx context.Context, userResolver shard.Resolver[int64], ids []int64) error {
	groups, err := shard.GroupByShard(userResolver, ids)
	if err != nil {
		return fmt.Errorf("group users for batch insert: %w", err)
	}

	fmt.Println("batch insert:")

	for _, group := range groups {
		err := group.Shard.Pool().InsertBatch(
			ctx,
			insertUsersSQL,
			func(batch xch.BatchWriter) error {
				for _, id := range group.Keys {
					current := usersByID[id]

					if err := batch.Append(current.ID, current.Name, current.Active); err != nil {
						return err
					}
				}

				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("batch insert on shard %q: %w", group.Shard.ID(), err)
		}

		fmt.Printf(
			"- shard=%s user_ids=%v\n",
			group.Shard.ID(),
			group.Keys,
		)
	}

	return nil
}

func showColocation(userResolver shard.Resolver[int64], ids ...int64) error {
	target, err := shard.SameShard(userResolver, ids...)
	if err != nil {
		return fmt.Errorf("resolve colocated users: %w", err)
	}

	fmt.Println()
	fmt.Println("colocation:")
	fmt.Printf("- user_ids=%v shard=%s\n", ids, target.ID())

	return nil
}

func showShardMismatch(userResolver shard.Resolver[int64], ids ...int64) error {
	_, err := shard.SameShard(userResolver, ids...)
	if !errors.Is(err, shard.ErrShardMismatch) {
		return fmt.Errorf("check shard mismatch: got %v, want shard.ErrShardMismatch", err)
	}

	var mismatch *shard.MismatchError
	if !errors.As(err, &mismatch) {
		return fmt.Errorf("check shard mismatch details: %w", err)
	}

	fmt.Printf(
		"- user_ids=%v mismatch=%s->%s\n",
		ids,
		mismatch.Expected,
		mismatch.Actual,
	)

	return nil
}

func batchLoadUsers(ctx context.Context, userResolver shard.Resolver[int64], ids []int64) error {
	partition, err := shard.PartitionByShard(userResolver, ids)
	if err != nil {
		return fmt.Errorf("partition users for batch select: %w", err)
	}

	fmt.Println()
	fmt.Println("batch select:")

	if len(partition.Unresolved) != 0 {
		fmt.Printf(
			"- unresolved user_ids=%v\n",
			partition.Unresolved,
		)
	}

	for _, group := range partition.Groups {
		users, err := loadUsers(ctx, group.Shard.Pool(), group.Keys)
		if err != nil {
			return fmt.Errorf("batch select on shard %q: %w", group.Shard.ID(), err)
		}

		fmt.Printf(
			"- shard=%s user_ids=%v\n",
			group.Shard.ID(),
			group.Keys,
		)

		for _, current := range users {
			fmt.Printf(
				"  user_id=%d name=%s active=%t\n",
				current.ID,
				current.Name,
				current.Active,
			)
		}
	}

	return nil
}

func loadUsers(ctx context.Context, pool *xch.Pool, ids []int64) ([]user, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(
		strings.Repeat("?, ", len(ids)),
		", ",
	)

	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}

	var users []user

	if err := pool.Select(
		ctx,
		&users,
		fmt.Sprintf(selectUsersSQL, placeholders),
		args...,
	); err != nil {
		return nil, err
	}

	return users, nil
}
