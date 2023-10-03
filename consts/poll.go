package consts

type PollMethod int

const (
	PollMethodApprove            PollMethod = iota + 1 // 赞同
	PollMethodReject                                   // 驳回
	PollMethodNeedModify                               // 需要修改
	PollMethodNeedCommonUserPoll                       // 需要普通用户投票
)

type PollStatus int

const (
	PollStatusUnknown           PollStatus = -1  // 未知，比如投票不存在
	PollStatusNotOpen           PollStatus = 0   // 未开放投票
	PollStatusOpen              PollStatus = 1   // 投票正常开放
	PollStatusProcessing        PollStatus = 2   // 处理中，停止投票
	PollStatusSuspended         PollStatus = 100 // 暂停投票
	PollStatusClosed            PollStatus = 101 // 关闭投票
	PollStatusOpenForCommonUser PollStatus = 102 // 开放给普通用户投票
	PollStatusApproved          PollStatus = 200 // 赞同
	PollStatusRejected          PollStatus = 201 // 驳回
	PollStatusNeedModify        PollStatus = 202 // 需要修改
)
