package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	sessionKey = "token"
)

// Session
type Session interface {
	// IsLoggedIn
	IsLoggedIn() bool
	// IsGuest
	IsGuest() bool
	// Token
	Token() string
	// SetToken
	SetToken(string) error
	// UserId
	UserId() int
	// SetUserId
	SetUserId(int)
}

// session
type session struct {
	core        *Core
	token       *jwt.Token
	tokenString string
}

// StartSession checks the request for a token, and if one is found, attaches it to the *core.Core
// Tokens are searched for in multiple places, with some places having a higher priority than others (in-case a token
// exists in more than one place).
//
// The order of precedence from highest to lowest is:
// - Authorization header
// - Query parameter
// - Cookie
func (c *Core) StartSession() error {
	c.Session = &session{core: c}

	auth := c.Request.Header.Get("Authorization")
	if auth != "" {
		if !strings.HasPrefix(auth, "Bearer") {
			return jwt.NewValidationError("Authorization header must begin with 'Bearer'. (eg 'Bearer {token}')", 0)
		}
		if len(auth) < 7 {
			return jwt.NewValidationError("Authorization header is missing token", 0)
		}
		return c.Session.SetToken(auth[7:])
	}

	token := c.Request.URL.Query().Get(sessionKey)
	if token != "" {
		return c.Session.SetToken(token)
	}

	cookie, err := c.Request.Cookie(sessionKey)
	if err == nil {
		return c.Session.SetToken(cookie.Value)
	}

	return nil
}

// IsLoggedIn
func (s *session) IsLoggedIn() bool {
	return s.token != nil && s.token.Valid
}

// IsGuest
func (s *session) IsGuest() bool {
	return !s.IsLoggedIn()
}

// Token
func (s *session) Token() string {
	if s.IsGuest() {
		return ""
	}

	if s.tokenString == "" {
		signedString, _ := s.token.SignedString([]byte(s.core.Config.CoreConfig().Server.Jwt.Secret))
		s.tokenString = signedString
	}

	return s.tokenString
}

// SetToken
func (s *session) SetToken(tokenString string) error {
	s.core.Logger.Debug("parsing token", "token", tokenString)
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]), jwt.ValidationErrorSignatureInvalid)
		}
		return []byte(s.core.Config.CoreConfig().Server.Jwt.Secret), nil
	})
	if err == nil {
		*s = session{core: s.core, token: token, tokenString: tokenString}
	}
	return err
}

// UserId
func (s *session) UserId() int {
	if s.IsGuest() {
		return 0
	}
	id, err := strconv.Atoi(s.token.Claims.(*jwt.StandardClaims).Subject)
	if err != nil {
		s.core.Logger.DPanic("could not parse claims subject to an int", "error", err)
	}
	return id
}

// SetUserId
func (s *session) SetUserId(id int) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Id:        s.core.Id,
		Audience:  strings.Join(s.core.Config.CoreConfig().Server.Jwt.Audience, ","),
		ExpiresAt: time.Now().Add(s.core.Config.CoreConfig().Server.Jwt.ExpiresAt).Unix(),
		IssuedAt:  time.Now().Unix(), Issuer: s.core.Config.CoreConfig().Server.Jwt.Issuer,
		NotBefore: time.Now().Add(s.core.Config.CoreConfig().Server.Jwt.NotBefore).Unix(),
		Subject:   strconv.Itoa(id)})
	token.Valid = true
	*s = session{core: s.core, token: token}
}
