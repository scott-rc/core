package core

import (
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	detail    ErrorDetailer
	determine ErrorDeterminer
)

// ErrorKind
type ErrorKind struct {
	// Code
	Code int
	// Title
	Title string
	// Message
	Message string
	// Severity
	Severity zapcore.Level
}

var (
	// KindInvalidJson
	KindInvalidJson = ErrorKind{400_000, "Invalid JSON", "You're payload contains invalid JSON", zapcore.InfoLevel}
	// KindRouteNotFound
	KindRouteNotFound = ErrorKind{404_000, "Not Found", "That route does not exist", zapcore.DebugLevel}
	// KindMethodNotAllowed
	KindMethodNotAllowed = ErrorKind{405_000, "Method Not Allowed", "That HTTP method is not allowed for this route", zapcore.DebugLevel}
	// KindUnknown
	KindUnknown = ErrorKind{500_000, "", "", zapcore.ErrorLevel}
)

// Error
func (k ErrorKind) Error() string {
	if k.Message != "" {
		return k.Message
	}
	if k.Title != "" {
		return k.Title
	}
	return "Unexpected Error"
}

// MarshalLogObject is used to implement zapcore.ObjectMarshaler interface.
func (k ErrorKind) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("code", k.Code)
	enc.AddString("title", k.Title)
	enc.AddString("message", k.Message)
	return nil
}

// ErrorDeterminer
type ErrorDeterminer func(*Core, error) ErrorKind

// SetErrorDeterminer
func SetErrorDeterminer(errorDeterminer ErrorDeterminer) {
	determine = errorDeterminer
}

// Error
type Error struct {
	core       *Core
	Kind       ErrorKind `json:"kind"`
	Message    string    `json:"message"`
	Details    []string  `json:"details"`
	Operations []string  `json:"operations"`
	Cause      error     `json:"err"`
}

// NewError
func NewError(core *Core, err error, args ...interface{}) Error {
	if e, ok := err.(Error); ok {
		return e
	}

	e := Error{core: core, Cause: err, Kind: KindUnknown}

	if kind, ok := err.(ErrorKind); ok {
		e.Kind = kind
		e.Message = kind.Message
	}

	for _, arg := range args {
		switch arg := arg.(type) {
		case ErrorKind:
			e.Kind = arg
		case string:
			e.Message = arg
		case zapcore.Level:
			e.Kind.Severity = arg
		default:
			core.Logger.DPanic("invalid argument given to errors.NewError: %v", "arg", arg)
		}
	}

	if e.Kind == KindUnknown {
		e.Kind = determine(core, e.Cause)
	}

	detail(&e)

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

	if e.core.Config.CoreConfig().Env != "production" {
		extensions["operations"] = e.core.Operations
		if e.Cause != nil {
			extensions["cause"] = map[string]interface{}{
				"message": e.Cause.Error(),
				"type":    fmt.Sprintf("%T", e.Cause),
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
		e.core.Logger.DPanic("couldn't get http status from errors.Error", zap.Object("error", e))
		return http.StatusInternalServerError
	}
	return status
}

// MarshalLogObject is used to implement zapcore.ObjectMarshaler interface.
func (e Error) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("message", e.Message)
	_ = enc.AddObject("kind", e.Kind)
	_ = enc.AddObject("exception", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
		enc.AddString("type", fmt.Sprintf("%T", e.Cause))
		enc.AddString("message", e.Cause.Error())
		return nil
	}))
	return nil
}

// ErrorDetailer
type ErrorDetailer func(e *Error)

// SetErrorDetailer
func SetErrorDetailer(errorDetailer ErrorDetailer) {
	detail = errorDetailer
}
