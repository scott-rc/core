package main

import (
	"template/app"
	"template/resolver"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/scott-rc/core"
)

func main() {
	core.Run(core.Options{
		Config:           &core.Config{},
		Resolver:         &resolver.Resolver{},
		ContextDecorator: app.ContextDecorator(),
		ErrorDecorator:   core.DefaultErrorDecorator,
	})
}
