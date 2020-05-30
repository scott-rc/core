package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap/zapcore"
)

var (
	uni *ut.UniversalTranslator
)

func init() {
	eng := en.New()
	uni = ut.New(eng, eng)
}

// ErrorDetailer is a function that takes a *core.Error so that it can add details to it.
type ErrorDetailer func(*Error)

func DefaultErrorDetailer(e *Error) {
	switch err := e.Cause.(type) {
	case validator.ValidationErrors:
		e.Kind = KindStructValidation
		for _, err := range err {
			e.Details = append(e.Details, err.Translate(uni.GetFallback()))
		}
	case *jwt.ValidationError:
		e.Kind = KindInvalidJwt
		e.Details = append(e.Details, err.Error())
	default:
		message := e.Cause.Error()
		switch {
		case strings.Contains(message, "models"):
			e.Kind = KindDatabase
		}
	}
}

var (
	detail ErrorDetailer
)

// ErrorKind
type ErrorKind struct {
	// Code represents the error code for this kind of error.
	// The first 3 digits are used to determine the HTTP Status that should be returned if this kind of error occurs.
	Code int
	// Title is a short, title cased description of the error.
	Title string
	// Message represents a more detailed description of the error.
	Message string
	// Severity indicates the level at which this error should be logged at.
	Severity zapcore.Level
}

var (
	// KindInvalidJson
	KindInvalidJson = ErrorKind{400_000, "Invalid JSON", "Your request body contains invalid JSON", zapcore.InfoLevel}
	// KindStructValidation
	KindStructValidation = ErrorKind{Code: 400_001, Title: "Bad Data", Message: "You're payload contains invalid data", Severity: zapcore.InfoLevel}

	// KindUnauthorized
	KindUnauthorized = ErrorKind{Code: 401_100, Title: "Unauthorized", Message: "You're not authorized to perform that action", Severity: zapcore.InfoLevel}
	// KindInvalidJwt
	KindInvalidJwt = ErrorKind{Code: 401_001, Title: "Invalid Token", Message: "Token could not be parsed", Severity: zapcore.InfoLevel}

	// KindRouteNotFound
	KindRouteNotFound = ErrorKind{404_000, "Not Found", "The requested url does not exist", zapcore.DebugLevel}

	// KindMethodNotAllowed
	KindMethodNotAllowed = ErrorKind{405_000, "Method Not Allowed", "The requested url does not support that HTTP method", zapcore.DebugLevel}

	// KindUnknown
	KindUnknown = ErrorKind{500_000, "Unexpected Error", "An unexpected error occurred while processing your request. Please try again later.", zapcore.DPanicLevel}
	// KindDatabase
	KindDatabase = ErrorKind{Code: 500_001, Severity: zapcore.ErrorLevel}
)

// Error
func (k ErrorKind) Error() string {
	if k.Message != "" {
		return k.Message
	}
	if k.Title != "" {
		return k.Title
	}
	return KindUnknown.Title
}

// MarshalLogObject is used to implement zapcore.ObjectMarshaler interface.
func (k ErrorKind) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("code", k.Code)
	enc.AddString("title", k.Title)
	enc.AddString("message", k.Message)
	return nil
}

// Error
type Error struct {
	core    *Core
	Kind    ErrorKind `json:"kind"`
	Message string    `json:"message"`
	Details []string  `json:"details"`
	Cause   error     `json:"err"`
}

// NewError
func NewError(core *Core, err error, overrides ...interface{}) Error {
	if e, ok := err.(Error); ok {
		// err is already an Error - this method has already been called
		// or it should have been constructed with all of it's fields filled out
		return e
	}

	e := Error{
		core:    core,
		Kind:    KindUnknown,
		Message: KindUnknown.Message,
		Details: []string{},
		Cause:   err,
	}

	if kind, ok := err.(ErrorKind); ok {
		// err is an ErrorKind, replace the kind
		e.Kind = kind
	}

	for _, override := range overrides {
		switch override := override.(type) {
		case ErrorKind:
			e.Kind = override
		case string:
			e.Message = override
		case zapcore.Level:
			e.Kind.Severity = override
		case error:
			e.Cause = override
		default:
			core.Logger.DPanic("invalid override given to core.NewError", "override", override)
		}
	}

	detail(&e)

	core.Logger.Log(e.Kind.Severity, "error occurred", "error", e)
	return e
}

// Error
func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Kind.Error()
}

// Extensions
func (e Error) Extensions() map[string]interface{} {
	extensions := map[string]interface{}{
		"code":  e.Kind.Code,
		"title": e.Kind.Title,
	}

	if len(e.Details) > 0 {
		extensions["details"] = e.Details
	}

	if e.core.Config.CoreConfig().Env != EnvProduction {
		extensions["operations"] = e.core.Operations
		if e.Cause != nil {
			extensions["cause"] = map[string]interface{}{
				"type":    fmt.Sprintf("%T", e.Cause),
				"message": e.Cause.Error(),
			}
		}
	}

	return extensions
}

// HttpStatus
func (e Error) HttpStatus() int {
	str := strconv.Itoa(e.Kind.Code)
	status, err := strconv.Atoi(str[:3])
	if err != nil {
		e.core.Logger.DPanic("couldn't get http status from core.Error", "error", e)
		return http.StatusInternalServerError
	}
	return status
}

// MarshalLogObject is used to implement zapcore.ObjectMarshaler interface.
func (e Error) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("message", e.Message)
	_ = enc.AddObject("kind", e.Kind)
	_ = enc.AddObject("cause", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("type", fmt.Sprintf("%T", e.Cause))
		enc.AddString("message", e.Cause.Error())
		return nil
	}))
	return nil
}
