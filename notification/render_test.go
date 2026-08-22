package notification

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "重写 testdata 下的 golden 快照")

// 刻意选一个不会与「当前年份」相同的年份，避免归一化误伤条目时间戳。
var baseTime = time.Date(2019, 8, 22, 10, 0, 0, 0, time.UTC)

func at(offset time.Duration) time.Time { return baseTime.Add(offset) }

func strptr(s string) *string { return &s }

// normalizeGolden 抹掉渲染结果中依赖当前时间的部分（布局里的签名日期与页脚年份），
// 否则快照每天都会失效。
func normalizeGolden(html string) string {
	html = strings.ReplaceAll(html, carbon.Now().Format("Y 年 n 月 j 日"), "<TODAY>")
	// 只替换页脚版权处的年份，不要碰条目时间戳。
	html = strings.ReplaceAll(html, "© "+carbon.Now().Format("Y")+" ", "© <YEAR> ")
	return html
}

func assertGolden(t *testing.T, name, html string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden.html")
	got := normalizeGolden(html)
	if *updateGolden {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "golden 缺失，用 -update 生成：%s", path)
	assert.Equal(t, string(want), got, "渲染结果与 golden 不一致；确认变更符合预期后用 -update 更新")
}

func contributorFixture() Digest {
	return Digest{
		Group:          GroupContributor,
		RecipientName:  "测试用户",
		WindowDuration: "3 分钟",
		ActionURL:      "https://hitokoto.cn/dashboard",
		ActionText:     "前往创作者中心",
		Items: []Item{
			{
				EventID: "e-appended-1", Type: EventHitokotoAppended, SentenceUUID: "u1",
				Hitokoto: "落霞与孤鹜齐飞", From: "滕王阁序", FromWho: strptr("王勃"),
				TypeLabel: "Poetry - 古诗词", OccurredAt: at(0), Creator: "测试用户",
			},
			{
				EventID: "e-reviewed-1", Type: EventHitokotoReviewed, SentenceUUID: "u1",
				Hitokoto: "落霞与孤鹜齐飞", From: "滕王阁序", FromWho: strptr("王勃"),
				TypeLabel: "Poetry - 古诗词", StatusCode: statusApproved, StatusLabel: "入库",
				OccurredAt: at(time.Minute), Creator: "测试用户",
			},
			{
				EventID: "e-reviewed-2", Type: EventHitokotoReviewed, SentenceUUID: "u2",
				Hitokoto: `<script>alert("xss")</script> & 「引号」`, From: "未知 & 来源", FromWho: nil,
				TypeLabel: "Other - 其他", StatusCode: statusRejected, StatusLabel: "驳回",
				OccurredAt: at(2 * time.Minute), Creator: "测试用户",
			},
			{
				EventID: "e-moved-1", Type: EventHitokotoMoved, SentenceUUID: "u3",
				Hitokoto: "海内存知己", From: "送杜少府之任蜀州", FromWho: strptr("王勃"),
				TypeLabel: "Poetry - 古诗词", StatusCode: statusNeedModify, StatusLabel: "亟待修改",
				OccurredAt: at(3 * time.Minute), Creator: "测试用户",
			},
			{
				EventID: "e-appended-2", Type: EventHitokotoAppended, SentenceUUID: "u4",
				Hitokoto: "山有木兮木有枝", From: "越人歌", FromWho: nil,
				TypeLabel: "Literature - 文学", OccurredAt: at(4 * time.Minute), Creator: "测试用户",
			},
		},
	}
}

func reviewerFixture() Digest {
	return Digest{
		Group:          GroupReviewer,
		RecipientName:  "审核员喵",
		WindowDuration: "5 分钟",
		ActionURL:      "https://reviewer.hitokoto.cn/dashboard/do_review",
		ActionText:     "前往审核员工作台",
		Items: []Item{
			{
				EventID: "p-created-1", Type: EventHitokotoPollCreated, SentenceUUID: "u1", PollID: 1024,
				Hitokoto: "云想衣裳花想容", From: "清平调", FromWho: strptr("李白"),
				TypeLabel: "Poetry - 古诗词", OccurredAt: at(0), Creator: "alice",
			},
			{
				EventID: "p-created-2", Type: EventHitokotoPollCreated, SentenceUUID: "u2", PollID: 1025,
				Hitokoto: "此情可待成追忆", From: "锦瑟", FromWho: nil,
				TypeLabel: "Poetry - 古诗词", OccurredAt: at(time.Minute), Creator: "bob",
			},
			{
				EventID: "p-finished-1", Type: EventHitokotoPollFinished, SentenceUUID: "u3", PollID: 1000,
				Hitokoto: "人生若只如初见", From: "木兰花令", FromWho: strptr("纳兰性德"),
				TypeLabel: "Poetry - 古诗词", StatusCode: statusApproved, StatusLabel: "入库",
				MethodLabel: "赞同", Point: 2, OccurredAt: at(2 * time.Minute), Creator: "carol",
			},
		},
	}
}

func TestRenderContributorDigestGolden(t *testing.T) {
	result, err := Render(contributorFixture())
	require.NoError(t, err)
	require.Equal(t, RenderDigest, result.Kind)

	assert.Equal(t, "喵！您有 4 条一言新动态（1 条已入库，2 条需处理）", result.Subject)
	assert.Equal(t, Metrics{EventCount: 5, Total: 4, Approved: 1, Rejected: 1, NeedModify: 1, Attention: 2, Pending: 1}, result.Metrics)
	assertGolden(t, "contributor_mixed", result.HTML)
}

func TestRenderReviewerDigestGolden(t *testing.T) {
	result, err := Render(reviewerFixture())
	require.NoError(t, err)
	require.Equal(t, RenderDigest, result.Kind)

	assert.Equal(t, "喵！您有 3 条审核动态（2 条待审，1 条已结算）", result.Subject)
	assertGolden(t, "reviewer_mixed", result.HTML)
}

func TestRenderOverflowGolden(t *testing.T) {
	digest := Digest{
		Group:          GroupContributor,
		RecipientName:  "高产用户",
		WindowDuration: "3 分钟",
		ActionURL:      "https://hitokoto.cn/dashboard",
		ActionText:     "前往创作者中心",
		DisplayCap:     3,
	}
	for i := range 10 {
		digest.Items = append(digest.Items, Item{
			EventID:      fmt.Sprintf("e-%02d", i),
			Type:         EventHitokotoAppended,
			SentenceUUID: fmt.Sprintf("u-%02d", i),
			Hitokoto:     fmt.Sprintf("第 %d 条句子", i),
			From:         "来源",
			TypeLabel:    "Original - 原创",
			OccurredAt:   at(time.Duration(i) * time.Minute),
			Creator:      "高产用户",
		})
	}

	result, err := Render(digest)
	require.NoError(t, err)
	assert.Equal(t, "喵！已成功收到您提交的 10 条新句子！", result.Subject)
	assert.Contains(t, result.HTML, "另有 <b>7</b> 条未在邮件中展开")
	assertGolden(t, "contributor_overflow", result.HTML)
}

// 宏未被导入时 pongo2 会渲染成空字符串且不报错（{% import %} 写在 block 外是
// 最容易踩的一种）。这里断言宏确实产出了标记，守住这条静默失效路径。
func TestRenderEmitsMacroMarkup(t *testing.T) {
	for name, digest := range map[string]Digest{
		"contributor": contributorFixture(),
		"reviewer":    reviewerFixture(),
	} {
		t.Run(name, func(t *testing.T) {
			result, err := Render(digest)
			require.NoError(t, err)

			assert.Contains(t, result.HTML, "border-radius:12px", "统计胶囊宏未产出标记")
			assert.Contains(t, result.HTML, "border-left:4px solid", "分节标题宏未产出标记")
			assert.Contains(t, result.HTML, "background-color:#f8fafc", "条目行宏未产出标记")
			assert.Contains(t, result.HTML, "class=\"button button-primary\"", "CTA 按钮未渲染")
		})
	}
}

func TestRenderEscapesUserContent(t *testing.T) {
	result, err := Render(contributorFixture())
	require.NoError(t, err)

	assert.NotContains(t, result.HTML, "<script>alert")
	assert.Contains(t, result.HTML, "&lt;script&gt;alert")
	assert.Contains(t, result.HTML, "未知 &amp; 来源")
}

func TestRenderNilFromWhoFallsBackToPlaceholder(t *testing.T) {
	result, err := Render(contributorFixture())
	require.NoError(t, err)
	assert.Contains(t, result.HTML, "未填写")
}

func TestRenderSingleItemFallsBackToSingleTemplate(t *testing.T) {
	digest := contributorFixture()
	digest.Items = digest.Items[:1]

	result, err := Render(digest)
	require.NoError(t, err)
	assert.Equal(t, RenderSingle, result.Kind)
	require.NotNil(t, result.Single)
	assert.Equal(t, "e-appended-1", result.Single.EventID)
	assert.Empty(t, result.HTML)
	assert.Empty(t, result.Subject)
}

// 提交后立刻出审核结果：输入仍是两条事件，应发摘要并展示合并徽章。
// 回退单条模板只看输入事件数——调用方手上没有重建单条模板所需的审核员信息。
func TestRenderUsesDigestWhenTwoEventsCollapseToOneRow(t *testing.T) {
	digest := contributorFixture()
	digest.Items = digest.Items[:2]

	result, err := Render(digest)
	require.NoError(t, err)
	assert.Equal(t, RenderDigest, result.Kind)
	assert.Nil(t, result.Single)
	assert.Contains(t, result.HTML, "提交并即时入库")
	assert.Equal(t, 2, result.Metrics.EventCount)
	assert.Equal(t, 1, result.Metrics.Total)
}

func TestRenderOmitsUnsafeActionURL(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "", "not a url", "ftp://example.com/x"} {
		t.Run(raw, func(t *testing.T) {
			digest := contributorFixture()
			digest.ActionURL = raw

			result, err := Render(digest)
			require.NoError(t, err)
			assert.NotContains(t, result.HTML, "javascript:")
			assert.NotContains(t, result.HTML, `class="button button-primary"`, "非法 action_url 时不应渲染 CTA")
		})
	}
}

// 终态缺少文案时必须有兜底，否则条目会落在「需要您关注」里却没有任何说明。
func TestRenderLabelsUnknownTerminalStatus(t *testing.T) {
	digest := contributorFixture()
	digest.Items[2].StatusCode = 999
	digest.Items[2].StatusLabel = ""

	result, err := Render(digest)
	require.NoError(t, err)
	assert.Contains(t, result.HTML, "未知状态（999）")
	assert.Equal(t, 2, result.Metrics.Attention)
}

func TestRenderRejectsMismatchedGroup(t *testing.T) {
	digest := contributorFixture()
	digest.Group = GroupReviewer

	_, err := Render(digest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "与摘要分组")
}

func TestRenderRejectsEmptyDigest(t *testing.T) {
	_, err := Render(Digest{Group: GroupContributor})
	require.Error(t, err)
}

func TestRenderRejectsUnknownEventType(t *testing.T) {
	digest := contributorFixture()
	digest.Items[0].Type = "hitokoto_unknown"

	_, err := Render(digest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "事件类型未知")
}

func TestCollapseFoldsAppendedIntoTerminal(t *testing.T) {
	items := []Item{
		{EventID: "reviewed", Type: EventHitokotoReviewed, SentenceUUID: "u1", StatusCode: statusApproved, OccurredAt: at(time.Minute)},
		{EventID: "appended", Type: EventHitokotoAppended, SentenceUUID: "u1", OccurredAt: at(0)},
	}
	got := Collapse(items)

	require.Len(t, got, 1)
	assert.Equal(t, "reviewed", got[0].EventID)
	assert.True(t, got[0].Collapsed)
	assert.Equal(t, "提交并即时入库", combinedLabel(got[0]))
	assert.False(t, items[0].Collapsed, "Collapse 不得修改入参")
}

// 只有「提交」这一条是冗余的。先入库、后被管理员移动至驳回是两次用户可见的
// 状态变迁，折叠掉前一条会把事件流变成状态快照。
func TestCollapseDropsOnlyTheAppendedEvent(t *testing.T) {
	items := []Item{
		{EventID: "appended", Type: EventHitokotoAppended, SentenceUUID: "u1", OccurredAt: at(0)},
		{EventID: "reviewed", Type: EventHitokotoReviewed, SentenceUUID: "u1", StatusCode: statusApproved, OccurredAt: at(time.Minute)},
		{EventID: "moved", Type: EventHitokotoMoved, SentenceUUID: "u1", StatusCode: statusRejected, OccurredAt: at(2 * time.Minute)},
	}
	got := Collapse(items)

	require.Len(t, got, 2)
	assert.Equal(t, "reviewed", got[0].EventID)
	assert.True(t, got[0].Collapsed, "合并徽章打在紧跟提交的那条终态上")
	assert.Equal(t, "提交并即时入库", combinedLabel(got[0]))
	assert.Equal(t, "moved", got[1].EventID)
	assert.False(t, got[1].Collapsed)
}

// 窗口内没有提交事件：句子是更早提交的，两条终态都要保留且都不打合并徽章。
func TestCollapsePreservesTerminalsWithoutAppended(t *testing.T) {
	items := []Item{
		{EventID: "reviewed", Type: EventHitokotoReviewed, SentenceUUID: "u1", StatusCode: statusApproved, OccurredAt: at(0)},
		{EventID: "moved", Type: EventHitokotoMoved, SentenceUUID: "u1", StatusCode: statusRejected, OccurredAt: at(time.Minute)},
	}
	got := Collapse(items)

	require.Len(t, got, 2)
	assert.Equal(t, "reviewed", got[0].EventID)
	assert.Equal(t, "moved", got[1].EventID)
	assert.False(t, got[0].Collapsed)
	assert.False(t, got[1].Collapsed)
	assert.Empty(t, combinedLabel(got[0]))
}

func TestCollapseKeepsDistinctSentencesApart(t *testing.T) {
	items := []Item{
		{EventID: "a1", Type: EventHitokotoAppended, SentenceUUID: "u1", OccurredAt: at(0)},
		{EventID: "a2", Type: EventHitokotoAppended, SentenceUUID: "u2", OccurredAt: at(time.Minute)},
		{EventID: "r2", Type: EventHitokotoReviewed, SentenceUUID: "u2", StatusCode: statusApproved, OccurredAt: at(2 * time.Minute)},
	}
	got := Collapse(items)

	require.Len(t, got, 2)
	assert.Equal(t, "a1", got[0].EventID)
	assert.Equal(t, "r2", got[1].EventID)
}

func TestCollapseSortsOutOfOrderArrivals(t *testing.T) {
	items := []Item{
		{EventID: "c", Type: EventHitokotoAppended, SentenceUUID: "u3", OccurredAt: at(2 * time.Minute)},
		{EventID: "a", Type: EventHitokotoAppended, SentenceUUID: "u1", OccurredAt: at(0)},
		{EventID: "b", Type: EventHitokotoAppended, SentenceUUID: "u2", OccurredAt: at(time.Minute)},
	}
	got := Collapse(items)

	require.Len(t, got, 3)
	assert.Equal(t, []string{"a", "b", "c"}, []string{got[0].EventID, got[1].EventID, got[2].EventID})
}

func TestBuildSubject(t *testing.T) {
	tests := []struct {
		name    string
		group   DigestGroup
		metrics Metrics
		want    string
	}{
		{"纯提交", GroupContributor, Metrics{Total: 3, Pending: 3}, "喵！已成功收到您提交的 3 条新句子！"},
		{"混合", GroupContributor, Metrics{Total: 5, Approved: 2, NeedModify: 1, Attention: 1}, "喵！您有 5 条一言新动态（2 条已入库，1 条需处理）"},
		{"全部驳回", GroupContributor, Metrics{Total: 4, Rejected: 4, Attention: 4}, "喵！您有 4 条一言新动态（0 条已入库，4 条需处理）"},
		{"纯新投票", GroupReviewer, Metrics{Total: 4, PollCreated: 4}, "喵！审核队列刷新了 4 条新投票待您审阅！"},
		{"纯结算", GroupReviewer, Metrics{Total: 2, PollFinished: 2}, "喵！您有 2 条投票已结束，结算结果出炉！"},
		{"审核混合", GroupReviewer, Metrics{Total: 3, PollCreated: 2, PollFinished: 1}, "喵！您有 3 条审核动态（2 条待审，1 条已结算）"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildSubject(tt.group, tt.metrics))
		})
	}
}

func TestBuildSectionsDropsEmptyAndTruncates(t *testing.T) {
	items := make([]Item, 5)
	got := buildSections(3,
		section{Title: "空", Items: nil},
		section{Title: "满", Items: items},
	)

	require.Len(t, got, 1)
	assert.Equal(t, "满", got[0].Title)
	assert.Len(t, got[0].Items, 3)
	assert.Equal(t, 2, got[0].Overflow)
}

func TestGroupOf(t *testing.T) {
	for _, tt := range []struct {
		event EventType
		group DigestGroup
	}{
		{EventHitokotoAppended, GroupContributor},
		{EventHitokotoReviewed, GroupContributor},
		{EventHitokotoMoved, GroupContributor},
		{EventHitokotoPollCreated, GroupReviewer},
		{EventHitokotoPollFinished, GroupReviewer},
	} {
		group, ok := GroupOf(tt.event)
		require.True(t, ok, tt.event)
		assert.Equal(t, tt.group, group)
	}

	_, ok := GroupOf("hitokoto_poll_daily_report")
	assert.False(t, ok)
}

// Gmail 在 102KB 处截断正文。用最坏情况（三个分节各占满展示上限、内容为长句）
// 验证摘要邮件不会触及该阈值。
func TestRenderStaysUnderGmailClipThreshold(t *testing.T) {
	const gmailClipBytes = 102 * 1024

	longSentence := strings.Repeat("这是一条相当长的句子内容用于压测邮件体积", 5)
	digest := Digest{
		Group:          GroupContributor,
		RecipientName:  "压测用户",
		WindowDuration: "15 分钟",
		ActionURL:      "https://hitokoto.cn/dashboard",
		ActionText:     "前往创作者中心",
	}
	// 每节都占满 DefaultDisplayCap（各自再多 5 条以触发溢出）：驳回 / 入库 / 待审。
	statuses := []struct {
		event  EventType
		code   int
		label  string
		suffix string
	}{
		{EventHitokotoReviewed, statusRejected, "驳回", "r"},
		{EventHitokotoReviewed, statusApproved, "入库", "a"},
		{EventHitokotoAppended, 0, "", "p"},
	}
	for _, s := range statuses {
		for i := range DefaultDisplayCap + 5 {
			digest.Items = append(digest.Items, Item{
				EventID:      fmt.Sprintf("e-%s-%02d", s.suffix, i),
				Type:         s.event,
				SentenceUUID: fmt.Sprintf("u-%s-%02d", s.suffix, i),
				Hitokoto:     longSentence,
				From:         "某部相当长的作品名称用于压测",
				FromWho:      strptr("某位名字也不算短的作者"),
				TypeLabel:    "Literature - 文学",
				StatusCode:   s.code,
				StatusLabel:  s.label,
				OccurredAt:   at(time.Duration(i) * time.Minute),
				Creator:      "压测用户",
			})
		}
	}

	result, err := Render(digest)
	require.NoError(t, err)
	require.Equal(t, RenderDigest, result.Kind)

	size := len(result.HTML)
	t.Logf("最坏情况摘要体积：%d 字节（阈值 %d）", size, gmailClipBytes)
	assert.Less(t, size, gmailClipBytes, "摘要邮件体积逼近 Gmail 截断阈值，需要收紧每节展示上限或行标记")
}
