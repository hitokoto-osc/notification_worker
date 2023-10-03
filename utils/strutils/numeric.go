package strutils

import (
	"github.com/samber/lo"
	"strconv"
)

func IsInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func IsFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func IsNumeric(s string) bool {
	return IsInteger(s) || IsFloat(s)
}

func MustInt(s string) int {
	return lo.Must(strconv.Atoi(s))
}

func MustInt64(s string) int64 {
	return lo.Must(strconv.ParseInt(s, 10, 64))
}
