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
	ctx := context.Background()

	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err, "should open connection to YDB")
	require.NotNil(t, drv)

	err = drv.Close()
	require.NoError(t, err, "should close connection")
}

func TestExec_CreateTable(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Drop table if exists
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)

	// Create table
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	err = drv.Exec(ctx, createTableQuery, nil, nil)
	require.NoError(t, err, "should create table")

	// Cleanup
	err = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	require.NoError(t, err, "should drop table")
}

func TestExec_Insert(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup: create table
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	err = drv.Exec(ctx, createTableQuery, nil, nil)
	require.NoError(t, err)

	// Test: insert data
	insertQuery := `
		INSERT INTO test_users (id, name, age)
		VALUES (1, 'Alice', 30)
	`
	err = drv.Exec(ctx, insertQuery, nil, nil)
	require.NoError(t, err, "should insert data")

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestExec_Update(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, drv.Exec(ctx, createTableQuery, nil, nil))
	require.NoError(t, drv.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))

	// Test: update data
	updateQuery := `
		UPDATE test_users
		SET age = 31
		WHERE id = 1
	`
	err = drv.Exec(ctx, updateQuery, nil, nil)
	require.NoError(t, err, "should update data")

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestExec_Delete(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, drv.Exec(ctx, createTableQuery, nil, nil))
	require.NoError(t, drv.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))

	// Test: delete data
	deleteQuery := `
		DELETE FROM test_users
		WHERE id = 1
	`
	err = drv.Exec(ctx, deleteQuery, nil, nil)
	require.NoError(t, err, "should delete data")

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestQuery_SelectData(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, drv.Exec(ctx, createTableQuery, nil, nil))
	require.NoError(t, drv.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)", nil, nil))
	require.NoError(t, drv.Exec(ctx, "INSERT INTO test_users (id, name, age) VALUES (2, 'Bob', 25)", nil, nil))

	// Test: query data
	selectQuery := "SELECT id, name, age FROM test_users ORDER BY id"
	var result query.Result
	err = drv.Query(ctx, selectQuery, nil, &result)
	require.NoError(t, err, "should query data")
	require.NotNil(t, result, "result should not be nil")

	// Verify we got results (basic check - detailed row reading would require more YDB SDK usage)
	require.NotNil(t, result)

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestQuery_EmptyTable(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, drv.Exec(ctx, createTableQuery, nil, nil))

	// Test: query empty table
	selectQuery := "SELECT id, name, age FROM test_users"
	var result query.Result
	err = drv.Query(ctx, selectQuery, nil, &result)
	require.NoError(t, err, "should query empty table")
	require.NotNil(t, result)

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestQuery_InvalidQuery(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Test: invalid query
	invalidQuery := "SELECT * FROM non_existent_table"
	var result query.Result
	err = drv.Query(ctx, invalidQuery, nil, &result)
	require.Error(t, err, "should return error for invalid query")
}

func TestExec_InvalidQuery(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Test: invalid exec query
	invalidQuery := "INSERT INTO non_existent_table (id) VALUES (1)"
	err = drv.Exec(ctx, invalidQuery, nil, nil)
	require.Error(t, err, "should return error for invalid query")
}

func TestExec_MultipleInserts(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	// Setup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
	createTableQuery := `
		CREATE TABLE test_users (
			id Int64 NOT NULL,
			name Utf8,
			age Int32,
			PRIMARY KEY (id)
		)
	`
	require.NoError(t, drv.Exec(ctx, createTableQuery, nil, nil))

	// Test: insert multiple rows
	for i := 1; i <= 10; i++ {
		insertQuery := fmt.Sprintf("INSERT INTO test_users (id, name, age) VALUES (%d, 'User%d', 20)", i, i)
		err = drv.Exec(ctx, insertQuery, nil, nil)
		require.NoError(t, err)
	}

	// Verify with query
	var result query.Result
	err = drv.Query(ctx, "SELECT COUNT(*) as cnt FROM test_users", nil, &result)
	require.NoError(t, err)

	// Cleanup
	_ = drv.Exec(ctx, "DROP TABLE test_users", nil, nil)
}

func TestContextCancellation(t *testing.T) {
	ctx := context.Background()
	drv, err := Open(ctx, defaultDSN)
	require.NoError(t, err)
	defer drv.Close()

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = drv.Exec(cancelCtx, "SELECT 1", nil, nil)
	require.Error(t, err, "should return error when context is cancelled")
}
