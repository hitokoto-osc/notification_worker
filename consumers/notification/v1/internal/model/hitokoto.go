// Package model 消息模型
package model

import (
	"github.com/hitokoto-osc/notification-worker/consts"
	"github.com/hitokoto-osc/notification-worker/consumers/notification/v1/internal/vcarbon"
)

// Hitokoto 消息结构体
// FIXME: 应该抽象成一个全局共用的结构体，而不是每个模块都有一个，因为历史因素 v2 应该重新设计。
type Hitokoto struct {
	To        string              `json:"to" validate:"email,required"`   // 对象邮件地址
	UUID      string              `json:"uuid" validate:"uuid4,required"` // 句子 UUID
	Hitokoto  string              `json:"hitokoto" validate:"required"`   // 句子
	From      string              `json:"from" validate:"required"`       // 来源
	Type      consts.HitokotoType `json:"type" validate:"required"`       // 分类
	FromWho   *string             `json:"from_who"`                       // 作者
	Creator   string              `json:"creator" validate:"required"`    // 提交者名称
	CreatedAt *vcarbon.Carbon     `json:"created_at" validate:"required"` // 提交时间
}

// HitokotoAppendedMessage 句子添加消息
type HitokotoAppendedMessage Hitokoto

// HitokotoMovedMessage 句子移动消息
type HitokotoMovedMessage struct {
	Hitokoto
	OperatedAt       *vcarbon.Carbon   `json:"operated_at" validate:"required"`       // 操作时间
	OperatorUsername string            `json:"operator_username" validate:"required"` // 操作者用户名
	OperatorUID      uint              `json:"operator_uid" validate:"required"`      // 操作者 UID
	Operate          consts.PollStatus `json:"operate" `                              // 操作
}

type HitokotoReviewedMessage struct {
	Hitokoto
	OperatedAt   *vcarbon.Carbon   `json:"operated_at" validate:"required"`   // 操作时间
	ReviewerName string            `json:"reviewer_name" validate:"required"` // 审核员名称
	ReviewerUid  int               `json:"reviewer_uid" validate:"required"`  // 审核员用户标识
	Status       consts.PollStatus `json:"status"`                            // 审核结果： 200 为通过，201 为驳回
}

// PollCreatedMessage 投票创建消息
type PollCreatedMessage struct {
	Hitokoto
	Username  string          `json:"user_name" validate:"required"`  // 收信人
	ID        uint            `json:"id" validate:"required"`         // 投票标识
	CreatedAt *vcarbon.Carbon `json:"created_at" validate:"required"` // 这里是投票创建时间， ISO 时间
}

// PollDailyReportMessage 每日审核员报告消息
type PollDailyReportMessage struct {
	CreatedAt         *vcarbon.Carbon `json:"created_at" validate:"required"` // 报告生成时间
	To                string          `json:"to" validate:"required,email"`   // 接收人地址
	Username          string          `json:"user_name" validate:"required"`  // 接收人名称
	SystemInformation *struct {
		Total             int `json:"total"`               // 平台当前剩余的投票数目
		ProcessTotal      int `json:"process_total"`       // 平台处理了的投票数目（过去 24 小时）
		ProcessAccept     int `json:"process_accept"`      // 平台处理为入库的投票数目（过去 24 小时）
		ProcessReject     int `json:"process_reject"`      // 平台处理为驳回的投票数目（过去 24 小时）
		ProcessNeedEdited int `json:"process_need_edited"` // 平台处理为亟待修改的投票数目（过去 24 小时）
	} `json:"system_information" validate:"required"` // 系统信息
	UserInformation *struct {
		Polled *struct {
			Total      int `json:"total"`       // 投票参与的总数
			Accept     int `json:"accept"`      // 投批准票的投票数目
			Reject     int `json:"reject"`      // 投拒绝票的投票数目
			NeedEdited int `json:"need_edited"` // 投需要修改的投票数目
		} `json:"polled"` // 用户参与了的投票数目（过去 24 小时）
		Waiting        int `json:"waiting"`          // 等待其他用户参与的投票数目（基于已投票的数目）
		Accepted       int `json:"accepted"`         // 已入库的投票数目（基于已投票的数目）
		Rejected       int `json:"rejected"`         // 已驳回的投票数目（基于已投票的数目）
		InNeedEdited   int `json:"in_need_edited"`   // 已进入亟待修改状态的投票数目（基于已投票的数目）
		WaitForPolling int `json:"wait_for_polling"` // 基于剩余投票数目，计算出来的等待投票数目。
	} `json:"user_information" validate:"required"` // 用户信息
}

// PollFinishedMessage 投票完成消息
type PollFinishedMessage struct {
	Hitokoto
	PollID    int               `json:"id"`         // 投票 ID
	UpdatedAt string            `json:"updated_at"` // 投票更新时间，这里也是结束时间
	Username  string            `json:"user_name"`  // 审核员名字
	CreatedAt string            `json:"created_at"` // 投票创建时间
	Status    consts.PollStatus `json:"status"`     // 投票结果： 200 入库，201 驳回，202 需要修改
	Method    consts.PollMethod `json:"method"`     // 审核员投票方式： 1 入库，2 驳回，3 需要修改
	Point     int               `json:"point"`      // 审核员投的票数
	// TODO: 加入审核员投票的意见标签？
}
