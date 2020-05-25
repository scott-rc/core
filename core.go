package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap/zapcore"
)

type contextKey int

// ContextKey is used to set and retrieve the *core.Core from a context.Context
const ContextKey = contextKey(iota)

// Core contains useful singletons (logger, config, db) and information specific to a request (id, session, etc...).
// It can be wrapped to contain additional fields for your application. Take a look at core.ResolverContextDecorator to
// see how to add additional fields.
//
// *core.Core will always be attached to the context.Context in your resolver method. Use the ContextKey to retrieve it.
type Core struct {
	Context    context.Context
	Config     Configuration
	Db         *sql.DB
	Id         string
	Logger     Logger
	Operations []string
	Request    *http.Request
	Session    Session
}

// AddOp adds an operation to the current core. Operations are used to display application specific
// stack traces when logging.
func (c *Core) AddOp(operation string) {
	c.Operations = append(c.Operations, operation)
	c.Logger.Debug(fmt.Sprintf("entering %s", operation))
}

// MarshalLogObject is used to implement zapcore.ObjectMarshaler interface.
func (c *Core) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("id", c.Id)
	_ = enc.AddObject("httpRequest", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("requestMethod", c.Request.Method)
		enc.AddString("requestUrl", c.Request.URL.String())
		enc.AddString("userAgent", c.Request.UserAgent())
		enc.AddString("remoteIp", c.Request.RemoteAddr)
		enc.AddString("referer", c.Request.Referer())
		return nil
	}))
	_ = enc.AddArray("operations", zapcore.ArrayMarshalerFunc(func(enc zapcore.ArrayEncoder) error {
		for _, op := range c.Operations {
			enc.AppendString(op)
		}
		return nil
	}))
	_ = enc.AddObject("session", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		s := c.Session.(*session)
		enc.AddString("token", s.tokenString)
		if s.token != nil {
			_ = enc.AddObject("claims", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				claims := s.token.Claims.(*jwt.StandardClaims)
				enc.AddString("aud", claims.Audience)
				enc.AddInt64("exp", claims.ExpiresAt)
				enc.AddString("jti", claims.Id)
				enc.AddInt64("iat", claims.IssuedAt)
				enc.AddString("iss", claims.Issuer)
				enc.AddInt64("nbf", claims.NotBefore)
				enc.AddString("sub", claims.Subject)
				return nil
			}))
		}
		return nil
	}))
	return nil
}

// Extensions is used to fill the extensions field on the root of the the response JSON object.
func (c *Core) Extensions() map[string]interface{} {
	ext := map[string]interface{}{
		"id": c.Id,
	}

	return ext
}
