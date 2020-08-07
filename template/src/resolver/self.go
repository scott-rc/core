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

func (r *Resolver) Self(ctx context.Context) (*types.SelfType, error) {
	c := r.core(ctx, "resolver.Self")
	if !c.Session.RefreshAccessToken() && c.Session.IsAnonymous() {
		return nil, nil
	}

	user, err := models.FindUser(c.Context, c.Db, c.Session.UserId())
	if err != nil {
		return nil, nil
	}

	return types.NewSelfType(c, user), nil
}

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

func (r *Resolver) SelfLogin(ctx context.Context, args *struct{ Credentials types.SelfLoginInputType }) (*types.SelfType, error) {
	c := r.core(ctx, "resolver.SelfLogin")
	user, err := models.Users(qm.Where("email = ?", args.Credentials.Email)).One(c.Context, c.Db)
	if err != nil {
		if err == sql.ErrNoRows {
			err = core.NewError(c.Core, err, core.KindInvalidCredentials)
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(args.Credentials.Password))
	if err != nil {
		return nil, core.NewError(c.Core, err, core.KindInvalidCredentials)
	}

	c.Session.Login(user.UserID)

	return types.NewSelfType(c, user), nil
}

func (r *Resolver) SelfLogout(ctx context.Context) (int32, error) {
	c := r.core(ctx, "resolver.SelfLogout")
	c.Session.Logout()
	return 0, nil
}
