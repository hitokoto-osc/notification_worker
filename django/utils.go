package django

import (
	"github.com/flosch/pongo2/v6"
	"go.uber.org/zap"
	"io/fs"
	"os"
	"path/filepath"
)

func executablePath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Dir(ex)
	return path, nil
}

func must[T any](fn func() (T, error)) T {
	defer zap.L().Sync()
	v, err := fn()
	if err != nil {
		zap.L().Fatal("must: failed to do", zap.Error(err))
	}
	return v
}

func getAllFiles(efs fs.FS) (files []string, err error) {
	if err = fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

// CopyPongoContextRecursive copies all keys from src to dst if dst does not have the key.
// If value of the key is pongo2.Context, it will be copied recursively.
func CopyPongoContextRecursive(dst, src pongo2.Context) {
	for k, v := range src {
		if ctx, ok := v.(pongo2.Context); ok {
			if _, ok = dst[k].(pongo2.Context); !ok {
				dst[k] = make(pongo2.Context)
			}
			CopyPongoContextRecursive(dst[k].(pongo2.Context), ctx)
		} else {
			dst[k] = v
		}
	}
}

func MergeContext(contexts ...Context) Context {
	var ctx = make(Context)
	for _, c := range contexts {
		CopyPongoContextRecursive(ctx, c)
	}
	return ctx
}
