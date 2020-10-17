package event

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"source.hitokoto.cn/hitokoto/notification-worker/aliyun/directmail"
	"source.hitokoto.cn/hitokoto/notification-worker/rabbitmq"
	"strconv"
	"time"
)

type hitokotoPollFinishedEvent struct {
}

func (t *hitokotoPollFinishedEvent) Receiver() *rabbitmq.Receiver {
	return &rabbitmq.Receiver{
		ExchangeType: amqp.ExchangeDirect,
		ExchangeName: "notification",
		ConsumerName: "HitokotoPollFinishedNotificationWorker",
		QueueName:    "hitokoto_poll_finished",
		BindingKey:   "notification.hitokoto_poll_finished", // 路由键
		Deliveries:   make(chan amqp.Delivery),
		HandlerFunc: func(msg amqp.Delivery) error { // 回调处理方法
			log.Debugf("[hitokoto_poll_finished]收到消息: %v  \n", string(msg.Body))
			message := hitokotoPollFinishedMessage{}
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				return err
			}
			// 解析 ISO 时间
			updatedAt, err := time.ParseInLocation("2006-01-02T15:04:05.999Z", message.UpdatedAt, time.UTC)
			if err != nil {
				return err
			}

			// 处理 JSON 数据
			var result = struct {
				StatusText string
				MethodText string
			}{}
			if message.Status == 200 {
				result.StatusText = "入库"
			} else if message.Status == 201 {
				result.StatusText = "驳回"
			} else if message.Status == 202 {
				result.StatusText = "需要修改"
			} else {
				result.StatusText = "未知状态"
			}
			if message.Method == 1 {
				result.MethodText = "批准"
			} else if message.Method == 2 {
				result.MethodText = "驳回"
			} else if message.Method == 3 {
				result.MethodText = "需要修改"
			} else {
				result.MethodText = "未知操作"
			}

			html := fmt.Sprintf(`<h2>您好，%s。</h2>
<p>投票（id: %d）于 %s 由系统自动处理。</p>
<p>句子信息：</p>
<ul>
  <li>内容：%s</li>
  <li>来源：%s</li>
  <li>作者：%s</li>
  <li>提交者： %s</li>
</ul>
<p>处理结果为：<strong>%s</strong>。
您在本次投票中投了 <b>%s</b> %s 票。如果您想了解投票的详细信息（包括“投票数据”），可以查看“审核员中心”的“结果与记录”页。<br />
如果您觉得消息提醒过于频繁，可以在“用户设置”页面关闭“投票结果通知”选项。</p>
<br />
<p>感谢您的支持，<br />
萌创团队 - 一言项目组<br />
%s</p>`,
				message.UserName,
				message.Id,
				updatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
				message.Hitokoto,
				message.From,
				message.FromWho,
				message.Creator,
				result.StatusText,
				result.MethodText,
				strconv.Itoa(message.Point), // 直接转换成字符串之后再插入
				time.Now().Format("2006年1月2日"),
			)
			err = directmail.SingleSendMail(message.To, "喵！投票结果出炉了！", html, true)
			return err
		},
	}
}

type hitokotoPollFinishedMessage struct {
	hitokotoAppendedMessage
	Id        int `json:"id"`         // 投票 ID
	UpdatedAt string `json:"updated_at"` // 投票更新时间，这里也是结束时间
	UserName  string `json:"user_name"`  // 审核员名字
	CreatedAt string `json:"created_at"` // 投票创建时间
	Status    int    `json:"status"`     // 投票结果： 200 入库，201 驳回，202 需要修改
	Method    int    `json:"method"`     // 审核员投票方式： 1 入库，2 驳回，3 需要修改
	Point     int    `json:"point"`      // 审核员投的票数
	// TODO: 加入审核员投票的意见标签？
}
