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
			message := hitokotoReviewedMessage{}
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
			var reviewResult string
			if message.Status == 200 { // 只可能通过或者驳回，目前。
				reviewResult = "通过"
			} else {
				reviewResult = "驳回"
			}

			html, err := django.RenderTemplate("email/hitokoto_reviewed", django.Context{
				"username":      message.Creator,
				"created_at":    carbon.CreateFromTimestamp(ts).Format("Y-m-d H:i:s"),
				"hitokoto":      message.Hitokoto,
				"from_who":      message.FromWho,
				"from":          message.From,
				"type":          utils.FormatHitokotoType(message.Type),
				"reviewer":      message.ReviewerName,
				"reviewer_uid":  message.ReviewerUid,
				"reviewed_at":   carbon.Parse(message.OperatedAt).Format("Y-m-d H:i:s"),
				"review_result": reviewResult,
				"now":           carbon.Now().Format("Y-m-d H:i:s"),
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

type hitokotoReviewedMessage struct {
	hitokotoAppendedMessage
	OperatedAt   string `json:"operated_at"`   // 操作时间
	ReviewerName string `json:"reviewer_name"` // 审核员名称
	ReviewerUid  int    `json:"reviewer_uid"`  // 审核员用户标识
	Status       int    `json:"status"`        // 审核结果： 200 为通过，201 为驳回
}
