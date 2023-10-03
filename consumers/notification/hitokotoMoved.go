package notification

import (
	"encoding/json"
	"github.com/cockroachdb/errors"
	"github.com/golang-module/carbon/v2"
	"github.com/hitokoto-osc/notification-worker/consumers/provider"
	"github.com/hitokoto-osc/notification-worker/django"
	"github.com/hitokoto-osc/notification-worker/logging"
	"github.com/hitokoto-osc/notification-worker/mail"
	"github.com/hitokoto-osc/notification-worker/mail/mailer"
	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	"github.com/hitokoto-osc/notification-worker/utils"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"strconv"
)

func init() {
	provider.Register(HitokotoMovedEvent())
}

// HitokotoMovedEvent 处理一言被管理员移动的通知
func HitokotoMovedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_moved",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_moved",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoMovedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("收到消息:", zap.ByteString("body", delivery.Body))
			message := new(HitokotoMovedMessage)
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return errors.Wrap(err, "无法解析消息体")
			}
			// 转换成时间戳
			ts, err := strconv.ParseInt(message.CreatedAt, 10, 64)
			if err != nil {
				return errors.Wrap(err, "无法解析时间戳")
			}

			// 处理数据
			var operate string
			if message.Operate == 200 { // 只可能通过或者驳回，目前。
				operate = "通过"
			} else {
				operate = "驳回"
			}

			html, err := django.RenderTemplate("email/hitokoto_reviewed", django.Context{
				"username":          message.Creator,
				"created_at":        carbon.CreateFromTimestamp(ts).Format("Y-m-d H:i:s"),
				"hitokoto":          message.Hitokoto,
				"from_who":          message.FromWho,
				"from":              message.From,
				"type":              utils.FormatHitokotoType(message.Type),
				"operate":           operate,
				"operator_username": message.OperatorUsername,
				"operator_uid":      message.OperatorUID,
				"operated_at":       message.OperatedAt,
			})
			if err != nil {
				return errors.Wrap(err, "渲染模板失败")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！您的句子已重新审核！",
					Body:    html,
				},
			})
			return err
		},
	}
}

type HitokotoMovedMessage struct {
	hitokotoAppendedMessage
	OperatedAt       string `json:"operated_at"`       // 操作时间
	OperatorUsername string `json:"operator_username"` // 操作者用户名
	OperatorUID      string `json:"operator_uid"`      // 操作者 UID
	Operate          int    `json:"operate"`           // 操作
}
