package resolver_test

import (
	"template/models"
	"template/test/testsuite"
	"template/types"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/suite"
)

type todoResolverTestSuite struct {
	*testsuite.Suite
}

func TestTodoResolver(t *testing.T) {
	suite.Run(t, &todoResolverTestSuite{&testsuite.Suite{}})
}

func (s *todoResolverTestSuite) Test_TodoInsert() {
	// arrange
	s.SignIn(nil)
	todo := s.Factory.GenTodo(nil)

	// act
	result, err := s.Resolver.TodoCreate(s.Core.Context, &struct {
		Todo types.TodoCreateInputType
	}{
		Todo: types.TodoCreateInputType{
			Title:       todo.Title,
			CompletedAt: s.Factory.ToGraphqlTime(todo.CompletedAt),
		}})

	// assert
	s.NoError(err)
	s.Equal(todo.Title, result.Title())
	s.Equal(todo.CompletedAt.Time, result.CompletedAt().Time)
}

func (s *todoResolverTestSuite) Test_TodoInsert_Validation() {
	// arrange
	s.SignIn(nil)
	badInputs := []types.TodoCreateInputType{
		{Title: "", CompletedAt: nil},
	}

	for _, input := range badInputs {
		// act
		todo, emptyTitle := s.Resolver.TodoCreate(s.Core.Context, &struct {
			Todo types.TodoCreateInputType
		}{
			Todo: input,
		})

		// assert
		s.IsType(validator.ValidationErrors{}, emptyTitle)
		s.Nil(todo)
	}
}

func (s *todoResolverTestSuite) Test_TodoUpdate() {
	// arrange
	s.SignIn(nil)
	todo := s.Factory.CreateTodo(map[string]interface{}{models.UserColumns.UserID: s.User.UserID})
	expected := types.TodoUpdateInputType{
		TodoId:      int32(todo.TodoID),
		Title:       faker.Paragraph(),
		CompletedAt: s.Factory.ToGraphqlTime(time.Now()),
	}

	// act
	result, err := s.Resolver.TodoUpdate(s.Core.Context, &struct {
		Todo types.TodoUpdateInputType
	}{
		Todo: expected})

	// assert
	s.NoError(err)
	s.NoError(todo.Reload(s.Core.Context, s.Core.Db))
	s.Equal(expected.Title, result.Title())
	s.Equal(expected.Title, todo.Title)
	s.Equal(expected.CompletedAt.Time, result.CompletedAt().Time)
	s.Equal(expected.CompletedAt.Time, expected.CompletedAt.Time)
}

func (s *todoResolverTestSuite) Test_TodoUpdate_Validation() {
	// arrange
	s.SignIn(nil)
	s.Factory.CreateTodo(map[string]interface{}{models.TodoColumns.UserID: s.User.UserID})
	badInputs := []types.TodoUpdateInputType{
		{TodoId: 0, Title: "", CompletedAt: nil},
		{TodoId: 0, Title: faker.Paragraph(), CompletedAt: s.Factory.NillableGraphqlTime(time.Now())},
		{TodoId: 1, Title: "", CompletedAt: s.Factory.NillableGraphqlTime(time.Now())},
	}

	for _, input := range badInputs {
		// act
		todo, err := s.Resolver.TodoUpdate(s.Core.Context, &struct {
			Todo types.TodoUpdateInputType
		}{
			Todo: input,
		})

		// assert
		s.IsType(validator.ValidationErrors{}, err)
		s.Nil(todo)
	}
}
