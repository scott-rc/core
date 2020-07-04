package resolver_test

import (
	"template/models"
	"template/test/testsuite"
	"template/types"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/suite"
)

type selfResolverTestSuite struct {
	*testsuite.Suite
}

func TestSelfResolver(t *testing.T) {
	suite.Run(t, &selfResolverTestSuite{&testsuite.Suite{}})
}

func (s *selfResolverTestSuite) Test_SelfCreate() {
	// arrange
	expected := types.SelfCreateInputType{
		Email:    faker.Email(),
		Password: faker.Password(),
	}

	// act
	self, err := s.Resolver.SelfCreate(s.Core.Context, &struct {
		Self types.SelfCreateInputType
	}{
		Self: expected,
	})
	s.NoError(err)
	user, err := models.FindUser(s.Core.Context, s.Core.Db, int(self.Id()))
	s.NoError(err)

	// assert
	s.Equal(expected.Email, self.Email())
	s.NotEqual(expected.Password, user.PasswordHash)
}

func (s *selfResolverTestSuite) Test_SelfCreate_Validation() {
	// arrange
	badInputs := []types.SelfCreateInputType{
		{Email: "", Password: ""},
		{Email: faker.Email(), Password: ""},
		{Email: faker.Email(), Password: "2short"},
		{Email: "", Password: faker.Password()},
		{Email: "not_an_email", Password: faker.Password()},
	}

	for _, input := range badInputs {
		// act
		self, err := s.Resolver.SelfCreate(s.Core.Context, &struct {
			Self types.SelfCreateInputType
		}{
			Self: input,
		})

		// assert
		s.IsType(validator.ValidationErrors{}, err)
		s.Nil(self)
	}
}

func (s *selfResolverTestSuite) Test_SelfAuthenticate() {
	// arrange
	pw := faker.Password()
	self := s.Factory.CreateUser(map[string]interface{}{models.UserColumns.PasswordHash: pw})

	// act
	token, err := s.Resolver.SelfAuthenticate(s.Core.Context, &struct {
		Credentials types.SelfAuthenticateInputType
	}{
		Credentials: types.SelfAuthenticateInputType{
			Email:    self.Email,
			Password: pw,
		},
	})

	// assert
	s.NoError(err)
	s.NotEmpty(token)
}
