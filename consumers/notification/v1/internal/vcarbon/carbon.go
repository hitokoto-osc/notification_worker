package vcarbon

import (
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/golang-module/carbon/v2"
	"github.com/hitokoto-osc/notification-worker/utils/strutils"
)

type Carbon struct {
	carbon.Carbon
}

// InvalidCarbonParseParameterError 无效的 Carbon 解析参数错误
// 用于在 JSON 解析时，当传入的参数无法被解析为 Carbon 时返回
// TODO: 未来可以考虑将这个错误抽象到 utils/errors 中
// TODO: 当 encoding/json/v2 发布后，可以直接抛弃这个错误，使用 encoding/json/v2 中的错误
type InvalidCarbonParseParameterError struct {
	Parameter  string
	Msg        string
	innerError error
}

func (e *InvalidCarbonParseParameterError) Error() string {
	return fmt.Sprintf("invalid carbon parse parameter: %s, %s", e.Parameter, e.Msg)
}

func (e *InvalidCarbonParseParameterError) Unwrap() error {
	return e.innerError
}

func NewFromCarbon(c carbon.Carbon) *Carbon {
	return &Carbon{c}
}

// UnmarshalJSON 重写 carbon.Carbon 的 JSON 解析方法
// 1. 兼容老版本传输的字符串版本的毫秒、秒时间戳
// 2. 支持新版本传输的 ISO 时间字符串
func (p *Carbon) UnmarshalJSON(data []byte) error {
	var str string
	err := sonic.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if strutils.IsInteger(str) { // Maybe timestamp
		length := len(str)
		var inner carbon.Carbon
		switch length {
		case 10: // 秒级时间戳
			inner = carbon.CreateFromTimestamp(strutils.MustInt64(str))
		case 13: // 毫秒级时间戳
			inner = carbon.CreateFromTimestampMilli(strutils.MustInt64(str))
		case 16: // 微秒级时间戳
			inner = carbon.CreateFromTimestampMicro(strutils.MustInt64(str))
		case 19: // 纳秒级时间戳
			inner = carbon.CreateFromTimestampNano(strutils.MustInt64(str))
		default:
			return &InvalidCarbonParseParameterError{
				Parameter: str,
				Msg:       "invalid timestamp length",
			}
		}
		*p = *NewFromCarbon(inner)
		return nil
	}
	*p = *NewFromCarbon(carbon.Parse(str))
	if p.Error != nil {
		return &InvalidCarbonParseParameterError{
			Parameter:  str,
			Msg:        p.Error.Error(),
			innerError: p.Error,
		}
	}
	return nil
}
