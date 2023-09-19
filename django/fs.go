package django

import (
	"go.uber.org/zap"
	"io/fs"
	"reflect"
)

type priorityFS []fs.FS

func (df priorityFS) Open(name string) (fs.File, error) {
	defer zap.L().Sync()
	for i := len(df) - 1; i >= 0; i-- {
		cf := df[i]
		zap.L().Debug("trying to open file", zap.String("name", name), zap.String("fs", reflect.TypeOf(cf).String()))
		f, err := cf.Open(name)
		if err == nil {
			return f, err
		}
	}
	return nil, fs.ErrNotExist
}
