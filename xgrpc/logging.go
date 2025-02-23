package xgrpc

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

// InterceptorLogger adapts zap logger to interceptor logger.
// This code is simple enough to be copied and not imported.
//
// Taken from https://github.com/grpc-ecosystem/go-grpc-middleware/blob/main/interceptors/logging/examples/zap/example_test.go
func InterceptorLogger(l xlog.Logger) logging.Logger {
	return logging.LoggerFunc(
		func(
			ctx context.Context,
			lvl logging.Level,
			msg string,
			fields ...any,
		) {
			f := make([]lf.Field, 0, len(fields)/2)
			for i := 0; i < len(fields); i += 2 {
				key := fields[i]
				value := fields[i+1]

				switch v := value.(type) {
				case string:
					f = append(f, lf.String(key.(string), v))
				case int:
					f = append(f, lf.Int(key.(string), v))
				case bool:
					f = append(f, lf.Bool(key.(string), v))
				default:
					f = append(f, lf.Any(key.(string), v))
				}
			}

			switch lvl {
			case logging.LevelDebug:
				l.Debug(msg, f...)
			case logging.LevelInfo:
				l.Info(msg, f...)
			case logging.LevelWarn:
				l.Warning(msg, f...)
			case logging.LevelError:
				l.Error(msg, f...)
			default:
				l.Error("unknown log level", lf.Int("level", int(lvl)))
				l.Info(msg, f...)
			}
		})
}
