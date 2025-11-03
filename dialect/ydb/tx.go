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

// YDBTx implements dialect.Tx for YDB driver and represents YBD's interactive transaction.
type YDBTx struct {
	dialect.Tx

	driver *YDBDriver

	// Channels for coordinating with DoTx goroutine
	operationsChan       chan *txOperation             // operations queue
	resultsChan          chan *txResult                // synchronous results
	commitOrRollbackChan chan *commitOrRollbackRequest // commit/rollback signal
	readySignal          chan struct{}                 // Signals DoTx is ready to receive operations
	closedSignal         chan struct{}                 // Signals DoTx has finished

	// Transaction state
	mutex          sync.Mutex
	isAboutToClose bool
	startErr       error

	doTxOpts []ydbQuery.DoTxOption
}

// txOperation represents a single Exec or Query operation.
type txOperation struct {
	query       string
	args        any
	operationFn func(ydbQuery.TxActor) error
}

// txResult is the result of executing an operation.
type txResult struct {
	err error
}

// commitOrRollbackRequest signals DoTx goroutine to commit or rollback.
type commitOrRollbackRequest struct {
	shouldCommit bool
	errChan      chan error
}

func newYDBTx(
	ctx context.Context,
	driver *YDBDriver,
) (*YDBTx, error) {
	tx := &YDBTx{
		driver:               driver,
		operationsChan:       make(chan *txOperation, 16),
		resultsChan:          make(chan *txResult),
		commitOrRollbackChan: make(chan *commitOrRollbackRequest),
		readySignal:          make(chan struct{}),
		closedSignal:         make(chan struct{}),
		doTxOpts:             getDoTxOptions(ctx),
	}

	tx.start(ctx)

	select {
	case <-tx.readySignal:
		return tx, nil
	case <-tx.closedSignal:
		return nil, txFailedToStartErr(tx.startErr)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Exec implements dialect.Exec method
func (tx *YDBTx) Exec(ctx context.Context, query string, args any, v any) error {
	if err := tx.checkTxState(); err != nil {
		return err
	}

	select {
	case tx.operationsChan <- &txOperation{
		query: query,
		args:  args,
		operationFn: func(actor ydbQuery.TxActor) error {
			return actor.Exec(
				ctx,
				query,
				getExecOptions(ctx)...,
			)
		},
	}:
	case <-ctx.Done():
		return ctx.Err()
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}

	select {
	case result := <-tx.resultsChan:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}
}

// Query implements dialect.Query method
func (tx *YDBTx) Query(ctx context.Context, query string, args any, v any) error {
	if err := tx.checkTxState(); err != nil {
		return err
	}

	select {
	case tx.operationsChan <- &txOperation{
		query: query,
		args:  args,
		operationFn: func(executor ydbQuery.TxActor) error {
			result, err := executor.Query(
				ctx,
				query,
				getExecOptions(ctx)...,
			)
			if err != nil {
				return err
			}

			ydbResult, ok := v.(*ydbQuery.Result)
			if !ok {
				return fmt.Errorf(
					"dialect/ydb: invalid type %T. expect *github.com/ydb-platform/ydb-go-sdk/v3/query.Result",
					v,
				)
			}

			*ydbResult = result
			return nil
		},
	}:
	case <-ctx.Done():
		return ctx.Err()
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}

	select {
	case result := <-tx.resultsChan:
		return result.err
	case <-ctx.Done():
		return ctx.Err()
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}
}

// Commit implements database/sql.Tx.Commit method
func (tx *YDBTx) Commit() error {
	if err := tx.prepareTxClose(); err != nil {
		return err
	}

	errChan := make(chan error, 1)
	select {
	case tx.commitOrRollbackChan <- &commitOrRollbackRequest{
		shouldCommit: true,
		errChan:      errChan,
	}:
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}

	select {
	case err := <-errChan:
		return err
	case <-tx.closedSignal:
		return txClosedUnexpectedlyErr()
	}
}

// Commit implements database/sql.Tx.Rollback method
func (tx *YDBTx) Rollback() error {
	if err := tx.prepareTxClose(); err != nil {
		return err
	}

	errChan := make(chan error, 1)
	select {
	case tx.commitOrRollbackChan <- &commitOrRollbackRequest{
		shouldCommit: false,
		errChan:      errChan,
	}:
	case <-tx.closedSignal:
		return nil
	}

	select {
	case <-errChan:
		return nil
	case <-tx.closedSignal:
		return nil
	}
}

func (tx *YDBTx) start(ctx context.Context) {
	go tx.runDoTx(ctx)
}

// runDoTx runs in background and coordinates with DoTx callback.
func (tx *YDBTx) runDoTx(ctx context.Context) {
	defer close(tx.closedSignal)

	err := tx.driver.driver.Query().DoTx(
		ctx,
		func(ctx context.Context, txActor ydbQuery.TxActor) error {
			// Signal that we're ready to receive operations
			close(tx.readySignal)

			for {
				select {
				case op := <-tx.operationsChan:
					err := op.operationFn(txActor)

					select {
					case tx.resultsChan <- &txResult{err: err}:
					case <-ctx.Done():
						return ctx.Err()
					}

				case req := <-tx.commitOrRollbackChan:
					if req.shouldCommit {
						req.errChan <- nil
						return nil
					} else {
						req.errChan <- nil
						return fmt.Errorf("dialect/ydb: transaction was rolled back")
					}

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		tx.doTxOpts...,
	)

	if err != nil {
		tx.mutex.Lock()
		tx.startErr = err
		tx.mutex.Unlock()
	}
}

func (tx *YDBTx) checkTxState() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	return tx.checkStartOrClosedErrs()
}

func (tx *YDBTx) prepareTxClose() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if err := tx.checkStartOrClosedErrs(); err != nil {
		return err
	}

	tx.isAboutToClose = true
	return nil
}

func (tx *YDBTx) checkStartOrClosedErrs() error {
	if tx.startErr != nil {
		return txFailedToStartErr(tx.startErr)
	}
	if tx.isAboutToClose {
		return txAlreadyClosedErr()
	}
	return nil
}

func txFailedToStartErr(startErr error) error {
	return fmt.Errorf("dialect/ydb: transaction failed to start: %v", startErr)
}

func txClosedUnexpectedlyErr() error {
	return fmt.Errorf("dialect/ydb: transaction closed unexpectedly")
}

func txAlreadyClosedErr() error {
	return fmt.Errorf("dialect/ydb: transaction already closed")
}
