package xch

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"os"
	"slices"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// IsNoRows reports whether err indicates that a query returned no rows.
func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// Exception returns the first ClickHouse server exception in err's tree.
//
// Native and HTTP protocol errors are handled uniformly. HTTP errors may carry
// both a [clickhouse.HTTPError] and an embedded [clickhouse.Exception].
func Exception(err error) (*clickhouse.Exception, bool) {
	var exception *clickhouse.Exception
	if !errors.As(err, &exception) {
		return nil, false
	}

	return exception, true
}

// ExceptionCode returns the ClickHouse server exception code carried by err.
func ExceptionCode(err error) (int32, bool) {
	exception, ok := Exception(err)
	if !ok {
		return 0, false
	}

	return exception.Code, true
}

// IsExceptionCode reports whether err carries one of the supplied ClickHouse
// server exception codes.
func IsExceptionCode(err error, codes ...int32) bool {
	code, ok := ExceptionCode(err)
	if !ok {
		return false
	}

	return slices.Contains(codes, code)
}

// HTTPError returns the first HTTP protocol error in err's tree.
func HTTPError(err error) (*clickhouse.HTTPError, bool) {
	var httpError *clickhouse.HTTPError
	if !errors.As(err, &httpError) {
		return nil, false
	}

	return httpError, true
}

// IsConnectionError reports whether err represents a client-side connection
// failure known to clickhouse-go or the Go networking stack.
//
// Context cancellation and context deadline expiry are deliberately not
// classified as connection failures. ClickHouse server exceptions and non-200
// HTTP responses are not connection failures solely because they were
// delivered over the network.
//
// This helper does not determine whether an operation is safe to retry.
func IsConnectionError(err error) bool {
	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, clickhouse.ErrConnectionClosed) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var networkError net.Error

	return errors.As(err, &networkError)
}
