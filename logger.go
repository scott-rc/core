package core

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger
type Logger interface {
	// Debug
	Debug(msg string, keysAndValues ...interface{})
	// Info
	Info(msg string, keysAndValues ...interface{})
	// Warn
	Warn(msg string, keysAndValues ...interface{})
	// Error
	Error(msg string, keysAndValues ...interface{})
	// DPanic
	DPanic(msg string, keysAndValues ...interface{})
	// Panic
	Panic(msg string, keysAndValues ...interface{})
	// Fatal
	Fatal(msg string, keysAndValues ...interface{})
	// Log
	Log(level zapcore.Level, msg string, keysAndValues ...interface{})
	// WithCore
	WithCore(*Core) Logger
	// With
	With(...interface{}) Logger
	// Clone
	Clone() Logger
	// Close
	Close() error
	// Printf
	Printf(string, ...interface{})
	// Verbose
	Verbose() bool
}

// logger
type logger struct {
	core  *Core
	impl  *zap.SugaredLogger
	level zapcore.Level
}

// newLogger
func newLogger(cfg *Config) Logger {
	encoder := zapcore.EncoderConfig{
		NameKey:       "logger",
		MessageKey:    "message",
		StacktraceKey: "stacktrace",
		CallerKey:     "caller",
		EncodeCaller:  zapcore.ShortCallerEncoder,
		LineEnding:    zapcore.DefaultLineEnding,

		// we use "time" and RFC3339 because google cloud logging requires it
		// https://cloud.google.com/logging/docs/agent/configuration?hl=en#timestamp-processing
		TimeKey:        "time",
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,

		// we use "severity" and CapitalLevelEncoder because google cloud logging requires it
		// https://cloud.google.com/logging/docs/agent/configuration?hl=en#special-fields
		LevelKey:    "severity",
		EncodeLevel: zapcore.CapitalLevelEncoder,
	}

	var level zapcore.Level
	switch cfg.Server.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	impl := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encoder), zapcore.Lock(os.Stderr), level))
	if cfg.Env == EnvDevelopment {
		impl = impl.WithOptions(zap.Development())
	}

	return &logger{core: nil, impl: impl.Sugar(), level: level}
}

// Debug
func (l *logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.DebugLevel, msg, keysAndValues...)
}

// Info
func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.InfoLevel, msg, keysAndValues...)
}

// Warn
func (l *logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.WarnLevel, msg, keysAndValues...)
}

// Error
func (l *logger) Error(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.ErrorLevel, msg, keysAndValues...)
}

// DPanic
func (l *logger) DPanic(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.DPanicLevel, msg, keysAndValues...)
}

// Panic
func (l *logger) Panic(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.PanicLevel, msg, keysAndValues...)
}

// Fatal
func (l *logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.FatalLevel, msg, keysAndValues...)
}

// Log
func (l *logger) Log(level zapcore.Level, msg string, keysAndValues ...interface{}) {
	if l.core != nil {
		keysAndValues = append(keysAndValues, "core", l.core)
	}
	switch level {
	case zapcore.DebugLevel:
		l.impl.Debugw(msg, keysAndValues...)
	case zapcore.InfoLevel:
		l.impl.Infow(msg, keysAndValues...)
	case zapcore.WarnLevel:
		l.impl.Warnw(msg, keysAndValues...)
	case zapcore.ErrorLevel:
		l.impl.Errorw(msg, keysAndValues...)
	case zapcore.DPanicLevel:
		l.impl.DPanicw(msg, keysAndValues...)
	case zapcore.PanicLevel:
		l.impl.Panicw(msg, keysAndValues...)
	case zapcore.FatalLevel:
		l.impl.Fatalw(msg, keysAndValues...)
	}
}

// WithCore
func (l *logger) WithCore(core *Core) Logger {
	return &logger{core: core, impl: l.impl}
}

// With
func (l *logger) With(keysAndValues ...interface{}) Logger {
	return &logger{core: l.core, impl: l.impl.With(keysAndValues...)}
}

// Clone
func (l *logger) Clone() Logger {
	return &logger{core: l.core, impl: l.impl.With()}
}

// Close
func (l *logger) Close() error {
	return l.impl.Sync()
}

// Printf
func (l *logger) Printf(format string, v ...interface{}) {
	content := fmt.Sprintf(format, v...)
	l.Info(content)
}

// Verbose
func (l *logger) Verbose() bool {
	return l.level == zapcore.DebugLevel
}
