package main

import (
	"os"
	"runtime"
	"source.hitokoto.cn/hitokoto/notification-worker/src/aliyun/directmail"

	// 项目内文件
	"source.hitokoto.cn/hitokoto/notification-worker/src/config"
	"source.hitokoto.cn/hitokoto/notification-worker/src/event"

	// 外部依赖
	log "github.com/sirupsen/logrus"
	// "github.com/streadway/amqp"
)

// 程序信息
var (
	DEBUG   = true
)

func initLogger() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	if DEBUG { // 内编
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func init() {
	initLogger()
	config.Init()
	// 设置生产日记级别
	if config.Debug() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	// 初始化阿里云 SDK
	directmail.InitAliyunDirectMail()
}

func main() {
	log.Infoln("服务已初始化，开始核心服务。程序版本：" + Version + "，构建于 " + runtime.Version())
	go event.InitRabbitMQEvent()
	select {} // 堵塞方法
}
