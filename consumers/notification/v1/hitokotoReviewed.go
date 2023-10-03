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
	provider.Register(HitokotoReviewedEvent())
}

// HitokotoReviewedEvent 处理一言成功添加事件
func HitokotoReviewedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_reviewed",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_reviewed",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoReviewedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_reviewed] 收到消息:", zap.ByteString("body", delivery.Body))
			message, err := validator.UnmarshalV[model.HitokotoReviewedMessage](ctx, delivery.Body)
			if err != nil {
				return errors.Wrap(err, "无法解析消息体")
			}
			html, err := django.RenderTemplate("email/hitokoto_reviewed", django.Context{
				"username":      message.Creator,
				"created_at":    message.CreatedAt.Format("Y-m-d H:i:s"),
				"hitokoto":      message.Hitokoto,
				"from_who":      message.FromWho,
				"from":          message.From,
				"type":          formatter.FormatHitokotoType(message.Type),
				"reviewer":      message.ReviewerName,
				"reviewer_uid":  message.ReviewerUid,
				"reviewed_at":   message.OperatedAt.Format("Y-m-d H:i:s"),
				"review_result": formatter.FormatPollStatus(message.Status),
			})
			if err != nil {
				return errors.Wrap(err, "渲染模板失败")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！您的句子审核结果出来了！",
					Body:    html,
				},
			})
			return err
		},
	}
}
