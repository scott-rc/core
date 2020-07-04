package resolver

import (
	"context"
	"template/models"
	"template/types"

	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/scott-rc/core"

	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func (r *Resolver) Todo(ctx context.Context, args *struct{ Id int32 }) (*types.TodoType, error) {
	c := r.core(ctx, "resolver.Todo")
	todo, err := models.FindTodo(c.Context, c.Db, int(args.Id))
	if err != nil {
		return nil, err
	}
	return types.NewTodoType(c, todo), nil
}

func (r *Resolver) Todos(ctx context.Context, args *types.QueryMods) ([]*types.TodoType, error) {
	c := r.core(ctx, "resolver.Todos")
	todos, err := models.Todos(append(args.GetQueryMods(), qm.OrderBy(models.TodoColumns.CreatedAt))...).All(c.Context, c.Db)
	if err != nil {
		return nil, err
	}

	todoTypes := make([]*types.TodoType, len(todos))
	for i, todo := range todos {
		todoTypes[i] = types.NewTodoType(c, todo)
	}
	return todoTypes, nil
}

func (r *Resolver) TodoCreate(ctx context.Context, args *struct{ Todo types.TodoCreateInputType }) (*types.TodoType, error) {
	c := r.core(ctx, "resolver.TodoCreate")
	if c.Session.IsGuest() {
		return nil, core.KindUnauthorized
	}

	err := c.Validate.Struct(args.Todo)
	if err != nil {
		return nil, err
	}

	todo := &models.Todo{Title: args.Todo.Title, UserID: c.Session.UserId()}
	if args.Todo.CompletedAt != nil {
		todo.CompletedAt = null.TimeFrom(args.Todo.CompletedAt.Time)
	}

	err = todo.Insert(c.Context, c.Db, boil.Infer())
	if err != nil {
		return nil, err
	}

	return types.NewTodoType(c, todo), nil
}

func (r *Resolver) TodoUpdate(ctx context.Context, args *struct{ Todo types.TodoUpdateInputType }) (*types.TodoType, error) {
	c := r.core(ctx, "resolver.TodoUpdate")
	if c.Session.IsGuest() {
		return nil, core.KindUnauthorized
	}

	err := c.Validate.Struct(args.Todo)
	if err != nil {
		return nil, err
	}

	todo, err := models.FindTodo(c.Context, c.Db, int(args.Todo.Id))
	if err != nil {
		return nil, err
	}

	todo.Title = args.Todo.Title
	if args.Todo.CompletedAt != nil {
		todo.CompletedAt = null.TimeFrom(args.Todo.CompletedAt.Time)
	} else {
		todo.CompletedAt = null.Time{}
	}

	_, err = todo.Update(c.Context, c.Db, boil.Whitelist(models.TodoColumns.Title, models.TodoColumns.CompletedAt))
	if err != nil {
		return nil, err
	}

	return types.NewTodoType(c, todo), nil
}
