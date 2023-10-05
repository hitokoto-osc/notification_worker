package django

import (
	"github.com/flosch/pongo2/v6"
	"github.com/golang-module/carbon/v2"
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/consts"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path"
)

type Context = pongo2.Context

var (
	// instance is the pongo2 template set instance.
	instance *pongo2.TemplateSet
)

// init initializes pongo2 template loader.
// It loads templates from embeddedFS and local filesystem.
// It also loads globals from globalsCtxProviders.
func init() {
	defer zap.L().Sync()
	var err error
	tplPublicDir := path.Join(must[string](executablePath), "resources/")
	loader, err := pongo2.NewHttpFileSystemLoader(http.FS(priorityFS{
		embeddedFS,
		os.DirFS(tplPublicDir),
	}), "template")
	if err != nil {
		zap.L().Fatal("failed to init pongo2 template loader", zap.Error(err))
	}
	instance = pongo2.NewSet("django", loader)
	// copy globals at init
	config.RegisterCallback(func() {
		instance.Debug = config.Debug()
		defaultContext := pongo2.Context{
			"app": pongo2.Context{
				"name":     consts.SiteName,
				"url":      consts.SiteURL,
				"version":  config.Version,
				"logo_url": consts.Logo,
				// footer
				"copyright": consts.Copyright,
			},
		}
		instance.Globals = lo.Assign(instance.Globals, defaultContext, globalsCtxProviders)
	})
}

// runtimeGlobals return runtime globals to pongo2 template runtime, because globals may be changed at runtime,
// such as language, time, tracing.
// This function should be called and injected before rendering a template.
func runtimeGlobals() Context {
	runtimeCtx := Context{
		"app": Context{
			"year": carbon.Now().Format("Y"),
		},
		"today": carbon.Now().Format("Y 年 n 月 j 日"),
	}
	return runtimeCtx
}

// GetTemplate returns a template from cache.
// Note that this method do not inject and return runtime globals.
func GetTemplate(filename string) (*pongo2.Template, error) {
	return instance.FromCache(filename + ".django")
}

// GetTemplateR returns a template from cache and runtime ctx.
// Note that you should inject runtime globals before rendering a template.
func GetTemplateR(filename string) (*pongo2.Template, Context, error) {
	tpl, err := GetTemplate(filename)
	return tpl, runtimeGlobals(), err
}

// RenderTemplate renders a template with context.
// This function is a shortcut of GetTemplate().Execute().
func RenderTemplate(name string, ctx Context) (string, error) {
	tpl, err := GetTemplate(name)
	if err != nil {
		return "", err
	}
	mergedCtx := runtimeGlobals()
	CopyPongoContextRecursive(mergedCtx, ctx)
	return tpl.Execute(mergedCtx)
}
