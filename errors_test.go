package xch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNoRows(t *testing.T) {
	t.Parallel()

	assert.True(t, IsNoRows(sql.ErrNoRows))
	assert.True(t, IsNoRows(fmt.Errorf("query user: %w", sql.ErrNoRows)))
	assert.False(t, IsNoRows(errors.New("other error")))
	assert.False(t, IsNoRows(nil))
}

func TestExceptionHelpers(t *testing.T) {
	t.Parallel()

	exception := &clickhouse.Exception{
		Code:    60,
		Name:    "DB::Exception",
		Message: "unknown table",
	}
	err := fmt.Errorf("query failed: %w", exception)

	got, ok := Exception(err)
	require.True(t, ok)
	assert.Same(t, exception, got)

	code, ok := ExceptionCode(err)
	require.True(t, ok)
	assert.Equal(t, int32(60), code)

	assert.True(t, IsExceptionCode(err, 47, 60, 241))
	assert.False(t, IsExceptionCode(err, 47, 241))
	assert.False(t, IsExceptionCode(err))

	got, ok = Exception(errors.New("other error"))
	assert.False(t, ok)
	assert.Nil(t, got)

	code, ok = ExceptionCode(errors.New("other error"))
	assert.False(t, ok)
	assert.Zero(t, code)
}

func TestHTTPError(t *testing.T) {
	t.Parallel()

	exception := &clickhouse.Exception{
		Code:    60,
		Name:    "DB::Exception",
		Message: "unknown table",
	}
	httpErr := &clickhouse.HTTPError{
		StatusCode: 404,
		Err:        exception,
	}
	err := fmt.Errorf("request failed: %w", httpErr)

	got, ok := HTTPError(err)
	require.True(t, ok)
	assert.Same(t, httpErr, got)

	gotException, ok := Exception(err)
	require.True(t, ok)
	assert.Same(t, exception, gotException)

	assert.True(t, IsExceptionCode(err, 60))

	got, ok = HTTPError(errors.New("other error"))
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestIsConnectionError(t *testing.T) {
	t.Parallel()

	exception := &clickhouse.Exception{
		Code:    60,
		Name:    "DB::Exception",
		Message: "unknown table",
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "context canceled",
			err:  fmt.Errorf("query: %w", context.Canceled),
			want: false,
		},
		{
			name: "context deadline exceeded",
			err:  fmt.Errorf("query: %w", context.DeadlineExceeded),
			want: false,
		},
		{
			name: "connection closed",
			err:  fmt.Errorf("query: %w", clickhouse.ErrConnectionClosed),
			want: true,
		},
		{
			name: "EOF",
			err:  fmt.Errorf("read: %w", io.EOF),
			want: true,
		},
		{
			name: "unexpected EOF",
			err:  fmt.Errorf("read: %w", io.ErrUnexpectedEOF),
			want: true,
		},
		{
			name: "net closed",
			err:  fmt.Errorf("read: %w", net.ErrClosed),
			want: true,
		},
		{
			name: "network deadline exceeded",
			err:  fmt.Errorf("read: %w", os.ErrDeadlineExceeded),
			want: true,
		},
		{
			name: "net error",
			err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: errors.New("network failure"),
			},
			want: true,
		},
		{
			name: "server exception",
			err:  exception,
			want: false,
		},
		{
			name: "HTTP error",
			err: &clickhouse.HTTPError{
				StatusCode: 500,
				Err:        errors.New("server failure"),
			},
			want: false,
		},
		{
			name: "pool acquire timeout",
			err:  clickhouse.ErrAcquireConnTimeout,
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("other error"),
			want: false,
		},
		{
			name: "context cancellation takes precedence",
			err:  errors.Join(context.Canceled, io.EOF),
			want: false,
		},
		{
			name: "context deadline takes precedence",
			err:  errors.Join(context.DeadlineExceeded, io.EOF),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.want, IsConnectionError(test.err))
		})
	}
}
