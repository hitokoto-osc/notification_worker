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
			message := hitokotoAppendedMessage{}
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return err
			}
			// 转换成时间戳
			ts, err := strconv.ParseInt(message.CreatedAt, 10, 64)
			if err != nil {
				return err
			}
			html, err := django.RenderTemplate("email/hitokoto_appended", django.Context{
				"username":   message.Creator,
				"created_at": carbon.CreateFromTimestamp(ts).Format("Y-m-d H:i:s"),
				"hitokoto":   message.Hitokoto,
				"from":       message.From,
				"from_who":   message.FromWho,
				"type":       utils.FormatHitokotoType(message.Type),
				"now":        carbon.Now().Format("Y 年 n 月 j 日"),
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

type hitokotoAppendedMessage struct {
	To        string `json:"to"`         // 对象邮件地址
	UUID      string `json:"uuid"`       // 句子 UUID
	Hitokoto  string `json:"hitokoto"`   // 句子
	From      string `json:"from"`       // 来源
	Type      string `json:"type"`       // 分类
	FromWho   string `json:"from_who"`   // 作者
	Creator   string `json:"creator"`    // 提交者名称
	CreatedAt string `json:"created_at"` // 提交时间
}
