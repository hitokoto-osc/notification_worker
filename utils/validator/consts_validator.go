package validator

import (
	"github.com/hitokoto-osc/notification-worker/consts"
	"reflect"
)

func validateHitokotoTypeValue(v reflect.Value) any {
	if value, ok := v.Interface().(consts.HitokotoType); ok {
		switch value {
		case consts.HitokotoTypeAnime,
			consts.HitokotoTypeComic,
			consts.HitokotoTypeGame,
			consts.HitokotoTypeLiterature,
			consts.HitokotoTypeOriginal,
			consts.HitokotoTypeInternet,
			consts.HitokotoTypeOther,
			consts.HitokotoTypeVideo,
			consts.HitokotoTypePoetry,
			consts.HitokotoTypeNetEase,
			consts.HitokotoTypePhilosophy,
			consts.HitokotoTypeJoke:
			return string(value)
		default:
			return nil
		}
	}
	return nil
}

func validatePollMethodValue(v reflect.Value) any {
	if value, ok := v.Interface().(consts.PollMethod); ok {
		switch value {
		case consts.PollMethodApprove, consts.PollMethodReject, consts.PollMethodNeedCommonUserPoll, consts.PollMethodNeedModify:
			return int(value)
		default:
			return nil
		}
	}
	return nil
}

func validatePollStatusValue(v reflect.Value) any {
	if value, ok := v.Interface().(consts.PollStatus); ok {
		switch value {
		case consts.PollStatusUnknown,
			consts.PollStatusNotOpen,
			consts.PollStatusOpen,
			consts.PollStatusProcessing,
			consts.PollStatusSuspended,
			consts.PollStatusClosed,
			consts.PollStatusOpenForCommonUser,
			consts.PollStatusApproved,
			consts.PollStatusRejected,
			consts.PollStatusNeedModify:
			return int(value)
		default:
			return nil
		}
	}
	return nil
}
