package logging

import (
	"context"
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

type Ctx interface {
	Get(key string) any
	Set(key string, value any)
}

func NewContext(ctx context.Context, fields ...zap.Field) {
	if ctx == nil {
		logger.Panic("context is nil")
	}
	c, ok := ctx.(Ctx)
	if !ok {
		logger.Panic("context is not Ctx")
	}
	c.Set(loggerKey, WithContext(ctx).With(fields...))
}

func WithContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return logger
	}
	c, ok := ctx.(Ctx)
	if !ok {
		return logger
	}
	l := c.Get(loggerKey)
	ctxLogger, ok := l.(*zap.Logger)
	if !ok {
		return logger
	}
	return ctxLogger
}
