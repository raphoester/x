package xlog

import "github.com/raphoester/x/xlog/lf"

type NopLogger struct{}

func (n NopLogger) Debug(_ string, _ ...lf.Field) {}

func (n NopLogger) Info(_ string, _ ...lf.Field) {}

func (n NopLogger) Warning(_ string, _ ...lf.Field) {}

func (n NopLogger) Error(_ string, _ ...lf.Field) {}

func (n NopLogger) WithFields(_ ...lf.Field) Logger { return n }

func (n NopLogger) WithCallerSkip(_ int) Logger { return n }
