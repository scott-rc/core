package loader

import (
	"context"
	"database/sql"

	"github.com/graph-gophers/dataloader"
	"github.com/scott-rc/core"
)

type Loader struct {
	Users Users
}

func NewLoader() *Loader {
	return &Loader{
		Users: Users{
			ByUserId: dataloader.NewBatchedLoader(usersByUserIds),
		},
	}
}

type intKey int

func (k intKey) String() string {
	return string(k)
}

func (k intKey) Raw() interface{} {
	return int(k)
}

func IntToKey(id int) dataloader.Key {
	return intKey(id)
}

func IntsToKeys(ids ...int) []dataloader.Key {
	keys := make([]dataloader.Key, len(ids))
	for i, val := range ids {
		keys[i] = intKey(val)
	}
	return keys
}

func KeysToInts(keys dataloader.Keys) []int {
	ints := make([]int, len(keys))
	for i, id := range keys {
		ints[i] = id.Raw().(int)
	}
	return ints
}

func db(c context.Context) *sql.DB {
	return c.Value(core.ContextKey).(*core.Core).Db
}
