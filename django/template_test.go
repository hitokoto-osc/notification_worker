package django

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 所有模板都必须能编译。语法错误（例如 pongo2 不支持的跨行 {# #} 注释）
// 只有在真正渲染时才会暴露，这里提前拦下。
func TestAllTemplatesCompile(t *testing.T) {
	files, err := getAllFiles(embeddedFS)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	compiled := 0
	for _, file := range files {
		if !strings.HasSuffix(file, ".django") {
			continue
		}
		// embeddedFS 的根是 template/，而 loader 的 baseDir 已经是 template。
		name := strings.TrimPrefix(path.Clean(file), "template/")
		t.Run(name, func(t *testing.T) {
			_, err := instance.FromCache(name)
			require.NoError(t, err)
		})
		compiled++
	}
	require.NotZero(t, compiled)
}
