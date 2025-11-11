// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	entSql "entgo.io/ent/dialect/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestOpenAndClose(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	driver := &YDBDriver{
		Driver: entSql.OpenDB(dialect.YDB, db),
	}

	// When
	mock.ExpectClose()
	err = driver.Close()

	// Then - verify closed
	require.NoError(t, err, "should close connection")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecCreateTable(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	driver := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	mock.ExpectExec("DROP TABLE IF EXISTS test_users").
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectExec("CREATE TABLE test_users").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_ = driver.Exec(ctx, "DROP TABLE IF EXISTS test_users", []any{}, nil)
	err = driver.Exec(ctx, `CREATE TABLE test_users (
		id Int64 NOT NULL,
		name Utf8,
		age Int32,
		PRIMARY KEY (id)
	)`, []any{}, nil)
	require.NoError(t, err, "CREATE TABLE should execute without err")

	// Then - verify table created
	mock.ExpectQuery("SELECT 1 FROM test_users").
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	var rows entSql.Rows
	err = driver.Query(ctx, "SELECT 1 FROM test_users", []any{}, &rows)
	require.NoError(t, err, "created table should exist")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecInsert(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	driver := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	mock.ExpectExec("INSERT INTO test_users").
		WillReturnResult(sqlmock.NewResult(1, 1))

	insertQuery := `INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)`
	err = driver.Exec(ctx, insertQuery, []any{}, nil)
	require.NoError(t, err, "INSERT data execute without err")

	// Then - verify row count
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) AS").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	var rows entSql.Rows
	err = driver.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", []any{}, &rows)
	require.NoError(t, err, "SELECT COUNT(*) should execute without err")

	require.True(t, rows.Next(), "Result should have at least 1 row")
	var count uint64
	err = rows.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count, "Table should contain exactly 1 row")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecUpdate(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	mock.ExpectExec("INSERT INTO test_users").
		WillReturnResult(sqlmock.NewResult(1, 1))

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, drv.Exec(ctx, insertDataQuery, []any{}, nil))

	// When
	mock.ExpectExec("UPDATE test_users SET age = 31 WHERE id = 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	updateQuery := `UPDATE test_users SET age = 31 WHERE id = 1`
	err = drv.Exec(ctx, updateQuery, []any{}, nil)
	require.NoError(t, err, "should update data")

	// Then
	mock.ExpectQuery("SELECT \\* FROM test_users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).
			AddRow(1, "Alice", 31))

	var rows entSql.Rows
	err = drv.Query(ctx, "SELECT * FROM test_users", []any{}, &rows)
	require.NoError(t, err)

	require.True(t, rows.Next())
	var id, age int64
	var name string
	err = rows.Scan(&id, &name, &age)
	require.NoError(t, err)
	require.Equal(t, int64(31), age, "Age should've been changed")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecDelete(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	mock.ExpectExec("INSERT INTO test_users").
		WillReturnResult(sqlmock.NewResult(1, 1))

	insertDataQuery := "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 30)"
	require.NoError(t, drv.Exec(ctx, insertDataQuery, []any{}, nil))

	// When
	mock.ExpectExec("DELETE FROM test_users WHERE id = 1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	deleteQuery := `DELETE FROM test_users WHERE id = 1`
	err = drv.Exec(ctx, deleteQuery, []any{}, nil)
	require.NoError(t, err, "DELETE request should execute without err")

	// Then
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	var rows entSql.Rows
	err = drv.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", []any{}, &rows)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var count uint64
	err = rows.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, uint64(0), count, "Table should be empty after DELETE")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryEmptyTable(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	mock.ExpectQuery("SELECT \\* FROM test_users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}))

	var rows entSql.Rows
	err = drv.Query(ctx, "SELECT * FROM test_users", []any{}, &rows)

	// Then
	require.NoError(t, err, "SELECT data should execute without err")

	counter := 0
	for rows.Next() {
		counter++
	}
	require.Equal(t, 0, counter, "Table should be empty")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecMultipleInserts(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	for i := 0; i < 10; i++ {
		insertQuery := fmt.Sprintf("INSERT INTO test_users (id, name, age) VALUES (%d, 'User%d', 20)", i, i)
		mock.ExpectExec("INSERT INTO test_users").
			WillReturnResult(sqlmock.NewResult(int64(i), 1))

		err := drv.Exec(ctx, insertQuery, []any{}, nil)
		require.NoError(t, err)
	}

	// Then
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	var rows entSql.Rows
	err = drv.Query(ctx, "SELECT COUNT(*) AS `count` FROM test_users", []any{}, &rows)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var count uint64
	err = rows.Scan(&count)
	require.NoError(t, err)
	require.Equal(t, uint64(10), count, "Table should contain exactly 10 rows")
	rows.Close()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryInvalidQuery(t *testing.T) {
	// Given
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	invalidQuery := "SELECT * FROM non_existent_table"
	mock.ExpectQuery("SELECT \\* FROM non_existent_table").
		WillReturnError(fmt.Errorf("table not found"))

	var rows entSql.Rows
	err = drv.Query(ctx, invalidQuery, []any{}, &rows)

	// Then
	require.Error(t, err, "should return error for invalid query")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContextCancellation(t *testing.T) {
	// Given
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	drv := &YDBDriver{Driver: entSql.OpenDB(dialect.YDB, db)}

	// When
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Then
	err = drv.Exec(cancelCtx, "SELECT 1", []any{}, nil)
	require.Error(t, err, "should return error when context is cancelled")
	require.Contains(t, err.Error(), "context canceled")
}
