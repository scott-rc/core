package types

import (
	"template/app"
	"template/loader"
	"template/models"

	"github.com/graph-gophers/graphql-go"
)

type TodoType struct {
	core *app.Core
	todo *models.Todo
}

func NewTodoType(core *app.Core, todo *models.Todo) *TodoType {
	return &TodoType{core: core, todo: todo}
}

func NewTodoTypes(core *app.Core, todos []*models.Todo) []*TodoType {
	res := make([]*TodoType, len(todos))
	for _, todo := range todos {
		res = append(res, NewTodoType(core, todo))
	}
	return res
}

func (t *TodoType) TodoId() int32 {
	return int32(t.todo.TodoID)
}

func (t *TodoType) Title() string {
	return t.todo.Title
}

func (t *TodoType) CreatedAt() graphql.Time {
	return graphql.Time{Time: t.todo.CreatedAt}
}

func (t *TodoType) UpdatedAt() graphql.Time {
	return graphql.Time{Time: t.todo.UpdatedAt}
}

func (t *TodoType) CompletedAt() *graphql.Time {
	if t.todo.CompletedAt.Valid {
		return &graphql.Time{Time: t.todo.CompletedAt.Time}
	}
	return nil
}

func (t *TodoType) User() (*UserType, error) {
	user, err := t.core.Loader.Users.ByUserId.Load(t.core.Context, loader.IntToKey(t.todo.UserID))()
	if err != nil {
		return nil, err
	}
	return NewUserType(t.core, user.(*models.User)), nil
}

type TodoCreateInputType struct {
	Title       string        `validate:"required"`
	CompletedAt *graphql.Time `validate:""`
}

type TodoUpdateInputType struct {
	TodoId      int32  `validate:"required,min=1"`
	Title       string `validate:"required"`
	CompletedAt *graphql.Time
}
