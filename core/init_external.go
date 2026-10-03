package core

import (
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"jiying/middleware/utils"
)

var headerLength = 0

type RegisterExternalPlugin func(f *Function)

var externalPluginHook RegisterExternalPlugin

func InitExternalPlugins() {
	DelOriginCommand()

	loadExternalPlugins()

	registerExternalCommands()
	addRoutes()
}

func loadExternalPlugins() {
	dir := filepath.Join(utils.GetResHome(), "plugin", "encrypted_ext")
	paths := utils.ReadFilePaths(dir)
	if len(paths) == 0 {
		return
	}
	for _, p := range paths {
		if utils.IsDir(p) {
			continue
		}
		registerExternalPlugin(p)
	}
}

func registerExternalPlugin(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		GetMyLog().Warn("外部插件读取失败 %s: %v", path, err)
		return
	}
	if len(data) < 4 {
		GetMyLog().Warn("外部插件文件过小 %s", path)
		return
	}

	lang := scriptLangOf(path)
	if lang == "" {
		lang = "js"
	}
	f := &Function{
		PluginPath:    path,
		PluginContent: string(data),
		PluginLang:    lang,
		Title:         scriptTitle(path),
		Origin:        false,
	}
	if externalPluginHook != nil {
		externalPluginHook(f)
	}
	appendFunction(f)
	GetMyLog().Info("外部插件登记 %s", path)

	if lang == "js" && ScriptEngine != nil {
		_ = runScript(f)
	}
}

var routesRegistered atomic.Bool

func registerExternalCommands() {
	funcs := FunctionsSnapshot()
	if len(funcs) == 0 {
		return
	}
	_ = AddCommand(5, funcs...)
}

func addRoutes() {
	if Server == nil {
		return
	}
	if !routesRegistered.CompareAndSwap(false, true) {
		return
	}
	extGroup := Server.Group("/ext")
	extGroup.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 200, "message": "pong"})
	})
	GetMyLog().Warn("外部插件路由 /ext/ping 已注册")
}
