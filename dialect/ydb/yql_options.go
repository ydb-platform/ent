package ydb

import (
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
)

type ctxKey int

const ctxKeyOptions ctxKey = 0

// YqlOptions holds all YDB-specific query options
// Usage example:
// opts := ydb.NewOptions().
//
//	WithDoOptions(query.WithIdempotent()).
//	WithExecOptions(query.WithParameters(...)).
//	WithExecOptions(query.WithTxControl(...))
type YqlOptions struct {
	doOptions   []query.DoOption
	execOptions []query.ExecuteOption
	queryArgs   []any
}

func NewYqlOptions() *YqlOptions {
	return &YqlOptions{}
}

func (o *YqlOptions) WithDoOptions(opts ...query.DoOption) *YqlOptions {
	o.doOptions = append(o.doOptions, opts...)
	return o
}

func (o *YqlOptions) WithExecOptions(opts ...query.ExecuteOption) *YqlOptions {
	o.execOptions = append(o.execOptions, opts...)
	return o
}

func (o *YqlOptions) WithQueryArgs(args ...any) *YqlOptions {
	o.queryArgs = append(o.queryArgs, args...)
	return o
}
