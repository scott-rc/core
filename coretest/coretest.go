package coretest

import (
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/scott-rc/core"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	Config                   core.Configuration
	ResolverContextDecorator core.ResolverContextDecorator
}

func NewCore(t *testing.T, opts Options) *core.Core {
	require.NotNil(t, opts.Config)
	require.NotNil(t, opts.ResolverContextDecorator)

	id, err := gonanoid.Nanoid()
	require.NoError(t, err)

	c := &core.Core{
		Id:         id,
		Logger:     testLogger{t},
		Config:     opts.Config,
		Request:    httptest.NewRequest("POST", "/api", nil),
		Operations: []string{},
		Validate:   validator.New(),

		// set later
		Context: nil,
		Session: nil,
		Db:      nil,
	}
	c.Context = opts.ResolverContextDecorator(context.WithValue(context.Background(), core.ContextKey, c))
	require.NoError(t, c.StartSession())

	if opts.Config.CoreConfig().Database.Test.Driver != "" {
		db, err := sql.Open(opts.Config.CoreConfig().Database.Test.Driver, opts.Config.CoreConfig().Database.Test.DataSourceName())
		require.NoError(t, err)
		c.Db = db
	}

	return c
}

type testLogger struct {
	impl *testing.T
}

func (t testLogger) Debug(msg string, keysAndValues ...interface{}) {
	t.impl.Log(append([]interface{}{msg}, keysAndValues...)...)
}

func (t testLogger) Info(msg string, keysAndValues ...interface{}) {
	t.impl.Log(append([]interface{}{msg}, keysAndValues...)...)
}

func (t testLogger) Warn(msg string, keysAndValues ...interface{}) {
	t.impl.Log(append([]interface{}{msg}, keysAndValues...)...)
}

func (t testLogger) Error(msg string, keysAndValues ...interface{}) {
	t.impl.Error(append([]interface{}{msg}, keysAndValues...)...)
}

func (t testLogger) DPanic(msg string, keysAndValues ...interface{}) {
	t.impl.Log(append([]interface{}{msg}, keysAndValues...)...)
	panic(msg)
}

func (t testLogger) Panic(msg string, keysAndValues ...interface{}) {
	t.impl.Log(append([]interface{}{msg}, keysAndValues...)...)
	panic(msg)
}

func (t testLogger) Fatal(msg string, keysAndValues ...interface{}) {
	t.impl.Fatal(append([]interface{}{msg}, keysAndValues...)...)
}

func (t testLogger) Log(level zapcore.Level, msg string, keysAndValues ...interface{}) {
	switch level {
	case zapcore.DebugLevel:
		t.Debug(msg, keysAndValues...)
	case zapcore.InfoLevel:
		t.Info(msg, keysAndValues...)
	case zapcore.WarnLevel:
		t.Warn(msg, keysAndValues...)
	case zapcore.ErrorLevel:
		t.Error(msg, keysAndValues...)
	case zapcore.DPanicLevel:
		t.DPanic(msg, keysAndValues...)
	case zapcore.PanicLevel:
		t.Panic(msg, keysAndValues...)
	case zapcore.FatalLevel:
		t.Fatal(msg, keysAndValues...)
	}
}

func (t testLogger) WithCore(*core.Core) core.Logger {
	return t
}

func (t testLogger) With(...interface{}) core.Logger {
	return t
}

func (t testLogger) Clone() core.Logger {
	return t
}

func (t testLogger) Close() error {
	return nil
}

func (t testLogger) Printf(format string, v ...interface{}) {
	content := fmt.Sprintf(format, v...)
	t.Info(content)
}

func (t testLogger) Verbose() bool {
	return true
}
