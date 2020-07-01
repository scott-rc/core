package types

import (
	"template/app"
	"template/models"
)

type UserType struct {
	core *app.Core
	user *models.User
}

func NewUserType(ctx *app.Core, user *models.User) *UserType {
	return &UserType{core: ctx, user: user}
}

func (u *UserType) Email() string {
	return u.user.Email
}

func (u *UserType) Todos() ([]*TodoType, error) {
	todos, err := u.user.Todos().All(u.core.Context, u.core.Db)
	if err != nil {
		return nil, err
	}
	return NewTodoTypes(u.core, todos), nil
}
