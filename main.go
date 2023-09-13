package main

import (
	"flag"
	"fmt"
	"github.com/hitokoto-osc/notification-worker/logging"
	"os"
	"runtime"

	"github.com/hitokoto-osc/notification-worker/aliyun/directmail"

	// 项目内文件
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/event"
)

// 程序信息
var (
	Version    = "development"
	BuildTag   = "Unknown"
	BuildTime  = "Unknown"
	CommitTime = "Unknown"
	DEBUG      = true
	v          bool
	c          string
)

func init() {
	flag.BoolVar(&v, "v", false, "查看版本信息")
	flag.StringVar(&c, "c", "", "设定配置文件")
	flag.Parse()
	if v {
		fmt.Printf("NotificationWorker ©2023 MoeTeam All Rights Reserved. \n当前版本: %s \n版控哈希: %s\n提交时间：%s\n编译时间：%s\n", Version, BuildTag, CommitTime, BuildTime)
		os.Exit(0)
	}
	logging.InitDefaultLogger(DEBUG)
	config.Init(c)
	// 设置生产日记级别
	if config.Debug() {
		DEBUG = true
	} else {
		DEBUG = false
	}
	logging.SetLogDebugConfig(DEBUG)
	// 初始化阿里云 SDK
	directmail.InitAliyunDirectMail()
}

func main() {

	logger := logging.GetLogger()
	logger.Info("服务已初始化，开始核心服务。程序版本：" + Version + "，构建于 " + runtime.Version() + "。 版控哈希：" + BuildTag)
	go event.InitRabbitMQEvent()
	select {} // 堵塞方法
}
