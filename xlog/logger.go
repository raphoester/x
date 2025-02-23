package xlog

import "github.com/raphoester/x/xlog/lf"

type Logger interface {
	Debug(message string, fields ...lf.Field)
	Info(message string, fields ...lf.Field)
	Warning(message string, fields ...lf.Field)
	Error(message string, fields ...lf.Field)
	WithFields(fields ...lf.Field) Logger
	WithCallerSkip(skip int) Logger
}
