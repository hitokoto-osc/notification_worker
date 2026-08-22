// Package notification 提供通知摘要（digest）的展示层模型与渲染。
//
// 本包只处理「已经归一化好的展示数据 → 邮件主题 + HTML」这一段，
// 不依赖 AMQP、领域消息模型或聚合引擎，因此可以独立于合并窗口先行验证。
// 领域消息到 Item 的归一化由 consumers/notification/v1 负责——
// 消息模型位于 internal 包，本包无法（也不应该）导入。
package notification

import "time"

type EventType string

const (
	EventHitokotoAppended     EventType = "hitokoto_appended"
	EventHitokotoReviewed     EventType = "hitokoto_reviewed"
	EventHitokotoMoved        EventType = "hitokoto_moved"
	EventHitokotoPollCreated  EventType = "hitokoto_poll_created"
	EventHitokotoPollFinished EventType = "hitokoto_poll_finished"
)

type DigestGroup string

const (
	// GroupContributor 投稿者视角：提交、审核结果、重新审核。
	GroupContributor DigestGroup = "contributor"
	// GroupReviewer 审核员视角：新投票、投票结算。
	GroupReviewer DigestGroup = "reviewer"
)

// Tone 决定模板里徽章与分节色带的呈现。业务状态到 tone 的映射在 Go 侧完成，
// 模板不解析状态文案。
type Tone string

const (
	ToneNeutral Tone = "neutral"
	ToneSuccess Tone = "success"
	ToneDanger  Tone = "danger"
	ToneWarning Tone = "warning"
)

// 与 consts.PollStatus 对应的终态取值。此处独立定义，避免展示层反向依赖 consts。
const (
	statusApproved   = 200
	statusRejected   = 201
	statusNeedModify = 202
)

// DefaultDisplayCap 是单个分节默认展示的条目上限。
// 超出部分折叠为溢出提示，用于规避 Gmail 在 102KB 处截断正文。
const DefaultDisplayCap = 15

// GroupOf 返回事件所属的聚合分组。未知事件类型返回 false。
func GroupOf(t EventType) (DigestGroup, bool) {
	switch t {
	case EventHitokotoAppended, EventHitokotoReviewed, EventHitokotoMoved:
		return GroupContributor, true
	case EventHitokotoPollCreated, EventHitokotoPollFinished:
		return GroupReviewer, true
	default:
		return "", false
	}
}

// isTerminal 表示该事件给出了句子的最终处理结果。
func isTerminal(t EventType) bool {
	return t == EventHitokotoReviewed || t == EventHitokotoMoved
}

// Item 是一条已归一化的展示条目，不含任何领域类型。
type Item struct {
	EventID      string
	Type         EventType
	SentenceUUID string

	Hitokoto  string
	From      string
	FromWho   *string
	TypeLabel string

	StatusCode  int
	StatusLabel string
	OccurredAt  time.Time

	// 审核员视角专用
	PollID      int64
	Creator     string
	MethodLabel string
	Point       int

	// Collapsed 由 Collapse 置位：同一句子的「提交」事件已被折叠进本条终态。
	Collapsed bool
}

// Digest 是一个收件人在一个窗口内待渲染的全部内容。
type Digest struct {
	Group          DigestGroup
	RecipientName  string
	WindowDuration string
	ActionURL      string
	ActionText     string
	// DisplayCap 为 0 时使用 DefaultDisplayCap。
	DisplayCap int
	Items      []Item
}

// Metrics 是摘要顶部统计胶囊与主题行的数据来源。
type Metrics struct {
	// EventCount 是折叠前的事件数，Total 是折叠后实际渲染的条目数。
	EventCount int
	Total      int

	Approved     int
	Rejected     int
	NeedModify   int
	Attention    int // Rejected + NeedModify + 其他非成功终态
	Pending      int
	PollCreated  int
	PollFinished int
}

// RenderKind 指示调用方应当发哪一种邮件。
type RenderKind uint8

const (
	// RenderSingle 表示窗口内只有一条事件，应回退到现有单条模板与原主题。
	RenderSingle RenderKind = iota
	// RenderDigest 表示已渲染出摘要邮件。
	RenderDigest
)

// Result 是渲染结果。Kind 为 RenderSingle 时只有 Single 有效。
type Result struct {
	Kind    RenderKind
	Single  *Item
	Subject string
	HTML    string
	Metrics Metrics
}

func (d Digest) displayCap() int {
	if d.DisplayCap <= 0 {
		return DefaultDisplayCap
	}
	return d.DisplayCap
}
