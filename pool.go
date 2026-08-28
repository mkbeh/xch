package xch

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Pool is a stable facade over a clickhouse-go native connection pool.
//
// The zero value is not usable; create a Pool with [Open] or [New]. Pool is safe
// for concurrent use whenever the underlying clickhouse-go Conn is safe for the
// corresponding operation.
type Pool struct {
	conn driver.Conn

	name   string
	labels map[string]string
}

// Open parses a clickhouse-go DSN and creates a Pool.
//
// Open does not ping ClickHouse or otherwise perform network I/O. The first
// operation, or an explicit call to [Pool.Ping], establishes connectivity.
func Open(dsn string, options ...Option) (*Pool, error) {
	config, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("xch: parse DSN: %w", err)
	}

	return New(config, options...)
}

// New creates a Pool from clickhouse-go options.
//
// New defensively copies the supplied options before passing them to
// clickhouse-go. The TLS configuration is copied with tls.Config.Clone.
// Function values, loggers, transports, values stored inside Settings, and
// objects referenced by the TLS clone remain shared dependencies and must be
// safe for concurrent use.
func New(config *clickhouse.Options, options ...Option) (*Pool, error) {
	if config == nil {
		return nil, errors.New("xch: config must not be nil")
	}

	settings := defaultSettings()

	if err := applyOptions(settings, options...); err != nil {
		return nil, err
	}

	conn, err := clickhouse.Open(config)
	if err != nil {
		return nil, fmt.Errorf("xch: open connection: %w", err)
	}

	return &Pool{
		conn:   conn,
		name:   settings.name,
		labels: settings.labels,
	}, nil
}

// Name returns the logical pool name configured with [WithName].
func (p *Pool) Name() string {
	return p.name
}

// Labels returns a detached copy of the pool labels.
func (p *Pool) Labels() map[string]string {
	return maps.Clone(p.labels)
}

// Ping verifies that ClickHouse is reachable.
func (p *Pool) Ping(ctx context.Context) error {
	return p.conn.Ping(ctx)
}

// ServerVersion returns the server version reported by clickhouse-go.
func (p *Pool) ServerVersion() (*driver.ServerVersion, error) {
	return p.conn.ServerVersion()
}

// Exec executes a query without returning rows.
func (p *Pool) Exec(ctx context.Context, query string, args ...any) error {
	return p.conn.Exec(ctx, query, args...)
}

// Query executes a query and returns its rows.
//
// Callers must close the returned rows.
func (p *Pool) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return p.conn.Query(ctx, query, args...)
}

// QueryRow executes a query expected to return at most one row.
//
// Callers must consume the returned row with Scan or ScanStruct.
func (p *Pool) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return p.conn.QueryRow(ctx, query, args...)
}

// Select executes a query and scans all returned rows into dest.
func (p *Pool) Select(ctx context.Context, dest any, query string, args ...any) error {
	return p.conn.Select(ctx, dest, query, args...)
}

// PrepareBatch prepares a ClickHouse insert batch.
//
// Callers should defer Close immediately after successful preparation and call
// Send to finalize the insert. Close releases resources but does not guarantee
// that buffered rows are sent.
func (p *Pool) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return p.conn.PrepareBatch(ctx, query, opts...)
}

// Raw returns the underlying clickhouse-go connection.
//
// The returned connection is borrowed. Callers must not close it directly;
// ownership remains with Pool.
func (p *Pool) Raw() driver.Conn {
	return p.conn
}

// Close closes the underlying clickhouse-go connection.
func (p *Pool) Close() error {
	return p.conn.Close()
}

// ClientInfo creates ClickHouse client information for an application or
// library identified by name and version.
func ClientInfo(name, version string) clickhouse.ClientInfo {
	return clickhouse.ClientInfo{
		Products: []struct {
			Name    string
			Version string
		}{
			{
				Name:    name,
				Version: version,
			},
		},
	}
}
