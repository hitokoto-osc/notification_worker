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

type hitokotoPollDailyReportEvent struct {
}

func (t *hitokotoPollDailyReportEvent) Receiver() *rabbitmq.Receiver {
	return &rabbitmq.Receiver{
		ExchangeType: amqp.ExchangeDirect,
		ExchangeName: "notification",
		ConsumerName: "HitokotoPollDailyReportNotificationWorker",
		QueueName:    "hitokoto_poll_daily_report",
		BindingKey:   "notification.hitokoto_poll_daily_report", // 路由键
		Deliveries:   make(chan amqp.Delivery),
		HandlerFunc: func(msg amqp.Delivery) error { // 回调处理方法
			log.Printf("[hitokoto_poll_daily_report]收到消息: %v  \n", string(msg.Body))
			message := hitokotoPollDailyReportMessage{}
			err := json.Unmarshal(msg.Body, &message)
			if err != nil {
				return err
			}
			// 解析 ISO 时间
			CreatedAt, err := time.ParseInLocation("2006-01-02T15:04:05.999Z", message.CreatedAt, time.UTC)
			if err != nil {
				return err
			}

			html := fmt.Sprintf(`<h2>您好，%s。</h2>
<p>今日份的投票报告制作好了！创建时间：%s</p>
<p>在过去 24 小时中，出现了这些变化：
平台统计：
<ul>
  <li>等待投票：%s</li>
  <li>处理投票：%s</li>
  <li>入库：%s</li>
  <li>驳回：%s</li>
  <li>需要修改：%s</li>
</ul>
用户统计：
<ul>
  <li>参与投票：%s</li>
  <li>等待处理：%s</li>
  <li>已入库：%s</li>
  <li>已驳回：%s</li>
  <li>需要修改：%s</li>
</ul>
<p>在参与的投票中，您对 %s 个投票选择了“批准”，您对 %s 个投票选择了“驳回”，您对 %s 个投票选择了“需要修改”。<br />
您还有 <strong>%s</strong> 个投票需要处理。</p>

<p>感谢您的付出！<br/>
“生命从无中来，到无中去，每个人都处于“上场——谢幕”这样一个循环中。这个循环不是悲伤，不是无意义，意义就在这过程中；生命之所以有趣，就在于过程中的体验和收获。”<br />
以此共勉。</p>
<br />
<p>萌创团队 - 一言项目组<br />
%s</p>`,

				CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
				strconv.Itoa(message.SystemInformation.Total),
				strconv.Itoa(message.SystemInformation.ProcessTotal),
				strconv.Itoa(message.SystemInformation.ProcessAccept),
				strconv.Itoa(message.SystemInformation.ProcessReject),
				strconv.Itoa(message.SystemInformation.ProcessNeedEdited),
				strconv.Itoa(message.UserInformation.Polled.Total),
				strconv.Itoa(message.UserInformation.Waiting),
				strconv.Itoa(message.UserInformation.Accepted),
				strconv.Itoa(message.UserInformation.Rejected),
				strconv.Itoa(message.UserInformation.InNeedEdited),
				strconv.Itoa(message.UserInformation.Polled.Accept),
				strconv.Itoa(message.UserInformation.Polled.Reject),
				strconv.Itoa(message.UserInformation.Polled.NeedEdited),
				strconv.Itoa(message.UserInformation.WaitForPolling),
				time.Now().Format("2006年1月2日"),
			)
			err = directmail.SingleSendMail(message.To, "喵！投票结果出炉了！", html, true)
			return err
		},
	}
}

type hitokotoPollDailyReportMessage struct {
	CreatedAt         string                                          `json:"created_at"`         // 报告生成时间
	SystemInformation hitokotoPollDailyReportMessageSystemInformation `json:"system_information"` // 系统信息
	UserInformation   hitokotoPollDailyReportMessageUserInformation   `json:"user_information"`   // 用户信息
}

type hitokotoPollDailyReportMessageSystemInformation struct {
	Total             int `json:"total"`               // 平台当前剩余的投票数目
	ProcessTotal      int `json:"process_total"`       // 平台处理了的投票数目（过去 24 小时）
	ProcessAccept     int `json:"process_accept"`      // 平台处理为入库的投票数目（过去 24 小时）
	ProcessReject     int `json:"process_reject"`      // 平台处理为驳回的投票数目（过去 24 小时）
	ProcessNeedEdited int `json:"process_need_edited"` // 平台处理为亟待修改的投票数目（过去 24 小时）
}

type hitokotoPollDailyReportMessageUserInformation struct {
	Polled         hitokotoPollDailyReportMessageUserInformationPolled `json:"polled"`           // 用户参与了的投票数目（过去 24 小时）
	Waiting        int                                                 `json:"waiting"`          // 等待其他用户参与的投票数目（基于已投票的数目）
	Accepted       int                                                 `json:"accepted"`         // 已入库的投票数目（基于已投票的数目）
	Rejected       int                                                 `json:"rejected"`         // 已驳回的投票数目（基于已投票的数目）
	InNeedEdited   int                                                 `json:"in_need_edited"`   // 已进入亟待修改状态的投票数目（基于已投票的数目）
	WaitForPolling int                                                 `json:"wait_for_polling"` // 基于剩余投票数目，计算出来的等待投票数目。
}

type hitokotoPollDailyReportMessageUserInformationPolled struct {
	Total      int `json:"total"`       // 投票参与的总数
	Accept     int `json:"accept"`      // 投批准票的投票数目
	Reject     int `json:"reject"`      // 投拒绝票的投票数目
	NeedEdited int `json:"need_edited"` // 投需要修改的投票数目
}
