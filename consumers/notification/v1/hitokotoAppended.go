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

// HitokotoAppendedEvent 返回处理一言成功添加事件的配置
func HitokotoAppendedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_appended",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_appended",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoAppendedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_appended] 收到消息: %v  \n", zap.ByteString("body", delivery.Body))
			message, err := validator.UnmarshalV[model.HitokotoAppendedMessage](ctx, delivery.Body)
			if err != nil {
				return errors.Wrap(err, "解析消息失败")
			}
			html, err := django.RenderTemplate("email/hitokoto_appended", django.Context{
				"username":   message.Creator,
				"created_at": message.CreatedAt.Format("Y-m-d H:i:s"),
				"hitokoto":   message.Hitokoto,
				"from":       message.From,
				"from_who":   message.FromWho,
				"type":       formatter.FormatHitokotoType(message.Type),
			})
			if err != nil {
				return errors.Wrap(err, "渲染模板失败")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！已经成功收到您提交的句子了！",
					Body:    html,
				},
			})
			return err
		},
	}
}
