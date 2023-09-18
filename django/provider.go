package django

import (
	"github.com/flosch/pongo2/v6"
	"strings"
)

var globalsCtxProviders = make(pongo2.Context)

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
			t  pongo2.Context
			ok bool
		)
		t, ok = o[v].(pongo2.Context)
		if !ok {
			o[v] = make(pongo2.Context)
		} else {
			o = t
		}
	}
}

func GetGlobals() pongo2.Context {
	return globalsCtxProviders
}
