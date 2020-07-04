package resolver

import (
	"context"
	"database/sql"
	"template/models"
	"template/types"

	"github.com/scott-rc/core"

	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"golang.org/x/crypto/bcrypt"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

func (r *Resolver) SelfCreate(ctx context.Context, args *struct{ Self types.SelfCreateInputType }) (*types.SelfType, error) {
	c := r.core(ctx, "resolver.SelfCreate")

	err := c.Validate.Struct(args.Self)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(args.Self.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{Email: args.Self.Email, PasswordHash: string(hash)}

	err = user.Insert(c.Context, c.Db, boil.Infer())
	if err != nil {
		return nil, err
	}

	return types.NewSelfType(c, user), nil
}

func (r *Resolver) SelfAuthenticate(ctx context.Context, args *struct {
	Credentials types.SelfAuthenticateInputType
}) (string, error) {
	c := r.core(ctx, "resolver.SelfAuthenticate")
	user, err := models.Users(qm.Where("email = ?", args.Credentials.Email)).One(c.Context, c.Db)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", core.NewError(c.Core, err, "The given email or password was incorrect")
		}
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(args.Credentials.Password))
	if err != nil {
		return "", core.NewError(c.Core, err, "The given email or password was incorrect")
	}

	c.Session.SetUserId(user.UserID)

	return c.Session.Token(), nil
}
