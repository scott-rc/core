package factory

import (
	"template/models"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func (f *Factory) overrideTodo(model *models.Todo, overrides map[string]interface{}) {
	for column, val := range overrides {
		switch column {
		case models.TodoColumns.CompletedAt:
			model.CompletedAt = val.(null.Time)
		case models.TodoColumns.CreatedAt:
			model.CreatedAt = val.(time.Time)
		case models.TodoColumns.Title:
			model.Title = val.(string)
		case models.TodoColumns.UpdatedAt:
			model.UpdatedAt = val.(time.Time)
		case models.TodoColumns.UserID:
			model.UserID = val.(int)
		default:
			f.assert.FailNow("unknown todo column", "column", column)
		}
	}
}

func (f *Factory) GenTodo(overrides map[string]interface{}) *models.Todo {
	todo := &models.Todo{
		CompletedAt: f.NillableTime(time.Now()),
		CreatedAt:   time.Now(),
		Title:       faker.Sentence(),
		TodoID:      0,
		UpdatedAt:   time.Now(),
		UserID:      0,
	}

	f.overrideTodo(todo, overrides)
	return todo
}

func (f *Factory) CreateTodo(overrides map[string]interface{}) *models.Todo {
	todo := f.GenTodo(overrides)
	f.assert.NoError(todo.Insert(f.core.Context, f.core.Db, boil.Infer()))
	return todo
}
