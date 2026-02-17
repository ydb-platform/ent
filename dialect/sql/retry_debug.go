// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package sql

import (
	"context"

	"entgo.io/ent/dialect"
)

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
