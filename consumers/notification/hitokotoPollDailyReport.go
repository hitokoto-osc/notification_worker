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
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func init() {
	provider.Register(HitokotoPollDailyReportEvent())
}

// HitokotoPollDailyReportEvent 每日审核员报告事件
func HitokotoPollDailyReportEvent() *rabbitmq.ConsumerRegisterOptions {
	return &rabbitmq.ConsumerRegisterOptions{
		Exchange: rabbitmq.Exchange{
			Name:    "notification",
			Type:    "direct",
			Durable: true,
		},
		Queue: rabbitmq.Queue{
			Name:    "hitokoto_poll_daily_report",
			Durable: true,
			Args: amqp.Table{
				"x-dead-letter-exchange":    "notification_failed",
				"x-dead-letter-routing-key": "notification_failed.notification_failed_collector",
			},
		},
		BindingOptions: rabbitmq.BindingOptions{
			RoutingKey: "notification.hitokoto_poll_daily_report",
		},
		ConsumerOptions: rabbitmq.ConsumerOptions{
			Tag:        "HitokotoPollDailyReportNotificationWorker",
			AckByError: true,
		},
		CallFunc: func(ctx rabbitmq.Ctx, delivery amqp.Delivery) error {
			logger := logging.WithContext(ctx)
			defer logger.Sync()
			logger.Debug("[hitokoto_poll_daily_report]收到消息: ", zap.ByteString("body", delivery.Body))
			message := hitokotoPollDailyReportMessage{}
			err := json.Unmarshal(delivery.Body, &message)
			if err != nil {
				return err
			}

			html, err := django.RenderTemplate("email/poll_daily_report", django.Context{
				"username":   message.UserName,
				"created_at": carbon.Parse(message.CreatedAt).Format("Y-m-d H:i:s"),
				"system": django.Context{
					"total":       message.SystemInformation.Total,
					"processed":   message.SystemInformation.ProcessTotal,
					"approved":    message.SystemInformation.ProcessAccept,
					"rejected":    message.SystemInformation.ProcessReject,
					"need_modify": message.SystemInformation.ProcessNeedEdited,
				},
				"user": django.Context{
					"polled": django.Context{
						"total":       message.UserInformation.Polled.Total,
						"approve":     message.UserInformation.Polled.Accept,
						"reject":      message.UserInformation.Polled.Reject,
						"need_modify": message.UserInformation.Polled.NeedEdited,
					},
					"wait_for_polling":   message.UserInformation.WaitForPolling,
					"waiting_for_others": message.UserInformation.Waiting,
					"approved":           message.UserInformation.Accepted,
					"rejected":           message.UserInformation.Rejected,
					"need_modify":        message.UserInformation.InNeedEdited,
				},
			})

			if err != nil {
				return errors.Wrap(err, "无法渲染模板")
			}
			err = mail.SendSingle(ctx, &mailer.Mailer{
				Type: mailer.TypeNormal,
				Mail: mailer.Mail{
					To:      []string{message.To},
					Subject: "喵！今日份的投票报告来了！",
					Body:    html,
				},
			})
			return err
		},
	}
}

type hitokotoPollDailyReportMessage struct {
	CreatedAt         string                                          `json:"created_at"`         // 报告生成时间
	To                string                                          `json:"to"`                 // 接收人地址
	UserName          string                                          `json:"user_name"`          // 接收人名称
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
