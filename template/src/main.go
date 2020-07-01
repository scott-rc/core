package main

import (
	"template/app"
	"template/resolver"

	_ "github.com/lib/pq"
	"github.com/scott-rc/core"
)

func main() {
	cfg := &core.Config{}
	core.LoadConfig(cfg)
	core.Run(core.Options{
		Config:                   cfg,
		ErrorDetailer:            core.DefaultErrorDetailer,
		ResolverContextDecorator: app.ContextDecorator(),
		Resolver:                 &resolver.Resolver{},
	})
}
