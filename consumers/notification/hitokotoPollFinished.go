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
			message := hitokotoPollFinishedMessage{}
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return err
			}

			// 处理 JSON 数据
			var (
				method string
				status string
			)
			switch message.Status {
			case 200:
				status = "入库"
			case 201:
				status = "驳回"
			case 202:
				status = "需要修改"
			default:
				status = "未知状态"
			}
			switch message.Method {
			case 1:
				method = "批准"
			case 2:
				method = "驳回"
			case 3:
				method = "需要修改"
			default:
				method = "未知操作"
			}

			html, err := django.RenderTemplate("email/poll_finished", django.Context{
				"username":    message.UserName,
				"poll_id":     message.Id,
				"operated_at": carbon.Parse(message.UpdatedAt).Format("Y-m-d H:i:s"),
				"hitokoto":    message.Hitokoto,
				"from":        message.From,
				"from_who":    message.FromWho,
				"creator":     message.Creator,
				"type":        utils.FormatHitokotoType(message.Type),
				"status":      status,
				"method":      method,
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

type hitokotoPollFinishedMessage struct {
	hitokotoAppendedMessage
	Id        int    `json:"id"`         // 投票 ID
	UpdatedAt string `json:"updated_at"` // 投票更新时间，这里也是结束时间
	UserName  string `json:"user_name"`  // 审核员名字
	CreatedAt string `json:"created_at"` // 投票创建时间
	Status    int    `json:"status"`     // 投票结果： 200 入库，201 驳回，202 需要修改
	Method    int    `json:"method"`     // 审核员投票方式： 1 入库，2 驳回，3 需要修改
	Point     int    `json:"point"`      // 审核员投的票数
	// TODO: 加入审核员投票的意见标签？
}
