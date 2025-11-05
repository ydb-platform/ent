// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
)

const (
	defaultDSN = "grpc://localhost:2136/local"
	
	createTestUsersTable = `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
)

// setupDriver opens a YDB driver connection and registers cleanup
func setupDriver(t *testing.T, ctx context.Context) *YDBDriver {
	t.Helper()
	
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err, "Should open connection to YDB")
	
	t.Cleanup(func() {
		driver.Close()
	})
	
	return driver
}

// setupTable creates `test_users`` table and registers cleanup to drop it
func setupTable(
	t *testing.T,
	ctx context.Context,
	driver *YDBDriver,
) {
	t.Helper()
	
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
	})
	
	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	err := driver.Exec(ctx, createTestUsersTable, nil, nil)
	require.NoError(t, err, "CREATE TABLE should execute without err")
}

// queryRowCount executes SELECT COUNT(*) and returns the count
func queryRowCount(
	t *testing.T,
	ctx context.Context,
	driver *YDBDriver,
) uint64 {
	t.Helper()
	
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })
	
	err := driver.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", nil, &result)
	require.NoError(t, err, "SELECT COUNT(*) should execute without err")
	require.NotNil(t, result, "Result of SELECT should not be nil")
	
	rs, err := result.NextResultSet(ctx)
	require.NoError(t, err, "Result should contain at least 1 result set")
	
	row, err := rs.NextRow(ctx)
	require.NoError(t, err, "Result set should contain at least 1 row")
	
	var resStruct struct {
		Count uint64 `sql:"count"`
	}
	err = row.ScanStruct(&resStruct)
	require.NoError(t, err, "Row scanning should execute without err")
	
	return resStruct.Count
}

// scanSingleRow queries and scans a single row into the provided struct
func scanSingleRow(
	t *testing.T,
	ctx context.Context,
	driver *YDBDriver,
	queryStr string,
	dest interface{},
) {
	t.Helper()
	
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })
	
	err := driver.Query(ctx, queryStr, nil, &result)
	require.NoError(t, err, "Query should execute without err")
	require.NotNil(t, result, "Result should not be nil")
	
	rs, err := result.NextResultSet(ctx)
	require.NoError(t, err, "Result should contain at least 1 result set")
	
	row, err := rs.NextRow(ctx)
	require.NoError(t, err, "Result set should contain at least 1 row")
	
	err = row.ScanStruct(dest)
	require.NoError(t, err, "Row scanning should succeed")
}

func TestOpenAndClose(t *testing.T) {
	// Given
	ctx := context.Background()

	// When
	driver, err := Open(ctx, defaultDSN)
	
	// Then
	require.NoError(t, err, "should open connection to YDB")
	require.NotNil(t, driver)

	// When
	err = driver.Close()
	
	// Then
	require.NoError(t, err, "should close connection")
}

func TestExecCreateTable(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)

	// Cleanup
	t.Cleanup(func() {
		driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	})

	// When
	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	err := driver.Exec(ctx, createTestUsersTable, nil, nil)
	require.NoError(t, err, "CREATE TABLE should execute without err")

	// Then
	var result query.Result
	err = driver.Query(ctx, "SELECT 1 FROM test_users", nil, &result)
	require.NoError(t, err, "created table should exist")
}

func TestExecInsert(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	// When
	insertQuery := `INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)`
	err := driver.Exec(ctx, insertQuery, nil, nil)
	require.NoError(t, err, "INSERT data execute without err")

	// Then
	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(1), count, "Table should contain exactly 1 row")
}

func TestExecUpdate(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, driver.Exec(ctx, insertDataQuery, nil, nil))

	// When
	updateQuery := `UPDATE test_users SET age = 31 WHERE id = 1`
	err := driver.Exec(ctx, updateQuery, nil, nil)
	require.NoError(t, err, "should update data")

	// Then
	var resStruct struct {
		Id   int64  `sql:"id"`
		Name string `sql:"name"`
		Age  int64  `sql:"age"`
	}
	scanSingleRow(t, ctx, driver, "SELECT * FROM test_users", &resStruct)
	require.Equal(t, int64(31), resStruct.Age, "Age should've been changed")
}

func TestExecDelete(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, driver.Exec(ctx, insertDataQuery, nil, nil))

	// When
	deleteQuery := `DELETE FROM test_users WHERE id = 1`
	err := driver.Exec(ctx, deleteQuery, nil, nil)
	require.NoError(t, err, "DELETE request should execute without err")

	// Then
	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(0), count, "Table should be empty after DELETE")
}

func TestQueryEmptyTable(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	// When
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err := driver.Query(ctx, "SELECT * FROM test_users", nil, &result)

	// Then
	require.NoError(t, err, "SELECT data should execute without err")
	require.NotNil(t, result, "Result of SELECT should not be nil")

	rs, err := result.NextResultSet(ctx)
	require.NoError(t, err, "Result should contain at least 1 result set")

	counter := 0
	for _, err := range rs.Rows(ctx) {
		require.NoError(t, err, "Rows are supposed to be empty, so no error should happen")
		counter++
	}
	require.Equal(t, 0, counter, "Table should be empty")
}

func TestExecMultipleInserts(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)
	setupTable(t, ctx, driver)

	// When
	for i := 0; i < 10; i++ {
		insertQuery := fmt.Sprintf("INSERT INTO test_users (id, name, age) VALUES (%d, 'User%d', 20)", i, i)
		err := driver.Exec(ctx, insertQuery, nil, nil)
		require.NoError(t, err)
	}

	// Then
	count := queryRowCount(t, ctx, driver)
	require.Equal(t, uint64(10), count, "Table should contain exactly 10 rows")
}

func TestQueryInvalidQuery(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)

	// When
	invalidQuery := "SELECT * FROM non_existent_table"
	var result query.Result
	err := driver.Query(ctx, invalidQuery, nil, &result)

	// Then
	require.Error(t, err, "should return error for invalid query")
}

func TestContextCancellation(t *testing.T) {
	// Given
	ctx := context.Background()
	driver := setupDriver(t, ctx)

	// When
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Then
	err := driver.Exec(cancelCtx, "SELECT 1", nil, nil)
	require.Error(t, err, "should return error when context is cancelled")
}
