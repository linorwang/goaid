package logger

import (
	"time"

	"go.uber.org/zap"
)

type ZapLogger struct {
	l *zap.Logger
}

func NewZapLogger(l *zap.Logger) *ZapLogger {
	return &ZapLogger{
		l: l,
	}
}

func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.l.Debug(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Info(msg string, args ...Field) {
	z.l.Info(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.l.Warn(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Error(msg string, args ...Field) {
	z.l.Error(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Fatal(msg string, args ...Field) {
	z.l.Fatal(msg, z.toArgs(args)...)
}

func (z *ZapLogger) Panic(msg string, args ...Field) {
	z.l.Panic(msg, z.toArgs(args)...)
}

func (z *ZapLogger) With(args ...Field) Logger {
	zapArgs := z.toArgs(args)
	return &ZapLogger{
		l: z.l.With(zapArgs...),
	}
}

func (z *ZapLogger) Sync() error {
	return z.l.Sync()
}

func (z *ZapLogger) toArgs(args []Field) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	res := make([]zap.Field, 0, len(args))
	for _, arg := range args {
		switch v := arg.Value.(type) {
		case string:
			res = append(res, zap.String(arg.Key, v))
		case int:
			res = append(res, zap.Int(arg.Key, v))
		case int64:
			res = append(res, zap.Int64(arg.Key, v))
		case int32:
			res = append(res, zap.Int32(arg.Key, v))
		case float64:
			res = append(res, zap.Float64(arg.Key, v))
		case bool:
			res = append(res, zap.Bool(arg.Key, v))
		case time.Time:
			res = append(res, zap.Time(arg.Key, v))
		case time.Duration:
			res = append(res, zap.Duration(arg.Key, v))
		case error:
			res = append(res, zap.Error(v))
		default:
			res = append(res, zap.Any(arg.Key, v))
		}
	}
	return res
}
