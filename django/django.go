package django

import (
	"github.com/flosch/pongo2/v6"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path"
)

var (
	// instance is the pongo2 template set instance.
	instance *pongo2.TemplateSet
)

// init initializes pongo2 template loader.
// It loads templates from embededFS and local filesystem.
// It also loads globals from globalsCtxProviders.
func init() {
	defer zap.L().Sync()
	var err error
	tplPublicDir := path.Join(must[string](executablePath), "resources/")
	loader, err := pongo2.NewHttpFileSystemLoader(http.FS(priorityFS{
		embededFS,
		os.DirFS(tplPublicDir),
	}), "./template")
	if err != nil {
		zap.L().Fatal("failed to init pongo2 template loader", zap.Error(err))
	}
	instance = pongo2.NewSet("django", loader)
	// copy globals at init
	instance.Globals = lo.Assign(instance.Globals, globalsCtxProviders)
}

// runtimeInjectGlobals injects globals to pongo2 template runtime, because globals may be changed at runtime,
// such as language, time, tracing.
// This function should be called getTemplate().
func runtimeInjectGlobals() {
	runtimeCtx := pongo2.Context{}
	CopyPongoContextRecursive(instance.Globals, runtimeCtx)
}

// GetTemplate returns a template from cache.
func GetTemplate(filename string) (*pongo2.Template, error) {
	runtimeInjectGlobals()
	return instance.FromCache(filename + ".django")
}

// RenderTemplate renders a template with context.
// This function is a shortcut of GetTemplate().Execute().
func RenderTemplate(name string, ctx pongo2.Context) (string, error) {
	tpl, err := GetTemplate(name)
	if err != nil {
		return "", err
	}
	return tpl.Execute(ctx)
}
