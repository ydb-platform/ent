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
)

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
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
		driver.Close()
	})

	// When
	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	err = driver.Exec(ctx, createTableQuery, nil, nil)
	require.NoError(t, err, "CREATE TABLE should execute without err")

	// Then
	var result query.Result
	err = driver.Query(ctx, "SELECT 1 FROM test_users", nil, &result)
	require.NoError(t, err, "created table should exist")
}

func TestExecInsert(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
		driver.Close()
	})

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	err = driver.Exec(ctx, createTableQuery, nil, nil)
	require.NoError(t, err)

	// When
	insertQuery := `
		INSERT INTO test_users (id, name, age)
		VALUES (1, 'Alice', 30)
	`
	err = driver.Exec(ctx, insertQuery, nil, nil)
	require.NoError(t, err, "INSERT data execute without err")

	// Then
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err = driver.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", nil, &result)
	require.NoError(t, err, "SELECT data after INSERT should execute without err")
	require.NotNil(t, result, "Result of SELECT should not be nil")

	for {
		rs, err := result.NextResultSet(ctx)
		require.NoError(t, err, "Result should contain at least 1 result set")

		for {
			row, err := rs.NextRow(ctx)
			require.NoError(t, err, "Result set should contain at least 1 row")

			var resStruct struct {
				Count uint64 `sql:"count"`
			}
			err = row.ScanStruct(&resStruct)
			require.NoError(t, err, "Row scanning should execute without err")

			require.Equal(t, uint64(1), resStruct.Count, "Table should contain exactly 1 row")
			break
		}
		break
	}
}

func TestExecUpdate(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
		driver.Close()
	})

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, driver.Exec(ctx, createTableQuery, nil, nil))

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, driver.Exec(ctx, insertDataQuery, nil, nil))

	// When
	updateQuery := `
		UPDATE test_users
		SET age = 31
		WHERE id = 1
	`
	err = driver.Exec(ctx, updateQuery, nil, nil)
	require.NoError(t, err, "should update data")

	// Then
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err = driver.Query(ctx, "SELECT * FROM test_users", nil, &result)
	require.NoError(t, err, "SELECT data after UPDATE should succeed")
	require.NotNil(t, result, "Result of SELECt should not be nil")

	for {
		rs, err := result.NextResultSet(ctx)
		require.NoError(t, err, "Result should contain at least 1 result set")

		for {
			row, err := rs.NextRow(ctx)
			require.NoError(t, err, "Result set should contain at least 1 row")

			var resStruct struct {
				Id   int64  `sql:"id"`
				Name string `sql:"name"`
				Age  int64  `sql:"age"`
			}
			err = row.ScanStruct(&resStruct)
			require.NoError(t, err, "Row scanning should succeed")

			require.Equal(t, int64(31), resStruct.Age, "Age should've been changed")
			break
		}
		break
	}
}

func TestExecDelete(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should succeed")
		driver.Close()
	})

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, driver.Exec(ctx, createTableQuery, nil, nil))

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, driver.Exec(ctx, insertDataQuery, nil, nil))

	// When
	deleteQuery := `
		DELETE FROM test_users
		WHERE id = 1
	`
	err = driver.Exec(ctx, deleteQuery, nil, nil)
	require.NoError(t, err, "DELETE request should execute without err")

	// Then
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err = driver.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", nil, &result)
	require.NoError(t, err, "SELECT data after DELETE should execute without err")
	require.NotNil(t, result, "Result of SELECT should not be nil")

	for {
		rs, err := result.NextResultSet(ctx)
		require.NoError(t, err, "Result should contain at least 1 result set")

		for {
			row, err := rs.NextRow(ctx)
			require.NoError(t, err, "Result set should contain at least 1 row")

			var resStruct struct {
				Count uint64 `sql:"count"`
			}
			err = row.ScanStruct(&resStruct)
			require.NoError(t, err, "Row scanning should execute without err")

			require.Equal(t, uint64(0), resStruct.Count, "Table should contain exactly 1 row")
			break
		}
		break
	}
}

func TestQueryEmptyTable(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
		driver.Close()
	})

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, driver.Exec(ctx, createTableQuery, nil, nil))

	// When
	selectQuery := "SELECT * FROM test_users"

	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err = driver.Query(ctx, selectQuery, nil, &result)

	// Then
	require.NoError(t, err, "SELECT data should execute without err")
	require.NotNil(t, result, "Result of SELECT should not be nil")

	for {
		rs, err := result.NextResultSet(ctx)
		require.NoError(t, err, "Result should contain at least 1 result set")

		counter := 0
		for _, err := range rs.Rows(ctx) {
			require.NoError(t, err, "Rows are supposed to be empty, so no error should happen")
			counter++
		}
		require.Equal(t, 0, counter, "Table should be empty")
		break
	}
}

func TestExecMultipleInserts(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		err := driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
		require.NoError(t, err, "DROP TABLE should execute without err")
		driver.Close()
	})

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, driver.Exec(ctx, createTableQuery, nil, nil))

	// When
	for i := 0; i < 10; i++ {
		insertQuery := fmt.Sprintf("INSERT INTO test_users (id, name, age) VALUES (%d, 'User%d', 20)", i, i)
		err = driver.Exec(ctx, insertQuery, nil, nil)
		require.NoError(t, err)
	}

	// Then
	var result query.Result
	t.Cleanup(func() { result.Close(ctx) })

	err = driver.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", nil, &result)
	require.NoError(t, err)
	require.NotNil(t, result, "Result of SELECT should not be nil")

	for {
		rs, err := result.NextResultSet(ctx)
		require.NoError(t, err, "Result should contain at least 1 result set")

		for {
			row, err := rs.NextRow(ctx)
			require.NoError(t, err, "Result set should contain at least 1 row")

			var resStruct struct {
				Count uint64 `sql:"count"`
			}
			err = row.ScanStruct(&resStruct)
			require.NoError(t, err, "Row scanning should execute without err")

			require.Equal(t, uint64(10), resStruct.Count, "Table should contain exactly 1 row")
			break
		}
		break
	}
}

func TestQueryInvalidQuery(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		driver.Close()
	})

	// When
	invalidQuery := "SELECT * FROM non_existent_table"
	var result query.Result
	err = driver.Query(ctx, invalidQuery, nil, &result)

	// Then
	require.Error(t, err, "should return error for invalid query")
}

func TestContextCancellation(t *testing.T) {
	// Given
	ctx := context.Background()
	driver, err := Open(ctx, defaultDSN)
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		driver.Close()
	})

	// When
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Then
	err = driver.Exec(cancelCtx, "SELECT 1", nil, nil)
	require.Error(t, err, "should return error when context is cancelled")
}
