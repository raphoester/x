package xlog

import (
	"fmt"
	"testing"

	"github.com/raphoester/x/xlog/lf"
)

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

type TestLogger struct {
	t      *testing.T
	fields []lf.Field
}

func (t *TestLogger) logMsg(level string, message string, fields ...lf.Field) {
	t.t.Helper()
	for _, field := range t.fields {
		message = fmt.Sprintf("%s %s=%v", message, field.Key(), field.Value())
	}

	for _, field := range fields {
		message = fmt.Sprintf("%s %s=%v", message, field.Key(), field.Value())
	}

	t.t.Logf("%s: %s", level, message)
}

func (t *TestLogger) Debug(message string, fields ...lf.Field) {
	t.t.Helper()
	t.logMsg("DEBUG", message, fields...)
}

func (t *TestLogger) Info(message string, fields ...lf.Field) {
	t.t.Helper()
	t.logMsg("INFO", message, fields...)
}

func (t *TestLogger) Warning(message string, fields ...lf.Field) {
	t.t.Helper()
	t.logMsg("WARNING", message, fields...)
}

func (t *TestLogger) Error(message string, fields ...lf.Field) {
	t.t.Helper()
	t.logMsg("ERROR", message, fields...)
}

func (t *TestLogger) WithFields(fields ...lf.Field) Logger {
	cp := *t
	cp.fields = append(cp.fields, fields...)
	return &cp
}

func (t *TestLogger) WithCallerSkip(int) Logger {
	return t
}
