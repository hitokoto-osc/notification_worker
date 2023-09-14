package logging

import (
	"context"
	hcontext "github.com/hitokoto-osc/notification-worker/context"
	"go.uber.org/zap"
)

const loggerKey = "logger"

var logger *zap.Logger

func setZapGlobalLogger() {
	zap.ReplaceGlobals(logger)
}

func GetLogger() *zap.Logger {
	return logger
}

func NewContext(ctx context.Context, fields ...zap.Field) {
	if ctx == nil {
		logger.Panic("context is nil")
	}
	hctx, ok := ctx.(hcontext.IContext)
	if !ok {
		logger.Panic("context is not hcontext.IContext")
	}
	hctx.Set(loggerKey, WithContext(ctx).With(fields...))
}

func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return logger
	}
	hctx, ok := ctx.(hcontext.IContext)
	if !ok {
		return logger
	}
	l := hctx.Get(loggerKey)
	ctxLogger, ok := l.(*zap.Logger)
	if !ok {
		return logger
	}
	return ctxLogger
}
