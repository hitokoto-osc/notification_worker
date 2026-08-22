package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/hitokoto-osc/notification-worker/consumers/provider"
	"github.com/hitokoto-osc/notification-worker/logging"
	"go.uber.org/zap"
	"math"
	"time"

	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func init() {
	provider.Register(HitokotoFailedMessageCollectEvent())
}

// maxRedeliveryCount 是消息被重新投递回原队列的最大次数，超出则进入死信桶。
const maxRedeliveryCount = 5

// checkXDeathCount 累计 x-death 头中记录的死信次数。
// 头部由 broker 写入，但仍需逐项校验类型——异常结构不应让 goroutine panic。
func checkXDeathCount(ctx context.Context, xDeath []interface{}) (int64, error) {
	logger := logging.WithContext(ctx)
	defer logger.Sync()
	count := int64(0)
	for i, v := range xDeath {
		table, ok := v.(amqp.Table)
		if !ok {
			return 0, errors.Newf("x-death[%d] 类型异常：%T", i, v)
		}
		raw, ok := table["count"]
		if !ok {
			return 0, errors.Newf("x-death[%d] 缺少 count 字段", i)
		}
		var item int64
		switch c := raw.(type) {
		case int64:
			item = c
		case int32:
			item = int64(c)
		case int:
			item = int64(c)
		default:
			return 0, errors.Newf("x-death[%d].count 类型异常：%T", i, raw)
		}
		if item < 0 {
			return 0, errors.Newf("x-death[%d].count 为负数：%d", i, item)
		}
		logger.Debug("x-death entry", zap.Int("index", i), zap.Int64("count", item))
		count += item
	}
	return count, nil
}

// headerString 读取一个必须存在且为非空字符串的头部字段。
func headerString(headers amqp.Table, key string) (string, error) {
	raw, ok := headers[key]
	if !ok {
		return "", errors.Newf("%s is missing", key)
	}
	value, ok := raw.(string)
	if !ok || value == "" {
		return "", errors.Newf("%s 类型或取值异常：%T", key, raw)
	}
	return value, nil
}

func wrapperHeader(header amqp.Table, body []byte) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"header": header,
		"body":   string(body),
	})
}

// publishToCan 把无法恢复的消息投递到死信桶队列。
func publishToCan(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
	producer, err := ctx.GetProducer(
		"notification_failed",
		"notification_failed_can",
		"notification_failed.notification_failed_can",
	)
	if err != nil {
		return err
	}
	body, err := wrapperHeader(delivery.Headers, delivery.Body)
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
	return nil
}

// HitokotoFailedMessageCollectEvent 处理通知死信
func HitokotoFailedMessageCollectEvent() *rabbitmq.ConsumerRegisterOptions {
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
			Tag: "HitokotoFailedMessageCollectWorker",
			// 本 handler 会阻塞式 sleep 最长 4^5 秒，需要限制并发在途消息数。
			Prefetch:   8,
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] received a new message: ",
				zap.String("headers", fmt.Sprintf("%+v", delivery.Headers)),
				zap.ByteString("body", delivery.Body),
			)

			count, err := parseRedeliveryCount(ctx, delivery)
			if err != nil {
				// 本队列的死信路由指回自身，返回错误会导致消息被无限热循环重投递。
				// 头部无法解析的消息已不可恢复，直接送入死信桶。
				logger.Error("[RabbitMQ.Producer.FailedMessageCollector] 死信头部无法解析，投递死信桶。",
					zap.Error(err),
					zap.String("headers", fmt.Sprintf("%+v", delivery.Headers)),
				)
				return publishToCan(ctx, delivery)
			}

			if count > maxRedeliveryCount {
				logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重试次数过多，投递死信桶。")
				if err = publishToCan(ctx, delivery); err != nil {
					return err
				}
				logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 投递成功.")
				return nil
			}

			originalExchange, err := headerString(delivery.Headers, "x-first-death-exchange")
			if err != nil {
				return err
			}
			originalQueue, err := headerString(delivery.Headers, "x-first-death-queue")
			if err != nil {
				return err
			}
			producer, err := ctx.GetProducer(originalExchange, originalQueue, "")
			if err != nil {
				return err
			}

			duration := time.Second * time.Duration(math.Pow(4, float64(count)))
			logger.Sugar().Debugf("[RabbitMQ.Producer.FailedMessageCollector] 当前错误计数：%v，等待 %d 秒后，尝试重新投递... ", count, duration/time.Second)
			time.Sleep(duration)
			if err = producer.Publish(ctx, amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				Headers:      delivery.Headers,
				Body:         delivery.Body,
			}); err != nil {
				return errors.WithMessagef(err, "[RabbitMQ.Producer.FailedMessageCollector] publish original queue (%v) failed.", fmt.Sprintf("%s.%s", originalExchange, originalQueue))
			}
			logger.Debug("[RabbitMQ.Producer.FailedMessageCollector] 重新投递成功")
			return nil
		},
	}
}

// parseRedeliveryCount 从 delivery 头部解析累计死信次数。
func parseRedeliveryCount(ctx rabbitmq.Ctx, delivery amqp.Delivery) (int64, error) {
	raw, ok := delivery.Headers["x-death"]
	if !ok {
		return 0, errors.New("x-death is missing")
	}
	entries, ok := raw.([]interface{})
	if !ok {
		return 0, errors.Newf("x-death 类型异常：%T", raw)
	}
	return checkXDeathCount(ctx, entries)
}
