package app

import (
	"context"
	"template/loader"

	"github.com/scott-rc/core"
)

type contextKey int

const (
	LoaderKey = contextKey(iota)
)

type Core struct {
	*core.Core
	Loader *loader.Loader
}

func ContextDecorator() core.ResolverContextDecorator {
	return func(ctx context.Context) context.Context {
		ctx = context.WithValue(ctx, LoaderKey, loader.NewLoader())
		return ctx
	}
}
