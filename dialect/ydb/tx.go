// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
)

// YDBTx implements [dialect.Tx] for YDB driver and represents YBD's interactive transaction.
type YDBTx struct {
	dialect.Tx

	driver *YDBDriver
	sqlTx  *sql.Tx
}

func newYDBTx(
	ctx context.Context,
	driver *YDBDriver,
	opts *sql.TxOptions,
) (*YDBTx, error) {
	tx, err := driver.dbSqlDriver.BeginTx(
		ctx,
		opts,
	)
	if err != nil {
		return nil, err
	}

	return &YDBTx{
		driver: driver,
		sqlTx:  tx,
	}, nil
}

// Exec implements dialect.Exec method
//
// args [any] must be an instance of [YqlOptions]
// v [any] must be [*sql.Result] or nil
func (tx *YDBTx) Exec(ctx context.Context, query string, args any, v any) error {
	yqlOpts, ok := args.(YqlOptions)
	if !ok && args != nil {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T  of 'args'. Expect dialect/ydb.YqlOptions",
			args,
		)
	}

	resPtr, ok := v.(*sql.Result)
	if !ok && v != nil {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T of 'v'.  expect *database/sql.Result",
			v,
		)
	}

	res, err := tx.sqlTx.ExecContext(
		ctx,
		query,
		yqlOpts.queryArgs...,
	)
	if err != nil {
		return err
	}

	if resPtr != nil {
		*resPtr = res
	}
	return nil
}

// Query implements the dialect.Query method.
//
// args [any] must be an instance of [YqlOptions]
// v [any] must be a *[*sql.Rows]
func (tx *YDBTx) Query(ctx context.Context, query string, args any, v any) error {
	yqlOpts, ok := args.(YqlOptions)
	if !ok && args != nil {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T  of 'args'. Expect dialect/ydb.YqlOptions",
			args,
		)
	}

	rowsPtr, ok := v.(**sql.Rows)
	if !ok {
		return fmt.Errorf(
			"dialect/ydb: invalid type %T of 'v'. expect **database/sql.Rows",
			v,
		)
	}

	rows, err := tx.sqlTx.QueryContext(
		ctx,
		query,
		yqlOpts.queryArgs...,
	)
	if err != nil {
		return err
	}

	*rowsPtr = rows
	return nil
}

// Commit implements [sql.Tx.Commit] method
func (tx *YDBTx) Commit() error {
	return tx.sqlTx.Commit()
}

// Commit implements [sql.Tx.Rollback] method
func (tx *YDBTx) Rollback() error {
	return tx.sqlTx.Rollback()
}
