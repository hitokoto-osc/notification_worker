package event

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"source.hitokoto.cn/hitokoto/notification-worker/config"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
	"strconv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func InitRabbitMQEvent() {
	log.Info("注册消息队列接收器...")
	rabbitMQConfig := &config.RabbitMQ{}
	rabbitMQClient := &rabbitmq.AmqpClient{}
	connectionStr := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitMQConfig.User(), rabbitMQConfig.Pass(), rabbitMQConfig.Host(), strconv.Itoa(rabbitMQConfig.Port()))
	rabbitMQClient.ConnectToBroker(connectionStr)

	consumer := &rabbitmq.Consumer{
		Client:    rabbitMQClient,
		Receivers: []*rabbitmq.Receiver{},
	}

	// 注册接收器
	consumer.Add((&hitokotoAppendedEvent{}).Receiver())
	consumer.Add((&hitokotoReviewedEvent{}).Receiver())
	consumer.Add((&hitokotoPollCreatedEvent{}).Receiver())
	consumer.Add((&hitokotoPollFinishedEvent{}).Receiver())
	consumer.Add((&hitokotoPollDailyReportEvent{}).Receiver())
	log.Info("已成功注册消息接收器，开始处理消息。")

	consumer.Subscribe()

	// TODO: 加入 Publish 接口

	select {}
}
