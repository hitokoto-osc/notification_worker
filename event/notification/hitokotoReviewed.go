package notification

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"source.hitokoto.cn/hitokoto/notification-worker/aliyun/directmail"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
	"strconv"
	"time"
)

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
		CallFunc: func(delivery amqp.Delivery) error {
			log.Debugf("[hitokoto_reviewed]收到消息: %v  \n", string(delivery.Body))
			message := hitokotoReviewedMessage{}
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return err
			}
			// 转换成时间戳
			ts, err := strconv.ParseInt(message.CreatedAt, 10, 64)
			if err != nil {
				return err
			}

			// 处理数据
			var reviewResult = &struct {
				StatusText string
				Desc       string
				OperatedAt time.Time
			}{}
			reviewResult.OperatedAt, err = time.ParseInLocation("2006-01-02T15:04:05.999Z", message.OperatedAt, time.UTC)
			if err != nil {
				return err
			}
			if message.Status == 200 { // 只可能通过或者驳回，目前。
				reviewResult.StatusText = "通过"
				reviewResult.Desc = "。"
			} else {
				reviewResult.StatusText = "驳回"
				reviewResult.Desc = "。如果您对审核结果有疑问，您可以在“提交历史”中点击“查看详情”，再点击“查看审核意见”查看审核意见。如果对处理结果不满意的话，可以发信至 <code>i@loli.online</code> 联系我们（备注句子 UUID）。"
			}
			html := fmt.Sprintf(`<h2>您好，%s。</h2>
<p>您于 %s 提交的句子： <b>%s</b> —— %s 「%s」， 已经审核完成。</p>
<p>审核结果为：<strong>%s</strong>，审核员 %s (%d) 于 %s 操作审核%s</p>
<br />
<p>感谢您的支持，<br />
萌创团队 - 一言项目组<br />
%s</p>`,
				message.Creator,
				time.Unix(ts, 0).Format("2006-01-02 15:04:05"),
				message.Hitokoto, message.FromWho,
				message.From,
				reviewResult.StatusText,
				message.ReviewerName,
				message.ReviewerUid,
				reviewResult.OperatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
				reviewResult.Desc,
				time.Now().Format("2006年1月2日"),
			)
			err = directmail.SingleSendMail(message.To, "喵！您的句子审核结果出来了！", html, true)
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
