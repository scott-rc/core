package factory

import (
	"math/rand"
	"template/app"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
)

type Factory struct {
	core   *app.Core
	assert *require.Assertions
}

func NewFactory(c *app.Core, assertions *require.Assertions) *Factory {
	return &Factory{
		core:   c,
		assert: assertions,
	}
}

func (f *Factory) NillableTime(x time.Time) null.Time {
	if rand.Float32() > 0.5 {
		return null.NewTime(x, true)
	}
	return null.NewTime(time.Time{}, false)
}

func (f *Factory) NillableGraphqlTime(x time.Time) *graphql.Time {
	t := f.NillableTime(x)
	if t.Valid {
		return &graphql.Time{Time: t.Time}
	}
	return nil
}

func (f *Factory) ToGraphqlTime(x interface{}) *graphql.Time {
	switch t := x.(type) {
	case time.Time:
		return &graphql.Time{Time: t}
	case null.Time:
		if t.Valid {
			return &graphql.Time{Time: t.Time}
		}
		return nil
	default:
		f.assert.FailNow("expected a time type", "x", x)
	}
	return nil
}
