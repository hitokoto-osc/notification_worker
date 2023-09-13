package notification

import (
	"context"
	"github.com/hitokoto-osc/notification-worker/logging"
	"go.uber.org/zap"

	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// HitokotoFailedMessageCanEvent 处理不可恢复通知死信 —— 发给管理员
func HitokotoFailedMessageCanEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification_failed",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "notification_failed_can",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector", // 万一发送失败了还是丢死信收集器咯
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification_failed.notification_failed_can",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoFailedMessageCollectWorker",
			AckByError: true,
		},
		CallFunc: func(ctx context.Context, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Error("[RabbitMQ.Producer.FailedMessageCan] 收到死信：", zap.ByteString("body", delivery.Body))
			// TODO: 修复死信循环的问题
			//			html := fmt.Sprintf(`<h1>您好，a632079。</h1>
			//<p>系统遇到了一封无法处理的死信，以下为详细信息：</p>
			//<pre>
			//	<code>
			//%v
			//	</code>
			//</pre>
			//`, string(delivery.Body))

			// err := directmail.SingleSendMail(ctx, "a632079@qq.com", "[一言告警] 出现不可恢复的死信！", html, true)
			return nil
		},
	}
}
