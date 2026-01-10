// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"
	ydb "github.com/ydb-platform/ydb-go-sdk/v3"
)

// YDBDriver is a [dialect.Driver] implementation for YDB.
type YDBDriver struct {
	*entSql.Driver

	nativeDriver *ydb.Driver
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
		return nil, err
	}

	dbSQLDriver := sql.OpenDB(conn)

	return &YDBDriver{
		Driver:       entSql.OpenDB(dialect.YDB, dbSQLDriver),
		nativeDriver: nativeDriver,
	}, nil
}

func (y *YDBDriver) NativeDriver() *ydb.Driver {
	return y.nativeDriver
}
