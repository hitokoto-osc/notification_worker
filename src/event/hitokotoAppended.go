package event

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"source.hitokoto.cn/hitokoto/notification-worker/src/aliyun/directmail"
	"source.hitokoto.cn/hitokoto/notification-worker/src/rabbitmq"
	"strconv"
	"time"
)

type hitokotoAppendedEvent struct {
}

// 处理一言成功添加事件
func (t *hitokotoAppendedEvent) Receiver() *rabbitmq.Receiver {
	return &rabbitmq.Receiver{
		ExchangeType: amqp.ExchangeDirect,
		ExchangeName: "notification",
		ConsumerName: "HitokotoAppendedNotificationWorker",
		QueueName:    "hitokoto_appended",
		BindingKey:   "notification.hitokoto_appended", // 路由键
		Deliveries:   make(chan amqp.Delivery),
		HandlerFunc: func(msg amqp.Delivery) error { // 回调处理方法
			log.Debugf("[hitokoto_appended] 收到消息: %v  \n", string(msg.Body))
			message := hitokotoAppendedMessage{}
			err := json.Unmarshal(msg.Body, &message)
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
