package event

import (
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"source.hitokoto.cn/hitokoto/notification-worker/config"
	"source.hitokoto.cn/hitokoto/notification-worker/event/notification"
	"source.hitokoto.cn/hitokoto/notification-worker/logging"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func InitRabbitMQEvent() {
	logger := logging.GetLogger()
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
	logger.Info("开始注册消息接收器...")
	instance.RegisterConsumerConfig(*notification.HitokotoFailedMessageCollectEvent(instance))
	instance.RegisterConsumerConfig(*notification.HitokotoFailedMessageCanEvent())
	instance.RegisterConsumerConfig(*notification.HitokotoAppendedEvent())
	instance.RegisterConsumerConfig(*notification.HitokotoReviewedEvent())
	instance.RegisterConsumerConfig(*notification.HitokotoPollCreatedEvent())
	instance.RegisterConsumerConfig(*notification.HitokotoPollFinishedEvent())
	instance.RegisterConsumerConfig(*notification.HitokotoPollDailyReportEvent())
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
