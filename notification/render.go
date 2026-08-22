package notification

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/cockroachdb/errors"
	"github.com/hitokoto-osc/notification-worker/django"
)

const occurredAtLayout = "2006-01-02 15:04:05"

type section struct {
	Title    string
	Tone     Tone
	Items    []Item
	Overflow int
}

// Collapse 折叠冗余的「提交」事件（决策 D6）。
//
// 同一句子在一封信里既显示「提交成功」又显示「审核通过」是冗余噪音，因此当窗口内
// 该句子已经有终态事件（审核 / 重新审核）时，丢弃它的「提交」事件，并在最早的那条
// 终态上置 Collapsed，供模板显示「提交并即时入库」一类的合并徽章。
//
// 终态事件之间不折叠：先「入库」后被管理员「移动至驳回」是两次用户可见的状态变迁，
// 丢掉前一条会让摘要从事件流变成状态快照。
//
// 输入不会被修改。返回结果按 OccurredAt 升序，同刻按 EventID 稳定排序。
func Collapse(items []Item) []Item {
	sorted := make([]Item, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].OccurredAt.Equal(sorted[j].OccurredAt) {
			return sorted[i].EventID < sorted[j].EventID
		}
		return sorted[i].OccurredAt.Before(sorted[j].OccurredAt)
	})

	type lifecycle struct {
		hasAppended   bool
		firstTerminal int // sorted 中最早一条终态事件的下标 +1，0 表示没有
	}
	states := make(map[string]lifecycle, len(sorted))
	for i, item := range sorted {
		if item.SentenceUUID == "" {
			continue
		}
		state := states[item.SentenceUUID]
		switch {
		case item.Type == EventHitokotoAppended:
			state.hasAppended = true
		case isTerminal(item.Type) && state.firstTerminal == 0:
			state.firstTerminal = i + 1
		}
		states[item.SentenceUUID] = state
	}

	collapsed := make([]Item, 0, len(sorted))
	for i, item := range sorted {
		state, tracked := states[item.SentenceUUID]
		folds := tracked && state.hasAppended && state.firstTerminal != 0
		if folds && item.Type == EventHitokotoAppended {
			continue // 该句子已有终态，「提交」这条是冗余的
		}
		if folds && i+1 == state.firstTerminal {
			item.Collapsed = true
		}
		collapsed = append(collapsed, item)
	}
	return collapsed
}

// Render 渲染一个窗口的摘要邮件。
//
// 输入事件数为 1 时不渲染摘要，而是返回 RenderSingle，由调用方回退到现有单条模板
// 与原主题——单句居中排版是现有模板的情感设计，塞进表格里观感很差。
func Render(digest Digest) (Result, error) {
	if len(digest.Items) == 0 {
		return Result{}, errors.New("digest 不能为空")
	}
	for i, item := range digest.Items {
		group, ok := GroupOf(item.Type)
		if !ok {
			return Result{}, errors.Newf("条目 %d 的事件类型未知：%q", i, item.Type)
		}
		if group != digest.Group {
			return Result{}, errors.Newf("条目 %d 属于分组 %q，与摘要分组 %q 不符", i, group, digest.Group)
		}
	}
	if len(digest.Items) == 1 {
		item := digest.Items[0]
		return Result{Kind: RenderSingle, Single: &item}, nil
	}

	// 注意：只有「输入事件数为 1」才回退单条模板。两条事件折叠成一行仍然是摘要——
	// 「提交并即时入库」徽章正是为这种情况准备的，且调用方手上没有重建单条模板
	// 所需的审核员信息。
	items := Collapse(digest.Items)
	metrics := calculateMetrics(len(digest.Items), items)
	subject := buildSubject(digest.Group, metrics)

	var (
		templateName string
		sections     []section
		pills        []django.Context
	)
	switch digest.Group {
	case GroupContributor:
		templateName = "email/digest_contributor"
		sections = contributorSections(items, digest.displayCap())
		pills = contributorPills(metrics)
	case GroupReviewer:
		templateName = "email/digest_reviewer"
		sections = reviewerSections(items, digest.displayCap())
		pills = reviewerPills(metrics)
	default:
		return Result{}, errors.Newf("未知的摘要分组：%q", digest.Group)
	}

	html, err := django.RenderTemplate(templateName, django.Context{
		"username":        digest.RecipientName,
		"window_duration": digest.WindowDuration,
		"total":           metrics.Total,
		"metrics":         pills,
		"sections":        sectionContexts(sections),
		"action_url":      safeActionURL(digest.ActionURL),
		"action_text":     digest.ActionText,
	})
	if err != nil {
		return Result{}, errors.Wrap(err, "渲染摘要模板失败")
	}

	return Result{
		Kind:    RenderDigest,
		Subject: subject,
		HTML:    html,
		Metrics: metrics,
	}, nil
}

// safeActionURL 只放行绝对 http(s) 地址。转义能防止属性逃逸，但拦不住
// javascript: 一类的 scheme；取值非法时返回空串，模板会连同 CTA 一起省略。
func safeActionURL(value string) string {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return value
}

func calculateMetrics(eventCount int, items []Item) Metrics {
	m := Metrics{EventCount: eventCount, Total: len(items)}
	for _, item := range items {
		switch item.Type {
		case EventHitokotoAppended:
			m.Pending++
		case EventHitokotoReviewed, EventHitokotoMoved:
			switch item.StatusCode {
			case statusApproved:
				m.Approved++
			case statusRejected:
				m.Rejected++
				m.Attention++
			case statusNeedModify:
				m.NeedModify++
				m.Attention++
			default:
				// 未预期的终态也让用户看见，而不是让整批渲染失败。
				m.Attention++
			}
		case EventHitokotoPollCreated:
			m.PollCreated++
		case EventHitokotoPollFinished:
			m.PollFinished++
		}
	}
	return m
}

func buildSubject(group DigestGroup, m Metrics) string {
	switch group {
	case GroupContributor:
		if m.Pending == m.Total {
			return fmt.Sprintf("喵！已成功收到您提交的 %d 条新句子！", m.Total)
		}
		// 计划原文为「%d 条需修改」，但仅取 NeedModify 会让「全部被驳回」的摘要
		// 显示成「0 条需修改」。此处取 Attention（驳回 + 亟待修改 + 其他非成功终态），
		// 文案相应改为「需处理」。
		return fmt.Sprintf("喵！您有 %d 条一言新动态（%d 条已入库，%d 条需处理）", m.Total, m.Approved, m.Attention)
	case GroupReviewer:
		switch {
		case m.PollCreated == m.Total:
			return fmt.Sprintf("喵！审核队列刷新了 %d 条新投票待您审阅！", m.Total)
		case m.PollFinished == m.Total:
			return fmt.Sprintf("喵！您有 %d 条投票已结束，结算结果出炉！", m.Total)
		default:
			return fmt.Sprintf("喵！您有 %d 条审核动态（%d 条待审，%d 条已结算）", m.Total, m.PollCreated, m.PollFinished)
		}
	default:
		return fmt.Sprintf("喵！您有 %d 条新动态", m.Total)
	}
}

func contributorSections(items []Item, limit int) []section {
	var attention, approved, pending []Item
	for _, item := range items {
		switch {
		case item.Type == EventHitokotoAppended:
			pending = append(pending, item)
		case item.StatusCode == statusApproved:
			approved = append(approved, item)
		default:
			attention = append(attention, item)
		}
	}
	// 「需要您关注」置顶：驳回与亟待修改是用户必须采取行动的信息。
	return buildSections(limit,
		section{Title: "需要您关注", Tone: ToneDanger, Items: attention},
		section{Title: "审核通过并入库", Tone: ToneSuccess, Items: approved},
		section{Title: "成功提交待审", Tone: ToneNeutral, Items: pending},
	)
}

func reviewerSections(items []Item, limit int) []section {
	var created, finished []Item
	for _, item := range items {
		if item.Type == EventHitokotoPollCreated {
			created = append(created, item)
			continue
		}
		finished = append(finished, item)
	}
	return buildSections(limit,
		section{Title: "新投票待审阅", Tone: ToneWarning, Items: created},
		section{Title: "投票已结算", Tone: ToneSuccess, Items: finished},
	)
}

// buildSections 丢弃空分节并按展示上限截断。
func buildSections(limit int, candidates ...section) []section {
	sections := make([]section, 0, len(candidates))
	for _, s := range candidates {
		if len(s.Items) == 0 {
			continue
		}
		if len(s.Items) > limit {
			s.Overflow = len(s.Items) - limit
			s.Items = s.Items[:limit]
		}
		sections = append(sections, s)
	}
	return sections
}

func contributorPills(m Metrics) []django.Context {
	return nonZeroPills(
		pill("已入库", m.Approved, ToneSuccess),
		pill("已驳回", m.Rejected, ToneDanger),
		pill("待修改", m.NeedModify, ToneWarning),
		pill("新提交", m.Pending, ToneNeutral),
	)
}

func reviewerPills(m Metrics) []django.Context {
	return nonZeroPills(
		pill("待审阅", m.PollCreated, ToneWarning),
		pill("已结算", m.PollFinished, ToneSuccess),
	)
}

func pill(label string, count int, tone Tone) django.Context {
	return django.Context{"label": label, "count": count, "tone": string(tone)}
}

func nonZeroPills(pills ...django.Context) []django.Context {
	result := make([]django.Context, 0, len(pills))
	for _, p := range pills {
		if p["count"].(int) > 0 {
			result = append(result, p)
		}
	}
	return result
}

func sectionContexts(sections []section) []django.Context {
	result := make([]django.Context, 0, len(sections))
	for _, s := range sections {
		items := make([]django.Context, 0, len(s.Items))
		for _, item := range s.Items {
			items = append(items, itemContext(item))
		}
		result = append(result, django.Context{
			"title":    s.Title,
			"tone":     string(s.Tone),
			"items":    items,
			"shown":    len(s.Items),
			"overflow": s.Overflow,
		})
	}
	return result
}

func itemContext(item Item) django.Context {
	var fromWho any
	if item.FromWho != nil {
		fromWho = *item.FromWho
	}
	return django.Context{
		"event_id":       item.EventID,
		"event_type":     string(item.Type),
		"sentence_uuid":  item.SentenceUUID,
		"hitokoto":       item.Hitokoto,
		"from":           item.From,
		"from_who":       fromWho,
		"type_label":     item.TypeLabel,
		"status_label":   statusLabel(item),
		"status_tone":    string(statusTone(item)),
		"combined":       item.Collapsed,
		"combined_label": combinedLabel(item),
		"occurred_at":    item.OccurredAt.Format(occurredAtLayout),
		"poll_id":        item.PollID,
		"creator":        item.Creator,
		"method_label":   item.MethodLabel,
		"point":          item.Point,
	}
}

// statusLabel 保证终态条目一定有文案。未知状态会被归入「需要您关注」分节，
// 若同时没有徽章文案，收件人就无从判断这条为什么需要关注。
func statusLabel(item Item) string {
	if item.StatusLabel != "" || !isTerminal(item.Type) {
		return item.StatusLabel
	}
	return fmt.Sprintf("未知状态（%d）", item.StatusCode)
}

func statusTone(item Item) Tone {
	switch item.Type {
	case EventHitokotoAppended, EventHitokotoPollCreated:
		return ToneNeutral
	}
	switch item.StatusCode {
	case statusApproved:
		return ToneSuccess
	case statusRejected:
		return ToneDanger
	case statusNeedModify:
		return ToneWarning
	default:
		return ToneNeutral
	}
}

// combinedLabel 是「窗口内提交并立刻出结果」的合并徽章文案。
// 未发生折叠时返回空串，模板会退回展示普通状态徽章。
func combinedLabel(item Item) string {
	if !item.Collapsed {
		return ""
	}
	switch item.StatusCode {
	case statusApproved:
		return "提交并即时入库"
	case statusRejected:
		return "提交并即时驳回"
	case statusNeedModify:
		return "提交并即时标记为亟待修改"
	default:
		return "提交并即时处理"
	}
}
