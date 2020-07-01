package factory

import (
	"template/models"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/crypto/bcrypt"
)

func (f *Factory) overrideUser(model *models.User, overrides map[string]interface{}) {
	for column, val := range overrides {
		switch column {
		case models.UserColumns.CreatedAt:
			model.CreatedAt = val.(time.Time)
		case models.UserColumns.Email:
			model.Email = val.(string)
		case models.UserColumns.PasswordHash:
			hash, err := bcrypt.GenerateFromPassword([]byte(val.(string)), bcrypt.DefaultCost)
			f.assert.NoError(err)
			model.PasswordHash = string(hash)
		case models.UserColumns.UserID:
			model.UserID = val.(int)
		case models.UserColumns.UpdatedAt:
			model.UpdatedAt = val.(time.Time)
		default:
			f.assert.FailNow("unknown todo column", "column", column)
		}
	}
}

func (f *Factory) GenUser(overrides map[string]interface{}) *models.User {
	user := &models.User{
		CreatedAt:    time.Now(),
		Email:        faker.Email(),
		PasswordHash: faker.Password(),
		UpdatedAt:    time.Now(),
		UserID:       0,
	}
	f.overrideUser(user, overrides)
	return user
}

func (f *Factory) CreateUser(overrides map[string]interface{}) *models.User {
	user := f.GenUser(overrides)
	f.assert.NoError(user.Insert(f.core.Context, f.core.Db, boil.Infer()))
	return user
}

func (f *Factory) CreateUsers(n int, overrides map[string]interface{}) models.UserSlice {
	var users models.UserSlice
	for i := 0; i < n; i++ {
		users = append(users, f.CreateUser(overrides))
	}
	return users
}
