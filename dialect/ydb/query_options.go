package ydb

import (
	"context"

	"github.com/ydb-platform/ydb-go-sdk/v3/query"
)

const (
	ctxKeyDoOptions int = iota
	ctxKeyDoTxOptions
	ctxKeyExecuteOptions
)

func WithDoOptions(ctx context.Context, opts ...query.DoOption) context.Context {
	return context.WithValue(ctx, ctxKeyDoOptions, opts)
}

func WithDoTxOptions(ctx context.Context, opts ...query.DoTxOption) context.Context {
	return context.WithValue(ctx, ctxKeyDoTxOptions, opts)
}

func WithExecOptions(ctx context.Context, opts ...query.ExecuteOption) context.Context {
	return context.WithValue(ctx, ctxKeyExecuteOptions, opts)
}

func getDoOptions(ctx context.Context) []query.DoOption {
	if opts, ok := ctx.Value(ctxKeyDoOptions).([]query.DoOption); ok {
		return opts
	}
	return []query.DoOption{}
}

func getDoTxOptions(ctx context.Context) []query.DoTxOption {
	if opts, ok := ctx.Value(ctxKeyDoTxOptions).([]query.DoTxOption); ok {
		return opts
	}
	return []query.DoTxOption{}
}

func getExecOptions(ctx context.Context) []query.ExecuteOption {
	if opts, ok := ctx.Value(ctxKeyExecuteOptions).([]query.ExecuteOption); ok {
		return opts
	}
	return []query.ExecuteOption{}
}
