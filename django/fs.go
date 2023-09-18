package django

import "io/fs"

type priorityFS []fs.FS

func (df priorityFS) Open(name string) (fs.File, error) {
	for i := len(df) - 1; i >= 0; i-- {
		cf := df[i]
		f, err := cf.Open(name)
		if err == nil {
			return f, err
		}
	}
	return nil, fs.ErrNotExist
}
