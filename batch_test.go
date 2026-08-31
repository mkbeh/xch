package xch

import (
	"context"
	"errors"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertBatch(t *testing.T) {
	t.Parallel()

	column := &batchTestColumn{}
	batch := &batchTestBatch{
		column: column,
	}

	var (
		gotQuery   string
		gotOptsLen int
	)

	pool := &Pool{
		conn: &batchTestConn{
			prepareBatchFunc: func(
				_ context.Context,
				query string,
				opts ...driver.PrepareBatchOption,
			) (driver.Batch, error) {
				gotQuery = query
				gotOptsLen = len(opts)

				return batch, nil
			},
		},
	}

	err := pool.InsertBatch(
		t.Context(),
		"INSERT INTO users (id, name)",
		func(writer BatchWriter) error {
			_, exposesLifecycle := writer.(driver.Batch)
			assert.False(t, exposesLifecycle)

			require.NoError(t, writer.Append(uint64(1), "Alice"))
			require.NoError(t, writer.AppendStruct(struct {
				ID   uint64
				Name string
			}{
				ID:   2,
				Name: "Bob",
			}))
			assert.Same(t, column, writer.Column(1))

			return nil
		},
		driver.WithReleaseConnection(),
		driver.WithCloseOnFlush(),
	)
	require.NoError(t, err)

	assert.Equal(t, "INSERT INTO users (id, name)", gotQuery)
	assert.Equal(t, 2, gotOptsLen)

	require.Len(t, batch.appended, 1)
	assert.Equal(t, []any{uint64(1), "Alice"}, batch.appended[0])
	require.Len(t, batch.appendedStructs, 1)
	assert.Equal(t, []int{1}, batch.columnIndexes)

	assert.Equal(t, 1, batch.sendCalls)
	assert.Equal(t, 1, batch.closeCalls)
	assert.Equal(t, []string{"send", "close"}, batch.lifecycle)
}

func TestInsertBatchRejectsNilFunction(t *testing.T) {
	t.Parallel()

	err := (&Pool{}).InsertBatch(
		t.Context(),
		"INSERT INTO users",
		nil,
	)

	require.EqualError(t, err, "xch: batch function is nil")
}

func TestInsertBatchWrapsPrepareError(t *testing.T) {
	t.Parallel()

	prepareErr := errors.New("prepare failed")

	pool := &Pool{
		conn: &batchTestConn{
			prepareBatchFunc: func(
				context.Context,
				string,
				...driver.PrepareBatchOption,
			) (driver.Batch, error) {
				return nil, prepareErr
			},
		},
	}

	err := pool.InsertBatch(
		t.Context(),
		"INSERT INTO users",
		func(BatchWriter) error { return nil },
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, prepareErr)
	assert.ErrorContains(t, err, "xch: prepare batch:")
}

func TestInsertBatchLifecycle(t *testing.T) {
	t.Parallel()

	callbackErr := errors.New("callback failed")
	sendErr := errors.New("send failed")
	closeErr := errors.New("close failed")

	tests := []struct {
		name          string
		callbackErr   error
		sendErr       error
		closeErr      error
		wantErrors    []error
		wantContains  []string
		wantSendCalls int
	}{
		{
			name:          "success",
			wantSendCalls: 1,
		},
		{
			name:        "callback error",
			callbackErr: callbackErr,
			wantErrors:  []error{callbackErr},
		},
		{
			name:          "send error",
			sendErr:       sendErr,
			wantErrors:    []error{sendErr},
			wantContains:  []string{"xch: send batch:"},
			wantSendCalls: 1,
		},
		{
			name:          "close error",
			closeErr:      closeErr,
			wantErrors:    []error{closeErr},
			wantContains:  []string{"xch: close batch:"},
			wantSendCalls: 1,
		},
		{
			name:         "callback and close errors",
			callbackErr:  callbackErr,
			closeErr:     closeErr,
			wantErrors:   []error{callbackErr, closeErr},
			wantContains: []string{"xch: close batch:"},
		},
		{
			name:          "send and close errors",
			sendErr:       sendErr,
			closeErr:      closeErr,
			wantErrors:    []error{sendErr, closeErr},
			wantContains:  []string{"xch: send batch:", "xch: close batch:"},
			wantSendCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			batch := &batchTestBatch{
				sendErr:  test.sendErr,
				closeErr: test.closeErr,
			}
			pool := newBatchTestPool(batch)

			err := pool.InsertBatch(
				t.Context(),
				"INSERT INTO users",
				func(BatchWriter) error {
					return test.callbackErr
				},
			)

			if len(test.wantErrors) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)

				for _, wantErr := range test.wantErrors {
					assert.ErrorIs(t, err, wantErr)
				}
				for _, want := range test.wantContains {
					assert.ErrorContains(t, err, want)
				}
			}

			assert.Equal(t, test.wantSendCalls, batch.sendCalls)
			assert.Equal(t, 1, batch.closeCalls)
		})
	}
}

func TestInsertBatchClosesBatchOnPanic(t *testing.T) {
	t.Parallel()

	batch := &batchTestBatch{}
	pool := newBatchTestPool(batch)

	assert.PanicsWithValue(t, "boom", func() {
		_ = pool.InsertBatch(
			t.Context(),
			"INSERT INTO users",
			func(BatchWriter) error {
				panic("boom")
			},
		)
	})

	assert.Equal(t, 0, batch.sendCalls)
	assert.Equal(t, 1, batch.closeCalls)
}

func newBatchTestPool(batch driver.Batch) *Pool {
	return &Pool{
		conn: &batchTestConn{
			prepareBatchFunc: func(
				context.Context,
				string,
				...driver.PrepareBatchOption,
			) (driver.Batch, error) {
				return batch, nil
			},
		},
	}
}

type batchTestConn struct {
	driver.Conn

	prepareBatchFunc func(
		context.Context,
		string,
		...driver.PrepareBatchOption,
	) (driver.Batch, error)
}

func (conn *batchTestConn) PrepareBatch(
	ctx context.Context,
	query string,
	opts ...driver.PrepareBatchOption,
) (driver.Batch, error) {
	return conn.prepareBatchFunc(ctx, query, opts...)
}

type batchTestBatch struct {
	driver.Batch

	sendErr  error
	closeErr error

	column driver.BatchColumn

	appended        [][]any
	appendedStructs []any
	columnIndexes   []int
	sendCalls       int
	closeCalls      int
	lifecycle       []string
}

func (batch *batchTestBatch) Append(values ...any) error {
	batch.appended = append(batch.appended, values)

	return nil
}

func (batch *batchTestBatch) AppendStruct(value any) error {
	batch.appendedStructs = append(batch.appendedStructs, value)

	return nil
}

func (batch *batchTestBatch) Column(index int) driver.BatchColumn {
	batch.columnIndexes = append(batch.columnIndexes, index)

	return batch.column
}

func (batch *batchTestBatch) Send() error {
	batch.sendCalls++
	batch.lifecycle = append(batch.lifecycle, "send")

	return batch.sendErr
}

func (batch *batchTestBatch) Close() error {
	batch.closeCalls++
	batch.lifecycle = append(batch.lifecycle, "close")

	return batch.closeErr
}

type batchTestColumn struct {
	driver.BatchColumn
}
