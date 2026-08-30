package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mkbeh/xch"
)

const defaultDSN = "clickhouse://clickhouse:clickhouse@localhost:9000/default?dial_timeout=5s"

type user struct {
	ID     uint64 `ch:"id"`
	Name   string `ch:"name"`
	Email  string `ch:"email"`
	Active bool   `ch:"active"`
}

func main() {
	if err := run(context.Background()); err != nil {
		if code, ok := xch.ExceptionCode(err); ok {
			log.Fatalf("ClickHouse exception %d: %v", code, err)
		}
		if xch.IsConnectionError(err) {
			log.Fatalf("ClickHouse connection error: %v", err)
		}

		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	pool, err := xch.Open(
		databaseDSN(),
		xch.WithName("basic-example"),
	)
	if err != nil {
		return fmt.Errorf("open pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping ClickHouse: %w", err)
	}

	if err := insertUsers(ctx, pool); err != nil {
		return fmt.Errorf("insert users: %w", err)
	}

	selected, err := loadUser(ctx, pool, 1)
	if err != nil {
		return fmt.Errorf("load user: %w", err)
	}

	if _, err := loadUser(ctx, pool, 999); err != nil {
		if !xch.IsNoRows(err) {
			return fmt.Errorf("load missing user: %w", err)
		}

		fmt.Println("missing user: not found")
	}

	active, err := listActiveUsers(ctx, pool)
	if err != nil {
		return fmt.Errorf("list active users: %w", err)
	}

	fmt.Printf("pool: %s\n", pool.Name())
	fmt.Printf(
		"selected user: %d %s <%s> active=%t\n",
		selected.ID,
		selected.Name,
		selected.Email,
		selected.Active,
	)

	fmt.Println("active users:")
	for _, current := range active {
		fmt.Printf("- %d %s <%s>\n", current.ID, current.Name, current.Email)
	}

	return nil
}

func insertUsers(ctx context.Context, pool *xch.Pool) error {
	if err := pool.Exec(ctx, "TRUNCATE TABLE xch_basic_example.users"); err != nil {
		return err
	}

	users := []user{
		{
			ID:     1,
			Name:   "Alice",
			Email:  "alice@example.com",
			Active: true,
		},
		{
			ID:     2,
			Name:   "Bob",
			Email:  "bob@example.com",
			Active: false,
		},
	}

	return pool.InsertBatch(
		ctx,
		"INSERT INTO xch_basic_example.users (id, name, email, active)",
		func(batch xch.BatchWriter) error {
			for _, current := range users {
				if err := batch.Append(
					current.ID,
					current.Name,
					current.Email,
					current.Active,
				); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func loadUser(ctx context.Context, pool *xch.Pool, userID uint64) (user, error) {
	var selected user

	err := pool.QueryRow(
		ctx,
		`SELECT id, name, email, active
		FROM xch_basic_example.users
		WHERE id = ?`,
		userID,
	).ScanStruct(&selected)
	if err != nil {
		return user{}, err
	}

	return selected, nil
}

func listActiveUsers(ctx context.Context, pool *xch.Pool) ([]user, error) {
	var users []user

	if err := pool.Select(
		ctx,
		&users,
		`SELECT id, name, email, active
		FROM xch_basic_example.users
		WHERE active
		ORDER BY id`,
	); err != nil {
		return nil, err
	}

	return users, nil
}

func databaseDSN() string {
	if value := os.Getenv("XCH_DATABASE_URL"); value != "" {
		return value
	}

	return defaultDSN
}
