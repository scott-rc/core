package loader

import (
	"context"
	"template/models"

	"github.com/graph-gophers/dataloader"
)

type Users struct {
	ByUserId dataloader.Interface
}

func usersByUserIds(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ids := KeysToInts(keys)
	users, err := models.Users(models.UserWhere.UserID.IN(ids)).All(ctx, db(ctx))
	results := make([]*dataloader.Result, len(keys))
	for i, user := range users {
		results[i] = &dataloader.Result{Data: user, Error: err}
	}
	return results
}
