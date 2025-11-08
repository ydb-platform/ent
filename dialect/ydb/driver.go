// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	ydb "github.com/ydb-platform/ydb-go-sdk/v3"
	ydbQuery "github.com/ydb-platform/ydb-go-sdk/v3/query"
)

// YDBDriver is a [dialect.Driver] implementation for YDB.
type YDBDriver struct {
	dialect.Driver

	nativeDriver *ydb.Driver
	dbSqlDriver  *sql.DB
}

func Open(ctx context.Context, dsn string) (*YDBDriver, error) {
	nativeDriver, err := ydb.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}

	conn, err := ydb.Connector(
		nativeDriver,
		ydb.WithAutoDeclare(),
		ydb.WithTablePathPrefix(nativeDriver.Name()),
	)
	if err != nil {
		panic(err)
	}

	dbSqlDriver := sql.OpenDB(conn)
	dbSqlDriver.SetMaxOpenConns(50)
	dbSqlDriver.SetMaxIdleConns(50)
	dbSqlDriver.SetConnMaxIdleTime(time.Second)

	return &YDBDriver{
		nativeDriver: nativeDriver,
		dbSqlDriver:  dbSqlDriver,
	}, nil
}

// DB returns the underlying *[sql.DB] instance.
func (y YDBDriver) DB() *sql.DB {
	return y.dbSqlDriver
}

// Exec implements the [dialect.Driver.Exec] method.
//
// args [any] must be an instance of [YqlOptions]
// v [any] is never used since YDB's [Executor.Exec] never returns value
func (y *YDBDriver) Exec(ctx context.Context, query string, args any, v any) error {
	yqlOpts, ok := args.(YqlOptions)
	if !ok && args != nil {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T  of 'args'. Expect dialect/ydb.YqlOptions",
			args,
		)
	}

	return y.nativeDriver.Query().Do(
		ctx,
		func(ctx context.Context, s ydbQuery.Session) error {
			return s.Exec(
				ctx,
				query,
				yqlOpts.execOptions...,
			)
		},
		yqlOpts.doOptions...,
	)
}

// Query implements the [dialect.Driver.Query] method.
//
// args [any] must be an instance of [YqlOptions]
// v [any] must an instance of [*ydbQuery.Result]
func (y *YDBDriver) Query(ctx context.Context, query string, args any, v any) error {
	ydbResult, ok := v.(*ydbQuery.Result)
	if !ok {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T  of 'v'. expect *github.com/ydb-platform/ydb-go-sdk/v3/query.Result",
			v,
		)
	}

	yqlOpts, ok := args.(YqlOptions)
	if !ok && args != nil {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T  of 'args'. Expect dialect/ydb.YqlOptions",
			args,
		)
	}

	return y.nativeDriver.Query().Do(
		ctx,
		func(ctx context.Context, s ydbQuery.Session) error {
			result, err := s.Query(
				ctx,
				query,
				yqlOpts.execOptions...,
			)
			if err != nil {
				return err
			}

			*ydbResult = result
			return nil
		},
		yqlOpts.doOptions...,
	)
}

// Tx starts and returns a new YDB interactive transaction.
func (y *YDBDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return newYDBTx(ctx, y)
}

// Close closes the underlying connection.
func (y *YDBDriver) Close() error {
	if y.nativeDriver == nil {
		return nil
	}
	ctx := context.Background()
	return y.nativeDriver.Close(ctx)
}

// Dialect implements the [dialect.Driver.Dialect] method.
func (y *YDBDriver) Dialect() string {
	return dialect.YDB
}
