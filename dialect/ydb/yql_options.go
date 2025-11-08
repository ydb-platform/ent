package ydb

import (
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
)

type ctxKey int

const ctxKeyOptions ctxKey = 0

// YqlOptions holds all YDB-specific query options
// Usage example:
// opts := ydb.NewOptions().
// 		WithDoOptions(query.WithIdempotent()).
//		WithExecOptions(query.WithParameters(...)).
//		WithExecOptions(query.WithTxControl(...))
type YqlOptions struct {
	doOptions   []query.DoOption
	doTxOptions []query.DoTxOption
	execOptions []query.ExecuteOption
}

func NewYqlOptions() *YqlOptions {
	return &YqlOptions{

	}
}

func (o *YqlOptions) WithDoOptions(opts ...query.DoOption) *YqlOptions {
	o.doOptions = append(o.doOptions, opts...)
	return o
}

func (o *YqlOptions) WithDoTxOptions(opts ...query.DoTxOption) *YqlOptions {
	o.doTxOptions = append(o.doTxOptions, opts...)
	return o
}

func (o *YqlOptions) WithExecOptions(opts ...query.ExecuteOption) *YqlOptions {
	o.execOptions = append(o.execOptions, opts...)
	return o
}
