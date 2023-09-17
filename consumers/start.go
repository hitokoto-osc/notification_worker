package consumers

import (
	"github.com/hitokoto-osc/notification-worker/config"
	_ "github.com/hitokoto-osc/notification-worker/consumers/notification"
	"github.com/hitokoto-osc/notification-worker/consumers/provider"
	"github.com/hitokoto-osc/notification-worker/logging"
	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func InitRabbitMQEvent() {
	logger := logging.GetLogger()
	defer logger.Sync()
	logger.Info("注册消息队列接收器...")
	c := &config.RabbitMQ{}
	logger.Info((&rabbitmq.Config{
		Host:     c.Host(),
		Port:     c.Port(),
		Username: c.User(),
		Password: c.Pass(),
		Vhost:    c.VHost(),
	}).URI())
	instance := rabbitmq.New(&rabbitmq.Config{
		Host:     c.Host(),
		Port:     c.Port(),
		Username: c.User(),
		Password: c.Pass(),
		Vhost:    c.VHost(),
	}, logger.Sugar())
	if err := instance.Init(); err != nil {
		logger.Fatal("无法启动实例", zap.Error(err))
	}

	// 注册接收器
	options := provider.Get()
	logger.Info("开始注册消息接收器...", zap.Any("options", options))
	for _, v := range options {
		instance.RegisterConsumerConfig(*v)
	}
	handleErr(instance.ConsumerSubscribe())
	logger.Info("已注册消息接收器，开始处理消息。")
	select {}
}

func handleErr(e error) {
	logger := logging.GetLogger()
	if e != nil {
		logger.Fatal("无法注册消费者", zap.Error(e))
	}
}
