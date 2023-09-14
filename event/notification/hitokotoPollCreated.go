package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-module/carbon/v2"
	"github.com/hitokoto-osc/notification-worker/aliyun/directmail"
	"github.com/hitokoto-osc/notification-worker/logging"
	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// HitokotoPollCreatedEvent 处理一言成功添加事件
func HitokotoPollCreatedEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_poll_created",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_poll_created",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoPollCreatedNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx context.Context, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_poll_created]收到消息：", zap.ByteString("body", delivery.Body))
			message := hitokotoPollCreatedMessage{}
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return err
			}
			// 解析 ISO 时间
			createdAt := carbon.Parse(message.CreatedAt)
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
				createdAt.Format("Y-m-d H:i:s"),
				message.Id,
				message.Hitokoto,
				message.From,
				message.FromWho,
				message.Creator,
				carbon.Now().Format("Y 年 n 月 j 日"),
			)
			err = directmail.SingleSendMail(ctx, message.To, "喵！新的野生投票菌出现了！", html, true)
			return err
		},
	}
}

type hitokotoPollCreatedMessage struct {
	hitokotoAppendedMessage
	UserName  string `json:"user_name"`  // 收信人
	Id        int    `json:"id"`         // 投票标识
	CreatedAt string `json:"created_at"` // 这里是投票创建时间， ISO 时间
}
