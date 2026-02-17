// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sql

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	"github.com/ydb-platform/ydb-go-sdk/v3/retry"
)

// YDBRetryExecutor implements sqlgraph.YDBRetryExecutor for YDB
type YDBRetryExecutor struct {
	db *sql.DB
}

// Do executes a read-only operation with retry support.
// It uses ydb-go-sdk's retry.Do which handles YDB-specific retryable errors.
// Options should be created using retry.WithIdempotent(), retry.WithLabel(), etc.
func (r *YDBRetryExecutor) Do(
	ctx context.Context,
	fn func(ctx context.Context, drv dialect.Driver) error,
	opts ...any,
) error {
	return retry.Do(
		ctx,
		r.db,
		func(ctx context.Context, conn *sql.Conn) error {
			return fn(ctx, newConnRetryDriver(conn))
		},
		retry.WithDoRetryOptions(toRetryOptions(opts)...),
	)
}

// DoTx executes the operation within a transaction with retry support.
// It uses ydb-go-sdk's retry.DoTx which handles YDB-specific retryable errors.
// Options should be created using retry.WithIdempotent(), retry.WithLabel(), etc.
func (r *YDBRetryExecutor) DoTx(
	ctx context.Context,
	fn func(ctx context.Context, drv dialect.Driver) error,
	opts ...any,
) error {
	return retry.DoTx(
		ctx,
		r.db,
		func(ctx context.Context, tx *sql.Tx) error {
			return fn(ctx, newTxRetryDriver(tx))
		},
		retry.WithDoTxRetryOptions(toRetryOptions(opts)...),
	)
}

// toRetryOptions converts a slice of any options to retry.Option slice
func toRetryOptions(opts []any) []retry.Option {
	retryOpts := make([]retry.Option, 0, len(opts))
	for _, opt := range opts {
		if ro, ok := opt.(retry.Option); ok {
			retryOpts = append(retryOpts, ro)
		}
	}
	return retryOpts
}

// ydbRetryDriver is designed for use only in sqlgraph,
// specifically - in retry.DoTx callbacks
type ydbRetryDriver struct {
	Conn
}

var _ dialect.Driver = (*ydbRetryDriver)(nil)

// newConnRetryDriver creates a new RetryDriver from a database connection.
func newConnRetryDriver(conn *sql.Conn) *ydbRetryDriver {
	return &ydbRetryDriver{
		Conn: Conn{ExecQuerier: conn},
	}
}

// newTxRetryDriver creates a new RetryDriver from a transaction.
func newTxRetryDriver(tx *sql.Tx) *ydbRetryDriver {
	return &ydbRetryDriver{
		Conn: Conn{ExecQuerier: tx},
	}
}

// sqlgraph creates nested transactions in several methods.
// But YDB doesnt support nested transactions.
// Therefore, this methods returns no-op tx
func (d *ydbRetryDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}

// Close is a no-op for RetryDriver since retry.DoTx manages the transaction lifecycle.
func (d *ydbRetryDriver) Close() error {
	return nil
}

// Dialect returns the YDB dialect name.
func (d *ydbRetryDriver) Dialect() string {
	return dialect.YDB
}
