package resolver

import (
	"context"
	"template/app"
	"template/loader"

	"github.com/scott-rc/core"
)

type Resolver struct{}

func (r *Resolver) core(ctx context.Context, operation string) *app.Core {
	c := &app.Core{
		Core:   ctx.Value(core.ContextKey).(*core.Core),
		Loader: ctx.Value(app.LoaderKey).(*loader.Loader),
	}
	c.Core.Context = ctx
	c.AddOp(operation)
	return c
}
