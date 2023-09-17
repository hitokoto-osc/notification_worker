package main

import (
	"github.com/hitokoto-osc/notification-worker/flags"
	"go.uber.org/zap"
	"runtime"
	// 项目内文件
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/consumers"
)

func init() {
	config.Parse()
	flags.Do()
}

func main() {
	zap.L().Info(
		"服务已初始化，开始核心服务。",
		zap.String("version", config.Version),
		zap.String("build_tag", config.BuildTag),
		zap.String("commit_time", config.CommitTime),
		zap.String("build_time", config.BuildTime),
		zap.String("runtime", runtime.Version()),
	)
	consumers.Run()
}
