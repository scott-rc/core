package types

import (
	"template/app"
	"template/models"

	"github.com/graph-gophers/graphql-go"
)

type SelfType struct {
	core *app.Core
	user *models.User
}

func NewSelfType(ctx *app.Core, user *models.User) *SelfType {
	return &SelfType{core: ctx, user: user}
}

func (s *SelfType) UserId() int32 {
	return int32(s.user.UserID)
}

func (s *SelfType) Email() string {
	return s.user.Email
}

func (s *SelfType) CreatedAt() graphql.Time {
	return graphql.Time{Time: s.user.CreatedAt}
}

func (s *SelfType) UpdatedAt() graphql.Time {
	return graphql.Time{Time: s.user.UpdatedAt}
}

func (s *SelfType) Todos() ([]*TodoType, error) {
	todos, err := s.user.Todos().All(s.core.Context, s.core.Db)
	if err != nil {
		return nil, err
	}
	return NewTodoTypes(s.core, todos), nil
}

type SelfCreateInputType struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
}

type SelfAuthenticateInputType struct {
	Email    string `validate:"required"`
	Password string `validate:"required"`
}
