// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxCommit(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	err = tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(1), count, "Commit should persist rows")
}

func TestTxRollback(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	err = tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil)
	require.NoError(t, err)

	err = tx.Rollback()
	require.NoError(t, err)

	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(0), count, "Rollback should discard rows")
}

func TestTxMultipleOperations(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	statements := []string{
		"INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)",
		"INSERT INTO test_users (id, name, age) VALUES (2, 'Bob', 25)",
		"INSERT INTO test_users (id, name, age) VALUES (3, 'Charlie', 35)",
		"UPDATE test_users SET age = 31 WHERE id = 1",
	}

	for _, stmt := range statements {
		err = tx.Exec(ctx, stmt, nil, nil)
		require.NoError(t, err, "Statement should execute without error: %s", stmt)
	}

	err = tx.Commit()
	require.NoError(t, err)

	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(3), count, "all inserts should persist")

	var row struct {
		Age int64 `sql:"age"`
	}
	scanSingleRow(t, ctx, driver, "SELECT age FROM test_users WHERE id = 1", &row)
	require.Equal(t, int64(31), row.Age, "Update inside transaction should persist")
}

func TestTxQueryWithinTransaction(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	require.NoError(t, driver.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	var result *sql.Rows
	err = tx.Query(ctx, "SELECT id, name FROM test_users WHERE id = 1", nil, &result)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NoError(t, result.Close())

	err = tx.Commit()
	require.NoError(t, err)
}

func TestTxUseAfterCommit(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))
	require.NoError(t, tx.Commit())

	err = tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (2, 'Bob', 25)", nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction has already been committed or rolled back")
}

func TestTxUseAfterRollback(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))
	require.NoError(t, tx.Rollback())

	err = tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (2, 'Bob', 25)", nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction has already been committed or rolled back")
}

func TestTxDoubleCommit(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))
	require.NoError(t, tx.Commit())

	err = tx.Commit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction has already been committed or rolled back")
}

func TestTxDoubleRollback(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	require.NoError(t, tx.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))
	require.NoError(t, tx.Rollback())

	err = tx.Rollback()
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction has already been committed or rolled back")
}

func TestTxInvalidQuery(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	err = tx.Exec(ctx, "INSERT INTO non_existent_table (id) VALUES (1)", nil, nil)
	require.Error(t, err)

	err = tx.Rollback()
	if err != nil {
        require.Contains(t, err.Error(), "Transaction not found")
    }
}

func TestTxQueryInvalidResultType(t *testing.T) {
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)
	require.NoError(t, driver.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))

	tx, err := driver.Tx(ctx)
	require.NoError(t, err)

	var wrongType int
	err = tx.Query(ctx, "SELECT id FROM test_users WHERE id = 1", nil, &wrongType)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid type")

	err = tx.Rollback()
	require.NoError(t, err)
}
