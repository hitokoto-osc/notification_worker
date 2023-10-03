package v1

import (
	"github.com/cockroachdb/errors"
	"github.com/golang-module/carbon/v2"
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
	"strconv"
)

func init() {
	provider.Register(HitokotoPollFinishedEvent())
}

// HitokotoPollFinishedEvent 一言投票完成事件
func HitokotoPollFinishedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_poll_finished",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_poll_finished",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoPollFinishedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_poll_finished]收到消息:", zap.ByteString("body", delivery.Body))
			// 解析消息
			message, err := validator.UnmarshalV[model.PollFinishedMessage](ctx, delivery.Body)
			if err != nil {
				return errors.Wrap(err, "解析消息失败")
			}
			// 渲染模板
			html, err := django.RenderTemplate("email/poll_finished", django.Context{
				"username":    message.Username,
				"poll_id":     message.PollID,
				"operated_at": carbon.Parse(message.UpdatedAt).Format("Y-m-d H:i:s"),
				"hitokoto":    message.Hitokoto,
				"from":        message.From,
				"from_who":    message.FromWho,
				"creator":     message.Creator,
				"type":        formatter.FormatHitokotoType(message.Type),
				"status":      formatter.FormatPollStatus(message.Status),
				"method":      formatter.FormatPollMethod(message.Method),
				"point":       strconv.Itoa(message.Point),
				"now":         carbon.Now().Format("Y 年 n 月 j 日"),
			})
			if err != nil {
				return errors.Wrap(err, "无法渲染模板")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！投票结果出炉了！",
					Body:    html,
				},
			})
			return err
		},
	}
}
