// Package xch provides a small production-oriented facade over the native
// clickhouse-go client.
//
// Constructors do not ping ClickHouse or otherwise perform network I/O.
// Applications should call [Pool.Ping] explicitly when startup readiness must
// depend on ClickHouse availability.
package xch
