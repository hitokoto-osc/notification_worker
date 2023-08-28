package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/errors"
	"runtime"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
)

var ProducerMapping = map[string]string{}

// getProducer get rabbitmq producer
func getProducer(instant *rabbitmq.Instance, exchangeName, queueName, routingKey string) (*rabbitmq.Producer, error) {
	uuid, ok := ProducerMapping[routingKey]
	if ok {
		producer, ok := instant.GetProducer(uuid)
		if !ok {
			return nil, errors.WithStack(errors.New("can't find specific producer, uuid:" + uuid))
		}
		return producer, nil
	}
	producer, err := instant.RegisterProducer(rabbitmq.ProducerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    exchangeName,
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    queueName,
			Durable: true,
		},
		PublishingOptions: rabbitmq.PublishingOptions{
			RoutingKey: func() string {
				if routingKey == "" {
					return exchangeName + "." + queueName
				} else {
					return routingKey
				}
			}(),
		},
	})
	if err != nil {
		return nil, err
	}
	ProducerMapping[producer.GetRoutingKey()] = producer.UUID
	return producer, nil
}

func checkXDeathCount(xDeath []interface{}) int64 {
	count := int64(0)
	for _, v := range xDeath {
		v := v.(amqp.Table)
		c, o := v["count"]
		log.Debug(c)
		log.Debug(v)
		if !o {
			// TODO: 未来解决，理论上不可能
			log.Warn("[event.hitokotoFailedMessageCollector] checkXDeathCount: unexpected behavior")
			log.Warn(xDeath)
		}
		count += c.(int64)
	}
	return count
}

func wrapperHeader(header amqp.Table, body []byte) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"header": header,
		"body":   string(body),
	})
}

// HitokotoFailedMessageCollectEvent 处理通知死信
func HitokotoFailedMessageCollectEvent(instant *rabbitmq.Instance) *rabbitmq.ConsumerRegisterOptions {
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
		CallFunc: func(ctx context.Context, delivery amqp.Delivery) (err error) {
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
			log.Debugf("[RabbitMQ.Producer.FailedMessageCollector] Headers: %+v", delivery.Headers)
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
			defer func(producer *rabbitmq.Producer) {
				e := producer.Shutdown()
				if e != nil {
					log.Errorf("[RabbitMQ.Producer.FailedMessageCollector] shutdown producer failed: %v", e)
				}
			}(producer)
			if count := checkXDeathCount(XDeath.([]interface{})); count <= 5 {
				log.Debugf("[RabbitMQ.Producer.FailedMessageCollector] 当前错误计数：%v，尝试重新投递... ", count)
				time.Sleep(1 * time.Second) // 暂停 1 s
				if err = producer.Publish(ctx, amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					Headers:      delivery.Headers,
					Body:         delivery.Body,
				}); err != nil {
					return errors.WithMessagef(err, "[RabbitMQ.Producer.FailedMessageCollector] publish original queue (%v) failed.", fmt.Sprintf("%v.%v", OriginalExchangeName, OriginalQueueName))
				}
				log.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重新投递成功")
			} else {
				log.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重试次数过多，投递死信桶。")
				// 丢到死信桶队列（无法恢复）
				producer, err = getProducer(instant, "notification_failed", "notification_failed_can", "notification_failed.notification_failed_can")
				if err != nil {
					return err
				}
				defer func() {
					e := producer.Shutdown()
					if e != nil {
						log.Errorf("[RabbitMQ.Producer.FailedMessageCollector] shutdown producer failed: %v", e)
					}
				}()
				var body []byte
				body, err = wrapperHeader(delivery.Headers, delivery.Body)
				if err != nil {
					return err
				}
				if err = producer.Publish(ctx, amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					Headers:      delivery.Headers,
					Body:         body,
				}); err != nil {
					return errors.WithMessage(err, "[RabbitMQ.Producer.FailedMessageCollector] publish can queue failed.")
				}
				log.Debug("[RabbitMQ.Producer.FailedMessageCollector] 投递成功.")
			}
			return
		},
	}
}
