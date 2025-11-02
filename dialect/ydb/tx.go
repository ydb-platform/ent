// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"fmt"
	"sync"

	"entgo.io/ent/dialect"
	ydbQuery "github.com/ydb-platform/ydb-go-sdk/v3/query"
)

// YDBTx implements dialect.Tx for YDB driver.
// YDBTx represents YBD's interactive transaction, contains all operations in a simple queue
// and execute them all in a single DoTx call on Commit().
type YDBTx struct {
	dialect.Tx

	driver *YDBDriver
	ctx    context.Context

	mutex      sync.Mutex
	operations []txOperation

	committed  bool
	rolledback bool
}

// txOperation represents a queued database operation.
type txOperation struct {
	query       string
	args        any
	result      any // pointer to store result. nil for Exec queries
	operationFn func(ydbQuery.TxActor) error
}

// Exec queues an execution operation.
func (tx *YDBTx) Exec(ctx context.Context, query string, args any, v any) error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if err := tx.checkCommitOrRollback(); err != nil {
		return err
	}

	// Queue the operation
	tx.operations = append(tx.operations, txOperation{
		query:  query,
		args:   args,
		result: nil,
		operationFn: func(actor ydbQuery.TxActor) error {
			execOpts := getExecOptions(ctx)
			return actor.Exec(ctx, query, execOpts...)
		},
	})

	return nil
}

// Query queues a query operation.
func (tx *YDBTx) Query(ctx context.Context, query string, args any, v any) error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if err := tx.checkCommitOrRollback(); err != nil {
		return err
	}

	// Queue the operation
	tx.operations = append(tx.operations, txOperation{
		query:  query,
		args:   args,
		result: v,
		operationFn: func(executor ydbQuery.TxActor) error {
			execOpts := getExecOptions(ctx)
			result, err := executor.Query(ctx, query, execOpts...)
			if err != nil {
				return err
			}

			// Store the result
			if ydbResult, ok := v.(*ydbQuery.Result); ok {
				*ydbResult = result
			}
			return nil
		},
	})

	return nil
}

// Commit executes all queued operations in a single YDB interactive transaction.
func (tx *YDBTx) Commit() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if err := tx.checkCommitOrRollback(); err != nil {
		return err
	}

	tx.committed = true

	if len(tx.operations) == 0 {
		return nil
	}

	doTxOpts := getDoTxOptions(tx.ctx)

	err := tx.driver.driver.Query().DoTx(
		tx.ctx,
		func(ctx context.Context, txActor ydbQuery.TxActor) error {
			for i, op := range tx.operations {
				if err := op.operationFn(txActor); err != nil {
					return fmt.Errorf("ydb/dialect: tx operation %d failed: %w", i, err)
				}
			}
			return nil
		},
		doTxOpts...,
	)

	return err
}

// Rollback discards all queued operations.
func (tx *YDBTx) Rollback() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if err := tx.checkCommitOrRollback(); err != nil {
		return err
	}

	tx.rolledback = true
	tx.operations = nil

	return nil
}

func (tx *YDBTx) checkCommitOrRollback() error {
	if tx.committed {
		return transactionAlreadyCommittedErr()
	}
	if tx.rolledback {
		return transactionAlreadyRolledbackErr()
	}
	return nil
}

func transactionAlreadyCommittedErr() error {
	return fmt.Errorf("ydb/dialect: transaction already committed")
}

func transactionAlreadyRolledbackErr() error {
	return fmt.Errorf("ydb/dialect: transaction already rolled back")
}
