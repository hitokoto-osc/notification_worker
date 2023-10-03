package v1

import (
	"github.com/cockroachdb/errors"
	"github.com/hitokoto-osc/notification-worker/consumers/notification/v1/internal/model"
	"github.com/hitokoto-osc/notification-worker/consumers/provider"
	"github.com/hitokoto-osc/notification-worker/django"
	"github.com/hitokoto-osc/notification-worker/logging"
	"github.com/hitokoto-osc/notification-worker/mail"
	"github.com/hitokoto-osc/notification-worker/mail/mailer"
	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	"github.com/hitokoto-osc/notification-worker/utils/formatter"
	"github.com/hitokoto-osc/notification-worker/utils/validator"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func init() {
	provider.Register(HitokotoAppendedEvent())
}

// HitokotoPollCreatedEvent 处理一言成功添加事件
func HitokotoPollCreatedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_poll_created",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_poll_created",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoPollCreatedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_poll_created]收到消息：", zap.ByteString("body", delivery.Body))
			message, err := validator.UnmarshalV[model.PollCreatedMessage](ctx, delivery.Body)
			if err != nil {
				return errors.Wrap(err, "解析消息失败")
			}
			html, err := django.RenderTemplate("email/poll_created", django.Context{
				"username":   message.Username,
				"created_at": message.CreatedAt.Format("Y-m-d H:i:s"),
				"poll_id":    message.ID,
				"hitokoto":   message.Hitokoto,
				"from":       message.From,
				"from_who":   message.FromWho,
				"type":       formatter.FormatHitokotoType(message.Type),
				"creator":    message.Creator,
			})
			if err != nil {
				return errors.Wrap(err, "无法渲染模板")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！新的野生投票菌出现了！",
					Body:    html,
				},
			})
			return err
		},
	}
}
