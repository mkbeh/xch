package xch

// PoolStats is a detached point-in-time snapshot of connection pool statistics.
type PoolStats struct {
	// Current state.

	// OpenConns is the number of currently open connections.
	OpenConns int

	// IdleConns is the number of currently idle connections.
	IdleConns int

	// MaxOpenConns is the maximum number of open connections allowed by the
	// pool.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections retained by the
	// pool.
	MaxIdleConns int
}

// Stats returns a detached snapshot of the current pool statistics.
func (p *Pool) Stats() PoolStats {
	stats := p.conn.Stats()

	return PoolStats{
		OpenConns:    stats.Open,
		IdleConns:    stats.Idle,
		MaxOpenConns: stats.MaxOpenConns,
		MaxIdleConns: stats.MaxIdleConns,
	}
}
