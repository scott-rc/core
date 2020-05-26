package core

import (
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap/zapcore"
)

// ErrorDeterminer is a function that takes an err and returns an ErrorKind.
type ErrorDeterminer func(*Core, error) ErrorKind

// ErrorDetailer is a function that takes a *core.Error so that it can add details to it.
type ErrorDetailer func(e *Error)

var (
	detail    ErrorDetailer
	determine ErrorDeterminer
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
	// KindRouteNotFound
	KindRouteNotFound = ErrorKind{404_000, "Not Found", "The requested url does not exist", zapcore.DebugLevel}
	// KindMethodNotAllowed
	KindMethodNotAllowed = ErrorKind{405_000, "Method Not Allowed", "The requested url does not support that HTTP method", zapcore.DebugLevel}
	// KindUnknown
	KindUnknown = ErrorKind{500_000, "Unexpected Error", "An unexpected error occurred while processing your request. Please try again later.", zapcore.ErrorLevel}
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
		// err is already an Error
		return e
	}

	e := Error{core: core, Cause: err, Kind: KindUnknown}

	if kind, ok := err.(ErrorKind); ok {
		// err is an ErrorKind, replace the kind and message
		e.Kind = kind
		e.Message = kind.Message
	}

	// override any given arguments
	for _, arg := range args {
		switch arg := arg.(type) {
		case ErrorKind:
			e.Kind = arg
		case string:
			e.Message = arg
		case zapcore.Level:
			e.Kind.Severity = arg
		default:
			core.Logger.DPanic("invalid argument given to core.NewError", "argument", arg)
		}
	}

	if e.Kind == KindUnknown {
		// we still don't know the ErrorKind, let them determine it
		e.Kind = determine(core, e.Cause)
	}

	// let them add any details to the Error
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
