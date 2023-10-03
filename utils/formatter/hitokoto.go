package formatter

import "github.com/hitokoto-osc/notification-worker/consts"

func FormatHitokotoType(t consts.HitokotoType) string {
	switch t {
	case consts.HitokotoTypeAnime:
		return "Anime - 动画"
	case consts.HitokotoTypeComic:
		return "Comic – 漫画"
	case consts.HitokotoTypeLiterature:
		return "Literature - 文学"
	case consts.HitokotoTypeGame:
		return "Game - 游戏"
	case consts.HitokotoTypeOriginal:
		return "Original - 原创"
	case consts.HitokotoTypeInternet:
		return "Internet - 来自网络"
	case consts.HitokotoTypeOther:
		return "Other - 其他"
	case consts.HitokotoTypeVideo:
		return "Video - 影视"
	case consts.HitokotoTypePoetry:
		return "Poetry - 古诗词"
	case consts.HitokotoTypeNetEase:
		return "NetEase - 网易云音乐"
	case consts.HitokotoTypePhilosophy:
		return "Philosophy - 哲学"
	case consts.HitokotoTypeJoke:
		return "Joke - 抖机灵"
	default:
		return "Unknown - 未知"
	}
}
