// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sql

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
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
	drv, logFn := unwrapDebugDriver(drv)

	getter, ok := drv.(RetryExecutorGetter)
	if !ok {
		return nil
	}

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

// unwrapDebugDriver extracts the underlying driver and log function from a DebugDriver.
func unwrapDebugDriver(drv dialect.Driver) (dialect.Driver, func(context.Context, ...any)) {
	if debugDriver, ok := drv.(*dialect.DebugDriver); ok {
		return debugDriver.Driver, debugDriver.Log()
	}
	return drv, nil
}

// RetryConfig holds retry configuration for sqlgraph operations.
// This is used to pass retry options to the RetryExecutor.
type RetryConfig struct {
	// Options are driver-specific retry options.
	Options []any
}
