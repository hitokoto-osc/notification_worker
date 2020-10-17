package event

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"source.hitokoto.cn/hitokoto/notification-worker/aliyun/directmail"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
	"time"
)

type hitokotoPollCreatedEvent struct {
}

// 处理一言成功添加事件
func (t *hitokotoPollCreatedEvent) Receiver() *rabbitmq.Receiver {
	return &rabbitmq.Receiver{
		ExchangeType: amqp.ExchangeDirect,
		ExchangeName: "notification",
		ConsumerName: "HitokotoPollCreatedNotificationWorker",
		QueueName:    "hitokoto_poll_created",
		BindingKey:   "notification.hitokoto_poll_created", // 路由键
		Deliveries:   make(chan amqp.Delivery),
		HandlerFunc: func(msg amqp.Delivery) error { // 回调处理方法
			log.Debugf("[hitokoto_poll_created]收到消息: %v  \n", string(msg.Body))
			message := hitokotoPollCreatedMessage{}
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				return err
			}
			// 解析 ISO 时间
			createdAt, err := time.ParseInLocation("2006-01-02T15:04:05.999Z", message.CreatedAt, time.UTC)
			if err != nil {
				return err
			}
			html := fmt.Sprintf(`<h2>您好，%s。</h2>
<p>我们在 %s 创建了一则新投票（id: %d）。</p>
<p>句子信息：</p>
<ul>
  <li>内容：%s</li>
  <li>来源：%s</li>
  <li>作者：%s</li>
  <li>提交者： %s</li>
</ul>
<p>请您尽快<a href="https://h5.poll.hitokoto.cn/poll" target="_blank">审核</a>。如果您觉得消息提醒过于频繁，可以在“用户设置”页面关闭“投票创建通知”选项。</p>
<br />
<p>感谢您的支持，<br />
萌创团队 - 一言项目组<br />
%s</p>`,
				message.UserName,
				createdAt.In(time.Local).Format("2006-01-02 15:04:05"),
				message.Id,
				message.Hitokoto,
				message.From,
				message.FromWho,
				message.Creator,
				time.Now().Format("2006年1月2日"),
			)
			err = directmail.SingleSendMail(message.To, "喵！新的野生投票菌出现了！", html, true)
			return err
		},
	}
}

type hitokotoPollCreatedMessage struct {
	hitokotoAppendedMessage
	UserName  string `json:"user_name"`  // 收信人
	Id        int `json:"id"`         // 投票标识
	CreatedAt string `json:"created_at"` // 这里是投票创建时间， ISO 时间
}
