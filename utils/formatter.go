package utils

var hitokotoTypes = map[string]string{
	"a": "Anime - 动画",
	"b": "Comic – 漫画",
	"c": "Game – 游戏",
	"d": "Novel – 小说",
	"e": "Myself – 原创",
	"f": "Internet – 来自网络",
	"g": "Other – 其他",
	"h": "Video – 影视",
	"i": "Poetry – 古诗词",
	"j": "NetEase – 网易云音乐",
	"k": "Philosophy – 哲学",
	"l": "Joke - 抖机灵",
}

func FormatHitokotoType(t string) string {
	if v, ok := hitokotoTypes[t]; ok {
		return v
	}
	return "未知分类"
}
