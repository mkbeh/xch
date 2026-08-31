package xch

import (
	"context"
	"errors"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// BatchWriter provides row and column population operations for an insert
// batch. Its lifecycle is managed by [Pool.InsertBatch].
//
// BatchWriter and values returned by Column are valid only while the
// InsertBatch callback is running and must not be retained afterward.
type BatchWriter interface {
	Append(values ...any) error
	AppendStruct(value any) error
	Column(index int) driver.BatchColumn
}

type batchWriter struct {
	batch driver.Batch
}

func (writer batchWriter) Append(values ...any) error {
	return writer.batch.Append(values...)
}

func (writer batchWriter) AppendStruct(value any) error {
	return writer.batch.AppendStruct(value)
}

func (writer batchWriter) Column(index int) driver.BatchColumn {
	return writer.batch.Column(index)
}

// InsertBatch prepares a ClickHouse insert batch, invokes fn to populate it,
// and sends the batch when the callback succeeds.
//
// InsertBatch owns the batch lifecycle. If fn returns an error, buffered rows
// are not sent. If fn panics, the batch is closed before the panic propagates.
//
// For direct access to the underlying clickhouse-go batch lifecycle, use
// [Pool.PrepareBatch].
func (p *Pool) InsertBatch(
	ctx context.Context,
	query string,
	fn func(BatchWriter) error,
	opts ...driver.PrepareBatchOption,
) (err error) {
	if fn == nil {
		return errors.New("xch: batch function is nil")
	}

	batch, err := p.conn.PrepareBatch(ctx, query, opts...)
	if err != nil {
		return fmt.Errorf("xch: prepare batch: %w", err)
	}

	defer func() {
		if closeErr := batch.Close(); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf("xch: close batch: %w", closeErr),
			)
		}
	}()

	if err := fn(batchWriter{batch: batch}); err != nil {
		return err
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("xch: send batch: %w", err)
	}

	return nil
}
