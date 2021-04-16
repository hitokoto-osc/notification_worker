package notification

import (
	"encoding/json"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"runtime"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
	"sync"
	"time"
)

var ProducerMapping = map[string]string{}

// getProducer get rabbitmq producer
func getProducer(instant *rabbitmq.RabbitMQInstant, exchangeName, queueName, routingKey string) (*rabbitmq.Producer, error) {
	uuid, ok := ProducerMapping[routingKey]
	if ok {
		producer, ok := instant.GetProducer(uuid)
		if !ok {
			return nil, errors.WithStack(errors.New("can't find specific producer, uuid:" + uuid))
		}
		return producer, nil
	}
	return instant.RegisterProducer(rabbitmq.ProducerRegisterOptions{
		Exchange:    rabbitmq.Exchange{
			Name: exchangeName,
			Type: "direct",
			Durable: true,
		},
		Queue:             rabbitmq.Queue{
			Name: queueName,
			Durable: true,
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: func () string {
				if routingKey == "" {
					return exchangeName + "." + queueName
				} else {
					return routingKey
				}}(),
		},
		PublishingOptions: rabbitmq.PublishingOptions{},
	})
}

func checkXDeathCount (xDeath []amqp.Table) (int) {
	count := 0
	for _, v := range xDeath {
		c, o := v["count"]
		if !o {
			// TODO: 未来解决，理论上不可能
			log.Warn("[event.hitokotoFailedMessageCollector] checkXDeathCount: unexpected behavior")
			log.Warn(xDeath)
		}
		count += c.(int)
	}
	return count
}

func wrapperHeader(header amqp.Table, body []byte) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"header": header,
		"body": string(body),
	})
}

// HitokotoFailedMessageCollectEvent 处理通知死信
func HitokotoFailedMessageCollectEvent(instant *rabbitmq.RabbitMQInstant) *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification_failed",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "notification_failed_collector",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification_failed.notification_failed_collector",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoFailedMessageCollectWorker",
			AckByError: true,
		},
		CallFunc: func(delivery amqp.Delivery) (err error) {
			defer func() {
				e := recover()
				if e != nil {
					switch e := e.(type) {
					case string:
						err = errors.New(e)
					case error:
						err = e
					case runtime.Error:
						err = e
					default:
						log.Error(e)
						err = errors.New("unknown error")
					}

				}
			}()
			XDeath, ok := delivery.Headers["x-death"]
			if !ok {
				return errors.New("x-death is missing")
			}
			OriginalExchangeName, ok := delivery.Headers["x-first-death-exchange"]
			if !ok {
				return errors.New("x-first-death-exchange is missing")
			}
			OriginalQueueName, ok := delivery.Headers["x-first-death-queue"]
			if !ok {
				return errors.New("x-first-death-queue is missing")
			}
			var producer *rabbitmq.Producer
			producer, err = getProducer(instant, OriginalExchangeName.(string), OriginalQueueName.(string), "")
			if err != nil {
				return err
			}
			if count := checkXDeathCount(XDeath.([]amqp.Table)); count <= 5 {
				time.Sleep(1 * time.Second) // 暂停 1 s
				if err = producer.Publish(amqp.Publishing{
					Body: delivery.Body,
				}); err != nil {
					return err
				}
				wg := sync.WaitGroup{}
				wg.Add(1)
				producer.NotifyReturn(func(message amqp.Return) {
					if message.ReplyCode != 200 {
						err = errors.Errorf("[RabbitMQ.Producer.FailedMessageCollector] publish original queue failed. %v - %v", message.ReplyCode, message.ReplyText)
					}
					wg.Done()
				})
				wg.Wait()
			} else {
				// 丢到死信桶队列（无法恢复）
				producer, err = getProducer(instant, "notification_failed", "notification_failed_can", "notification_failed.notification_failed_can")
				if err != nil {
					return err
				}
				var body []byte
				body, err = wrapperHeader(delivery.Headers, delivery.Body)
				if err != nil {
					return err
				}
				if err = producer.Publish(amqp.Publishing{
					Body: body,
				}); err != nil {
					return err
				}
				wg := sync.WaitGroup{}
				wg.Add(1)
				producer.NotifyReturn(func(message amqp.Return) {
					if message.ReplyCode != 200 {
						err = errors.Errorf("[RabbitMQ.Producer.FailedMessageCollector] publish can queue failed. %v - %v", message.ReplyCode, message.ReplyText)
					}
					wg.Done()
				})
				wg.Wait()
			}
			return
		},
	}
}
