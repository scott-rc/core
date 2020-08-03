package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	accessTokenKey  = "access_token"
	refreshTokenKey = "refresh_token"
)

// Session
type Session interface {
	// IsLoggedIn
	IsLoggedIn() bool
	// IsAnonymous
	IsAnonymous() bool
	// AccessToken
	AccessToken() string
	// RefreshAccessToken
	RefreshAccessToken() bool
	// UserId
	UserId() int
	// Login
	Login(int)
	// Logout
	Logout()
}

// session
type session struct {
	core              *Core
	accessToken       *jwt.Token
	accessTokenString string
}

// StartSession checks the request for a token, and if one is found, attaches it to the *core.Core.
//
// Access Tokens are searched for in multiple places, with some places having a higher priority than
// others (in-case an access token exists in more than one place).
//
// The order of precedence from highest to lowest is:
// - Authorization header (Bearer)
// - Query parameter (access_token)
// - Cookie (access_token)
func (c *Core) StartSession() error {
	c.Session = &session{core: c}
	accessTokenString, err := getAccessTokenString(c)
	if err != nil {
		return err
	}

	if accessTokenString != "" {
		accessToken, err := parseToken(c, accessTokenString, false)
		if err != nil {
			return err
		}

		c.Session = &session{core: c, accessToken: accessToken, accessTokenString: accessTokenString}
	}

	return nil
}

// IsLoggedIn
func (s *session) IsLoggedIn() bool {
	return s.accessToken != nil && s.accessToken.Valid
}

// IsAnonymous
func (s *session) IsAnonymous() bool {
	return !s.IsLoggedIn()
}

// AccessToken
func (s *session) AccessToken() string {
	if s.IsAnonymous() {
		return ""
	}

	if s.accessTokenString == "" {
		signedString, err := s.accessToken.SignedString([]byte(s.core.Config.CoreConfig().Server.Jwt.AccessToken.Secret))
		if err != nil {
			s.core.Logger.DPanic("issue signing access token", "error", err, "accessToken", s.accessToken)
		}
		s.accessTokenString = signedString
	}

	return s.accessTokenString
}

// RefreshAccessToken
func (s *session) RefreshAccessToken() bool {
	refreshToken := getRefreshToken(s.core)
	if refreshToken == nil {
		return false
	}

	userId, err := strconv.Atoi(refreshToken.Claims.(*jwt.StandardClaims).Subject)
	if err != nil {
		setRefreshToken(s.core, nil)
		s.core.Logger.DPanic("could not parse refresh token's subject to an int")
	}

	*s = session{core: s.core, accessToken: generateToken(s.core, userId, false)}
	return true
}

// UserId
func (s *session) UserId() int {
	if s.IsAnonymous() {
		return 0
	}
	id, err := strconv.Atoi(s.accessToken.Claims.(*jwt.StandardClaims).Subject)
	if err != nil {
		s.core.Logger.DPanic("could not parse access token's subject to an int", "error", err)
	}
	return id
}

// Login
func (s *session) Login(userId int) {
	if s.core.Config.CoreConfig().Server.Jwt.RefreshToken != nil {
		refreshToken := generateToken(s.core, userId, true)
		setRefreshToken(s.core, refreshToken)
	}
	*s = session{core: s.core, accessToken: generateToken(s.core, userId, false)}
}

// Logout
func (s *session) Logout() {
	setRefreshToken(s.core, nil)
}

func getAccessTokenString(core *Core) (string, error) {
	accessTokenCookie, err := core.Request.Cookie(accessTokenKey)
	if err == nil {
		core.Logger.Debug("using access token within cookie", "accessToken", accessTokenCookie.Value)
		return accessTokenCookie.Value, nil
	}

	auth := core.Request.Header.Get("Authorization")
	if auth != "" {
		if !strings.HasPrefix(auth, "Bearer ") || len(auth) < 8 {
			return "", NewError(core, KindInvalidJwt, "Authorization header must begin with 'Bearer' followed by the access token. (eg 'Bearer {access_token}')")
		}
		core.Logger.Debug("using access token within authorization header", "accessToken", auth[7:])
		return auth[7:], nil
	}

	token := core.Request.URL.Query().Get(accessTokenKey)
	if token != "" {
		core.Logger.Debug("using access token within query parameter")
	}

	return token, nil
}

func getRefreshToken(core *Core) *jwt.Token {
	refreshTokenCookie, err := core.Request.Cookie(refreshTokenKey)
	if err != nil {
		return nil
	}

	core.Logger.Debug("using refresh token within cookie", "refreshToken", refreshTokenCookie.Value)
	refreshToken, err := parseToken(core, refreshTokenCookie.Value, true)
	if err != nil {
		setRefreshToken(core, nil)
		return nil
	}

	return refreshToken
}

func generateToken(core *Core, userId int, isRefreshToken bool) *jwt.Token {
	cfg := core.Config.CoreConfig().Server.Jwt.AccessToken
	if isRefreshToken {
		cfg = *core.Config.CoreConfig().Server.Jwt.RefreshToken
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.StandardClaims{
		Id:        core.Id,
		Audience:  strings.Join(cfg.Audience, ","),
		ExpiresAt: time.Now().Add(cfg.ExpiresAt).Unix(),
		IssuedAt:  time.Now().Unix(), Issuer: cfg.Issuer,
		NotBefore: time.Now().Add(cfg.NotBefore).Unix(),
		Subject:   strconv.Itoa(userId),
	})
	token.Valid = true

	if isRefreshToken {
		core.Logger.Debug("generated refresh token", "token", token)
	} else {
		core.Logger.Debug("generated access token", "token", token)
	}

	return token
}

func parseToken(core *Core, token string, isRefreshToken bool) (*jwt.Token, error) {
	msg := "parsing access token"
	secret := core.Config.CoreConfig().Server.Jwt.AccessToken.Secret
	if isRefreshToken {
		msg = "parsing refresh token"
		secret = core.Config.CoreConfig().Server.Jwt.RefreshToken.Secret
	}

	core.Logger.Debug(msg, "token", token)
	return jwt.ParseWithClaims(token, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]), jwt.ValidationErrorSignatureInvalid)
		}
		return []byte(secret), nil
	})
}

func setRefreshToken(core *Core, refreshToken *jwt.Token) {
	refreshTokenString := ""
	maxAge := -1
	var err error

	if refreshToken != nil {
		refreshTokenString, err = refreshToken.SignedString([]byte(core.Config.CoreConfig().Server.Jwt.RefreshToken.Secret))
		if err == nil {
			maxAge = int(core.Config.CoreConfig().Server.Jwt.RefreshToken.ExpiresAt.Seconds())
		} else {
			core.Logger.DPanic("issue signing refresh token", "error", err, "refreshToken", refreshToken)
		}
	}

	http.SetCookie(core.w, &http.Cookie{
		Name:     refreshTokenKey,
		Value:    refreshTokenString,
		MaxAge:   maxAge,
		Secure:   core.Config.CoreConfig().Env != EnvDevelopment,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
