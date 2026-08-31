package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/mkbeh/xch/topology/shard"
)

const (
	upsertTenantSQL = `
		INSERT INTO xch_shard_geo_example.tenants (
			id,
			name,
			region,
			last_seen_at
		)
		VALUES (?, ?, ?, now64(3))
	`

	loadTenantSQL = `
		SELECT
			id,
			name,
			region
		FROM xch_shard_geo_example.tenants
		WHERE id = ?
		ORDER BY last_seen_at DESC
		LIMIT 1
	`
)

type tenantKey struct {
	TenantID string
	Region   string
}

type tenant struct {
	ID     string `ch:"id"`
	Name   string `ch:"name"`
	Region string `ch:"region"`
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

	tenantResolver, err := newTenantResolver(topology)
	if err != nil {
		return fmt.Errorf("create tenant resolver: %w", err)
	}

	tenants := []tenant{
		{ID: "tenant-42", Name: "Alice", Region: "eu"},
		{ID: "tenant-77", Name: "Bob", Region: "us"},
	}

	fmt.Println("geo routing:")

	for _, current := range tenants {
		if err := processTenant(ctx, tenantResolver, current); err != nil {
			return err
		}
	}

	_, err = tenantResolver.Resolve(
		tenantKey{
			TenantID: "tenant-99",
			Region:   "apac",
		},
	)
	if !errors.Is(err, shard.ErrNoShard) {
		return fmt.Errorf("resolve unsupported region: got %v, want shard.ErrNoShard", err)
	}

	fmt.Println("unsupported region: no shard")

	return nil
}

func processTenant(
	ctx context.Context,
	tenantResolver shard.Resolver[tenantKey],
	current tenant,
) error {
	key := tenantKey{
		TenantID: current.ID,
		Region:   current.Region,
	}

	target, err := tenantResolver.Resolve(key)
	if err != nil {
		return fmt.Errorf("resolve tenant %q: %w", current.ID, err)
	}

	if err := upsertTenant(ctx, target, current); err != nil {
		return fmt.Errorf("upsert tenant %q on shard %s: %w", current.ID, target.ID(), err)
	}

	stored, err := loadTenant(ctx, target, current.ID)
	if err != nil {
		return fmt.Errorf("load tenant %q from shard %s: %w", current.ID, target.ID(), err)
	}

	fmt.Printf(
		"- tenant=%s region=%s shard=%s name=%s\n",
		stored.ID,
		stored.Region,
		target.ID(),
		stored.Name,
	)

	return nil
}

func upsertTenant(ctx context.Context, target shard.Shard, current tenant) error {
	return target.Pool().Exec(
		ctx,
		upsertTenantSQL,
		current.ID,
		current.Name,
		current.Region,
	)
}

func loadTenant(ctx context.Context, target shard.Shard, tenantID string) (tenant, error) {
	var stored tenant

	err := target.Pool().
		QueryRow(ctx, loadTenantSQL, tenantID).
		ScanStruct(&stored)
	if err != nil {
		return tenant{}, err
	}

	return stored, nil
}
