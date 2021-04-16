package event

import (
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"source.hitokoto.cn/hitokoto/notification-worker/config"
	"source.hitokoto.cn/hitokoto/notification-worker/event/notification"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func InitRabbitMQEvent() {
	log.Info("注册消息队列接收器...")
	c := &config.RabbitMQ{}
	instant := rabbitmq.New(&rabbitmq.Config{
		Host:     c.Host(),
		Port:     c.Port(),
		Username: c.User(),
		Password: c.Pass(),
		Vhost:    c.VHost(),
	}, log.StandardLogger())
	if err := instant.Init(); err != nil {
		log.Fatal(err)
	}

	// 注册接收器
	log.Info("开始注册消息接收器...")
	instant.RegisterConsumerConfig(*notification.HitokotoFailedMessageCanEvent())
	instant.RegisterConsumerConfig(*notification.HitokotoFailedMessageCollectEvent(instant))
	instant.RegisterConsumerConfig(*notification.HitokotoAppendedEvent())
	instant.RegisterConsumerConfig(*notification.HitokotoReviewedEvent())
	instant.RegisterConsumerConfig(*notification.HitokotoPollCreatedEvent())
	instant.RegisterConsumerConfig(*notification.HitokotoPollFinishedEvent())
	instant.RegisterConsumerConfig(*notification.HitokotoPollDailyReportEvent())
	handleErr(instant.ConsumerSubscribe())
	log.Info("已注册消息接收器，开始处理消息。")
	select {}
}

func handleErr (e error) {
	if e != nil {
		log.Fatal(e)
	}
}
