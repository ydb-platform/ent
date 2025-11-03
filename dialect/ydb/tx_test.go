// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckTxState(t *testing.T) {
	t.Run("normal state", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       nil,
		}
		
		err := tx.checkTxState()
		require.NoError(t, err)
	})
	
	t.Run("tx is about to close", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: true,
			startErr:       nil,
		}
		
		err := tx.checkTxState()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction already closed")
	})
	
	t.Run("tx failed to start", func(t *testing.T) {
		startErr := errors.New("start failed")
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       startErr,
		}
		
		err := tx.checkTxState()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction failed to start")
		require.Contains(t, err.Error(), "start failed")
	})
}

func TestPrepareTxClose(t *testing.T) {
	t.Run("first call succeeds", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       nil,
		}
		
		err := tx.prepareTxClose()
		require.NoError(t, err)
		require.True(t, tx.isAboutToClose, "isAboutToClose should be set to true")
	})
	
	t.Run("second call fails", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       nil,
		}
		
		err := tx.prepareTxClose()
		require.NoError(t, err)
		
		err = tx.prepareTxClose()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction already closed")
	})
	
	t.Run("fails if start failed", func(t *testing.T) {
		startErr := errors.New("start failed")
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       startErr,
		}
		
		err := tx.prepareTxClose()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction failed to start")
		require.False(t, tx.isAboutToClose, "isAboutToClose should remain false")
	})
}

func TestCheckStartOrClosedErrs(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       nil,
		}
		
		err := tx.checkStartOrClosedErrs()
		require.NoError(t, err)
	})
	
	t.Run("start error", func(t *testing.T) {
		startErr := errors.New("db error")
		tx := &YDBTx{
			isAboutToClose: false,
			startErr:       startErr,
		}
		
		err := tx.checkStartOrClosedErrs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction failed to start")
		require.Contains(t, err.Error(), "db error")
	})
	
	t.Run("already closed", func(t *testing.T) {
		tx := &YDBTx{
			isAboutToClose: true,
			startErr:       nil,
		}
		
		err := tx.checkStartOrClosedErrs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction already closed")
	})
	
	t.Run("both errors - start error takes precedence", func(t *testing.T) {
		startErr := errors.New("start error")
		tx := &YDBTx{
			isAboutToClose: true,
			startErr:       startErr,
		}
		
		err := tx.checkStartOrClosedErrs()
		require.Error(t, err)
		require.Contains(t, err.Error(), "transaction failed to start")
	})
}
