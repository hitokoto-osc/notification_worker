package logging

import (
	"context"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	hcontext "source.hitokoto.cn/hitokoto/notification-worker/context"
)

const loggerKey = "logger"

var logger *zap.Logger

func InitDefaultLogger(debug bool) {
	SetLogDebugConfig(debug)
}

func SetLogDebugConfig(debug bool) {
	var (
		c   zap.Config
		err error
	)
	if debug {
		c = zap.NewDevelopmentConfig()
	} else {
		c = zap.NewProductionConfig()
	}
	// 统一配置
	c.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	c.OutputPaths = []string{"stdout"}
	c.ErrorOutputPaths = []string{"stderr"}
	logger, err = c.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		panic(errors.WithMessage(err, "logger construction failed"))
	}
	defer logger.Sync()
	logger.Debug("logger construction succeeded")
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
