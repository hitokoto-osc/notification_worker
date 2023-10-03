package formatter

import "github.com/hitokoto-osc/notification-worker/consts"

// FormatPollMethod formats user's poll method.
func FormatPollMethod(t consts.PollMethod) string {
	switch t {
	case consts.PollMethodApprove:
		return "赞同"
	case consts.PollMethodReject:
		return "驳回"
	case consts.PollMethodNeedModify:
		return "亟待修改"
	case consts.PollMethodNeedCommonUserPoll:
		return "需要普通用户投票"
	default:
		return "未知"
	}
}

// FormatPollStatus formats poll status.
func FormatPollStatus(t consts.PollStatus) string {
	switch t {
	case consts.PollStatusNotOpen:
		return "未开放"
	case consts.PollStatusOpen:
		return "开放"
	case consts.PollStatusProcessing:
		return "处理中"
	case consts.PollStatusSuspended:
		return "暂停"
	case consts.PollStatusClosed:
		return "已关闭"
	case consts.PollStatusOpenForCommonUser:
		return "开放给普通用户投票"
	case consts.PollStatusApproved:
		return "入库"
	case consts.PollStatusRejected:
		return "驳回"
	case consts.PollStatusNeedModify:
		return "亟待修改"
	default:
		return "未知"
	}
}
