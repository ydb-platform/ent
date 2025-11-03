// Copyright 2024-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package ydb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
)

func TestQueryWithInvalidResultType(t *testing.T) {
	drv := &YDBDriver{}

	tests := []struct {
		name   string
		result any
	}{
		{"int pointer", new(int)},
		{"string pointer", new(string)},
		{"struct pointer", new(struct{})},
		{"nil", nil},
		{"query.Result value (not pointer)", query.Result(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := drv.Query(context.Background(), "SELECT 1", nil, tt.result)
			require.Error(t, err)
			require.Contains(t, err.Error(), "dialect/ydb: invalid type")
			require.Contains(t, err.Error(), "expect *github.com/ydb-platform/ydb-go-sdk/v3/query.Result")
		})
	}
}
