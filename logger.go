package core

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	DPanic(msg string, keysAndValues ...interface{})
	Panic(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	Log(level zapcore.Level, msg string, keysAndValues ...interface{})
	WithCore(*Core) Logger
	With(...interface{}) Logger
	Clone() Logger
	Close() error
}

type logger struct {
	core *Core
	impl *zap.SugaredLogger
}

func newLogger(cfg *Config) (Logger, error) {
	encoder := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "severity",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var level zapcore.Level
	switch cfg.Log.Level {
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

	return &logger{core: nil, impl: impl.Sugar()}, nil
}

func (l *logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.DebugLevel, msg, keysAndValues...)
}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.InfoLevel, msg, keysAndValues...)
}

func (l *logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.WarnLevel, msg, keysAndValues...)
}

func (l *logger) Error(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.ErrorLevel, msg, keysAndValues...)
}

func (l *logger) DPanic(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.DPanicLevel, msg, keysAndValues...)
}

func (l *logger) Panic(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.PanicLevel, msg, keysAndValues...)
}

func (l *logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.Log(zapcore.FatalLevel, msg, keysAndValues...)
}

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

func (l *logger) WithCore(core *Core) Logger {
	return &logger{core: core, impl: l.impl}
}

func (l *logger) With(keysAndValues ...interface{}) Logger {
	return &logger{core: l.core, impl: l.impl.With(keysAndValues...)}
}

func (l *logger) Clone() Logger {
	return &logger{core: l.core, impl: l.impl.With()}
}

func (l *logger) Close() error {
	return l.impl.Sync()
}
