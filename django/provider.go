package django

import (
	"strings"
)

var globalsCtxProviders = make(Context)

func SetGlobals(path string, value any) {
	keys := strings.Split(path, ".")
	if len(keys) == 0 {
		panic("invalid path")
	}
	var o = globalsCtxProviders
	for i, v := range keys {
		if i == len(keys)-1 {
			o[v] = value
			return
		}
		var (
			t  Context
			ok bool
		)
		t, ok = o[v].(Context)
		if !ok {
			o[v] = make(Context)
		} else {
			o = t
		}
	}
}

func GetGlobals() Context {
	return globalsCtxProviders
}
