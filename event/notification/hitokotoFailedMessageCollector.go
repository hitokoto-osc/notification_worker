package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"math"
	"source.hitokoto.cn/hitokoto/notification-worker/logging"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
)

var ProducerMapping = make(map[string]string)

// getProducer get rabbitmq producer
func getProducer(ctx context.Context, instance *rabbitmq.Instance, exchangeName, queueName, routingKey string) (*rabbitmq.Producer, error) {
	logger := logging.WithContext(ctx)
	if routingKey == "" {
		routingKey = exchangeName + "." + queueName
	}
	uuid, ok := ProducerMapping[routingKey]
	if ok {
		var producer *rabbitmq.Producer
		producer, ok = instance.GetProducer(uuid)
		if ok {
			return producer, nil
		}
		logger.Warn("[event.hitokotoFailedMessageCollector] producer not found, try to recreate it.",
			zap.String("uuid", uuid),
			zap.String("routingKey", routingKey),
			zap.String("exchangeName", exchangeName),
			zap.String("queueName", queueName),
		)
	}
	producer, err := instance.RegisterProducer(rabbitmq.ProducerRegisterOptions{
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
			RoutingKey: routingKey,
		},
	})
	if err != nil {
		return nil, err
	}
	ProducerMapping[producer.GetRoutingKey()] = producer.UUID
	return producer, nil
}

func checkXDeathCount(ctx context.Context, xDeath []interface{}) int64 {
	logger := logging.WithContext(ctx)
	defer logger.Sync()
	count := int64(0)
	for _, v := range xDeath {
		table := v.(amqp.Table)
		c, o := table["count"]
		logger.Debug("c, v", zap.Any("c", c), zap.Any("v", v))
		if !o {
			// TODO: 未来解决，理论上不可能
			logger.Warn("[event.hitokotoFailedMessageCollector] checkXDeathCount: unexpected behavior",
				zap.Any("xDeath", xDeath),
			)
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
func HitokotoFailedMessageCollectEvent(instance *rabbitmq.Instance) *rabbitmq.ConsumerRegisterOptions {
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
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			//defer func() {
			//	e := recover()
			//	if e != nil {
			//		switch v := e.(type) {
			//		case string:
			//			err = errors.WithStack(errors.New(v))
			//		case error:
			//			err = v
			//		case runtime.Error:
			//			err = v
			//		default:
			//			logger.Error("[RabbitMQ.Producer.FailedMessageCollector] unknown error: ", zap.Any("error", e))
			//			err = errors.New("unknown error")
			//		}
			//	}
			//}()
			logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] received a new message: ",
				zap.String("headers", fmt.Sprintf("%+v", delivery.Headers)),
				zap.ByteString("body", delivery.Body),
			)
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
			producer, err = getProducer(
				ctx,
				instance,
				OriginalExchangeName.(string),
				OriginalQueueName.(string),
				"",
			)
			if err != nil {
				return err
			}
			if count := checkXDeathCount(ctx, XDeath.([]interface{})); count <= 5 {
				duration := time.Second * time.Duration(math.Pow(4, float64(count)))
				logger.Sugar().Debugf("[RabbitMQ.Producer.FailedMessageCollector] 当前错误计数：%v，等待 %d 秒后，尝试重新投递... ", count, duration/time.Second)
				time.Sleep(duration)
				if err = producer.Publish(ctx, amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					Headers:      delivery.Headers,
					Body:         delivery.Body,
				}); err != nil {
					return errors.WithMessagef(err, "[RabbitMQ.Producer.FailedMessageCollector] publish original queue (%v) failed.", fmt.Sprintf("%v.%v", OriginalExchangeName, OriginalQueueName))
				}
				logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重新投递成功")
			} else {
				logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重试次数过多，投递死信桶。")
				// 丢到死信桶队列（无法恢复）
				producer, err = getProducer(
					ctx,
					instance,
					"notification_failed",
					"notification_failed_can",
					"notification_failed.notification_failed_can",
				)
				if err != nil {
					return err
				}
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
				logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 投递成功.")
			}
			return
		},
	}
}
