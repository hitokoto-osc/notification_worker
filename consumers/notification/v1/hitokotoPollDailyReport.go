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
	"github.com/hitokoto-osc/notification-worker/utils/validator"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func init() {
	provider.Register(HitokotoPollDailyReportEvent())
}

// HitokotoPollDailyReportEvent 每日审核员报告事件
func HitokotoPollDailyReportEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_poll_daily_report",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_poll_daily_report",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoPollDailyReportNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_poll_daily_report]收到消息: ", zap.ByteString("body", delivery.Body))
			message, err := validator.UnmarshalV[model.PollDailyReportMessage](ctx, delivery.Body)
			if err != nil {
				return errors.Wrap(err, "解析消息失败")
			}

			html, err := django.RenderTemplate("email/poll_daily_report", django.Context{
				"username":   message.Username,
				"created_at": message.CreatedAt.Format("Y-m-d H:i:s"),
				"system": django.Context{
					"total":       message.SystemInformation.Total,
					"processed":   message.SystemInformation.ProcessTotal,
					"approved":    message.SystemInformation.ProcessAccept,
					"rejected":    message.SystemInformation.ProcessReject,
					"need_modify": message.SystemInformation.ProcessNeedEdited,
				},
				"user": django.Context{
					"polled": django.Context{
						"total":       message.UserInformation.Polled.Total,
						"approve":     message.UserInformation.Polled.Accept,
						"reject":      message.UserInformation.Polled.Reject,
						"need_modify": message.UserInformation.Polled.NeedEdited,
					},
					"wait_for_polling":   message.UserInformation.WaitForPolling,
					"waiting_for_others": message.UserInformation.Waiting,
					"approved":           message.UserInformation.Accepted,
					"rejected":           message.UserInformation.Rejected,
					"need_modify":        message.UserInformation.InNeedEdited,
				},
			})
			if err != nil {
				return errors.Wrap(err, "无法渲染模板")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！今日份的投票报告来了！",
					Body:    html,
				},
			})
			return err
		},
	}
}
