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
		CallFunc: func(delivery amqp.Delivery) error {
			log.Debugf("[hitokoto_appended] 收到消息: %v  \n", string(delivery.Body))
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
			html := fmt.Sprintf(`<h2>您好，%s。</h2>
<p>您于 %s 提交的句子： <b>%s</b> —— %s 「%s」， 已经进入审核队列了。</p>
<p>我们会尽快处理您的句子。当审核结果出来时，我们将会通过邮件通知您。</p>
<br />
<p>感谢您的支持，<br />
萌创团队 - 一言项目组<br />
%s</p>`,
				message.Creator,
				time.Unix(ts, 0).Format("2006-01-02 15:04:05"),
				message.Hitokoto, message.FromWho,
				message.From,
				time.Now().Format("2006年1月2日"),
			)
			err = directmail.SingleSendMail(message.To, "喵！已经成功收到您提交的句子了！", html, true)
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
