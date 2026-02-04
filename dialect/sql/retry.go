// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sql

import (
	"context"
	"database/sql"
	"errors"

	"entgo.io/ent/dialect"
	"github.com/ydb-platform/ydb-go-sdk/v3/retry"
)

// RetryExecutor is an interface for database operations with automatic retries.
type RetryExecutor interface {
	// Do executes the given function within a retry loop without a transaction.
	// The function receives a dialect.Driver that wraps the connection.
	// opts are driver-specific retry options (e.g., ydb retry.Option).
	Do(
		ctx context.Context,
		fn func(ctx context.Context, drv dialect.Driver) error,
		opts ...any,
	) error

	// DoTx executes the given function within a retry loop with a transaction.
	// The function receives a dialect.Driver that wraps the database/sql.Tx transaction.
	// opts are driver-specific retry options (e.g., ydb retry.Option).
	DoTx(
		ctx context.Context,
		fn func(ctx context.Context, drv dialect.Driver) error,
		opts ...any,
	) error
}

// NewRetryExecutor creates a new RetryExecutor with the given database connection
func NewRetryExecutor(
	sqlDialect string,
	db *sql.DB,
) RetryExecutor {
	if sqlDialect == dialect.YDB && db != nil {
		return &YDBRetryExecutor{db: db}
	} else {
		return nil
	}
}

// RetryExecutorGetter is an optional interface that drivers can implement to provide
// a RetryExecutor for automatic retry handling.
// If a driver implements this interface,
// sqlgraph operations will use the RetryExecutor for database operations.
type RetryExecutorGetter interface {
	// RetryExecutor returns the RetryExecutor for this driver.
	// If nil is returned, no retry handling will be applied.
	RetryExecutor() RetryExecutor
}

// GetRetryExecutor returns the RetryExecutor for the given driver if available.
// If the driver is wrapped with a DebugDriver, the returned executor will preserve
// debug logging by wrapping the driver passed to callback functions.
func GetRetryExecutor(drv dialect.Driver) RetryExecutor {
	var logFn func(context.Context, ...any)
	if dd, ok := drv.(*dialect.DebugDriver); ok {
		logFn = dd.Log()
		drv = dd.Driver
	}
	if getter, ok := drv.(RetryExecutorGetter); ok {
		executor := getter.RetryExecutor()
		if executor == nil {
			return nil
		}
		if logFn != nil {
			return &debugRetryExecutor{
				RetryExecutor: executor,
				log:           logFn,
			}
		}
		return executor
	}
	return nil
}

// debugRetryExecutor wraps a RetryExecutor to preserve debug logging.
type debugRetryExecutor struct {
	RetryExecutor
	log func(context.Context, ...any)
}

// Do executes the operation with debug logging preserved.
func (d *debugRetryExecutor) Do(
	ctx context.Context,
	fn func(ctx context.Context, drv dialect.Driver) error,
	opts ...any,
) error {
	return d.RetryExecutor.Do(
		ctx,
		func(ctx context.Context, drv dialect.Driver) error {
			return fn(ctx, dialect.DebugWithContext(drv, d.log))
		},
		opts...,
	)
}

// DoTx executes the operation within a transaction with debug logging preserved.
func (d *debugRetryExecutor) DoTx(
	ctx context.Context,
	fn func(ctx context.Context, drv dialect.Driver) error,
	opts ...any,
) error {
	return d.RetryExecutor.DoTx(
		ctx,
		func(ctx context.Context, drv dialect.Driver) error {
			return fn(ctx, dialect.DebugWithContext(drv, d.log))
		},
		opts...,
	)
}

// RetryConfig holds retry configuration for sqlgraph operations.
// This is used to pass retry options to the RetryExecutor.
type RetryConfig struct {
	// Options are driver-specific retry options.
	Options []any
}

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
	err := retry.Do(
		ctx,
		r.db,
		func(ctx context.Context, conn *sql.Conn) error {
			return fn(ctx, newConnRetryDriver(conn))
		},
		retry.WithDoRetryOptions(toRetryOptions(opts)...),
	)
	return unwrapYDBError(err)
}

// DoTx executes the operation within a transaction with retry support.
// It uses ydb-go-sdk's retry.DoTx which handles YDB-specific retryable errors.
// Options should be created using retry.WithIdempotent(), retry.WithLabel(), etc.
func (r *YDBRetryExecutor) DoTx(
	ctx context.Context,
	fn func(ctx context.Context, drv dialect.Driver) error,
	opts ...any,
) error {
	err := retry.DoTx(
		ctx,
		r.db,
		func(ctx context.Context, tx *sql.Tx) error {
			return fn(ctx, newTxRetryDriver(tx))
		},
		retry.WithDoTxRetryOptions(toRetryOptions(opts)...),
	)
	return unwrapYDBError(err)
}

// unwrapYDBError extracts the original error from YDB's error wrapping.
// YDB SDK wraps errors with stack traces and retry context, which changes
// the error message.
func unwrapYDBError(err error) error {
	if err == nil {
		return nil
	}
	original := err
	for {
		unwrapped := errors.Unwrap(original)
		if unwrapped == nil {
			break
		}
		original = unwrapped
	}
	return original
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

// retryDriver is designed for use only in sqlgraph,
// specifically - in retry.DoTx callbacks
type retryDriver struct {
	Conn
}

var _ dialect.Driver = (*retryDriver)(nil)

// newConnRetryDriver creates a new RetryDriver from a database connection.
func newConnRetryDriver(conn *sql.Conn) *retryDriver {
	return &retryDriver{
		Conn: Conn{ExecQuerier: conn},
	}
}

// newTxRetryDriver creates a new RetryDriver from a transaction.
func newTxRetryDriver(tx *sql.Tx) *retryDriver {
	return &retryDriver{
		Conn: Conn{ExecQuerier: tx},
	}
}

// sqlgraph creates nested transactions in several methods.
// But YDB doesnt support nested transactions.
// Therefore, this methods returns no-op tx
func (d *retryDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}

// Close is a no-op for RetryDriver since retry.DoTx manages the transaction lifecycle.
func (d *retryDriver) Close() error {
	return nil
}

// Dialect returns the YDB dialect name.
func (d *retryDriver) Dialect() string {
	return dialect.YDB
}
