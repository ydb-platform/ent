// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"
	ydb "github.com/ydb-platform/ydb-go-sdk/v3"
	ydbQuery "github.com/ydb-platform/ydb-go-sdk/v3/query"
)

// YDBDriver is a dialect.Driver implementation for YDB.
type YDBDriver struct {
	dialect.Driver

	driver *ydb.Driver
}

func Open(ctx context.Context, dsn string) (*YDBDriver, error) {
	db, err := ydb.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &YDBDriver{
		driver: db,
	}, nil
}

// Exec implements the dialect.Exec method.
// v any is never used since YDB's Executor.Exec never returns value
func (y *YDBDriver) Exec(ctx context.Context, query string, args any, v any) error {
	doOpts := getDoOptions(ctx)
	execOpts := getExecOptions(ctx)

	return y.driver.Query().Do(
		ctx,
		func(ctx context.Context, s ydbQuery.Session) error {
			return s.Exec(
				ctx,
				query,
				execOpts...,
			)
		},
		doOpts...,
	)
}

// Query implements the dialect.Query method.
// Type of v any must be *github.com/ydb-platform/ydb-go-sdk/v3/query.Result
func (y *YDBDriver) Query(ctx context.Context, query string, args any, v any) error {
	ydbResult, ok := v.(*ydbQuery.Result)
	if !ok {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T. expect *github.com/ydb-platform/ydb-go-sdk/v3/query.Result",
			v,
		)
	}

	doOpts := getDoOptions(ctx)
	execOpts := getExecOptions(ctx)

	return y.driver.Query().Do(
		ctx,
		func(ctx context.Context, s ydbQuery.Session) error {
			result, err := s.Query(
				ctx,
				query,
				execOpts...,
			)
			if err != nil {
				return err
			}

			defer func() {
				_ = result.Close(ctx)
			}()

			*ydbResult = result
			return nil
		},
		doOpts...,
	)
}

// Tx starts and returns a new YDB interactive transaction.
func (y *YDBDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return newYDBTx(ctx, y)
}

// Close closes the underlying connection.
func (y *YDBDriver) Close() error {
	if y.driver == nil {
		return nil
	}
	ctx := context.Background()
	return y.driver.Close(ctx)
}

// Dialect implements the dialect.Dialect method.
func (y *YDBDriver) Dialect() string {
	return dialect.YDB
}
