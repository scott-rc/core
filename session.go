package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// Session
type Session interface {
	IsLoggedIn() bool
	IsGuest() bool
	Token() string
	SetToken(string) error
	UserId() int
	SetUserId(int)
}

type session struct {
	core        *Core
	token       *jwt.Token
	tokenString string
}

// StartSession
func (c *Core) StartSession() error {
	c.Session = &session{core: c}

	cookie, err := c.Request.Cookie("token")
	if err == nil {
		err = c.Session.SetToken(cookie.Value)
		if err != nil {
			return err
		}
	}

	token := c.Request.URL.Query().Get("token")
	if token != "" {
		err := c.Session.SetToken(token)
		if err != nil {
			return err
		}
	}

	auth := c.Request.Header.Get("Authorization")
	if auth != "" {
		if !strings.HasPrefix(auth, "Bearer") {
			return jwt.NewValidationError("Authorization header must begin with 'Bearer'. (eg 'Bearer {token}')", 0)
		}
		if len(auth) < 7 {
			return jwt.NewValidationError("Authorization header missing token", 0)
		}
		err := c.Session.SetToken(auth[7:])
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *session) IsLoggedIn() bool {
	return s.token != nil && s.token.Valid
}

func (s *session) IsGuest() bool {
	return !s.IsLoggedIn()
}

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
