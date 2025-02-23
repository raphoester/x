package xlog

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"github.com/raphoester/x/xlog/lf"
	"github.com/raphoester/x/xtime"
)

type SLoggerConfig struct {
	Level    string `yaml:"level"`
	Encoding string `yaml:"encoding"`
}

func (c *SLoggerConfig) ResetToDefault() {
	c.Level = "info"
	c.Encoding = "console"
}

func NewSLogger(
	config SLoggerConfig,
	timeProvider xtime.Provider,
) *SLogger {
	var level slog.Level
	switch config.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warning", "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// doc for field rewrites :
	//https://cloud.google.com/logging/docs/agent/logging/configuration?hl=fr#special-fields

	handlerOptions := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
			}

			if a.Key == slog.MessageKey {
				a.Key = "textPayload"
			}

			return a
		},
	}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	if config.Encoding == "json" {
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	}

	return &SLogger{
		timeProvider: timeProvider,
		logger:       slog.New(handler),
	}
}

type SLogger struct {
	logger       *slog.Logger
	skipCallers  int
	timeProvider xtime.Provider
}

func (s *SLogger) WithFields(fields ...lf.Field) Logger {
	clone := s.clone()
	clone.logger = clone.logger.With(s.parseFields(fields)...)
	return clone
}

func (s *SLogger) WithCallerSkip(skip int) Logger {
	if skip <= 0 {
		return s
	}

	clone := s.clone()
	clone.skipCallers += skip
	return clone
}

func (s *SLogger) clone() *SLogger {
	slogger := *s.logger
	return &SLogger{
		logger:       &slogger,
		timeProvider: s.timeProvider,
		skipCallers:  s.skipCallers,
	}
}

func (s *SLogger) parseFields(fields []lf.Field) []any {
	ret := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		ret = append(ret, f.Key(), f.Value())
	}
	return ret
}

func (s *SLogger) log(level slog.Level, message string, fields ...lf.Field) {
	var pcs [1]uintptr
	runtime.Callers(s.skipCallers, pcs[:])

	r := slog.NewRecord(s.timeProvider.Now(), level, message, pcs[0])
	r.Add(s.parseFields(fields)...)

	_ = s.logger.Handler().Handle(context.Background(), r)
}

func (s *SLogger) Debug(message string, fields ...lf.Field) {
	s.log(slog.LevelDebug, message, fields...)
}

func (s *SLogger) Info(message string, fields ...lf.Field) {
	s.log(slog.LevelInfo, message, fields...)
}

func (s *SLogger) Warning(message string, fields ...lf.Field) {
	s.log(slog.LevelWarn, message, fields...)
}

func (s *SLogger) Error(message string, fields ...lf.Field) {
	s.log(slog.LevelError, message, fields...)
}
