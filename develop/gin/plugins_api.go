package gin

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/utils"
)

func initPluginsAPI() {
	g := core.Server.Group("/js")
	g.Use(AuthRequired())

	g.GET("/menu", getJsMenuReal)
	g.GET("/content", getJsContentReal)
	g.POST("/content", requireConsoleRole(core.RoleOps), saveJsContent)
	g.GET("/toggle", requireConsoleRole(core.RoleOps), toggleJsPlugin)
	g.GET("/debug", requireConsoleRole(core.RoleOps), debugJsPlugin)
	g.POST("/debug", requireConsoleRole(core.RoleOps), debugJsPluginPost)
}

func pluginLangExt(language string) string {
	if strings.EqualFold(language, "coffee") {
		return ".coffee"
	}
	return ".js"
}

func pluginFunction(full string) *core.Function {
	for _, f := range core.FunctionsSnapshot() {
		if f != nil && f.PluginPath == full {
			return f
		}
	}
	return nil
}

func getJsMenuReal(c *gin.Context) {
	ext := pluginLangExt(c.Query("language"))
	dir := filepath.Join(utils.GetResHome(), "scripts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonOK(c, []interface{}{})
		return
	}
	out := []map[string]interface{}{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ext {
			continue
		}
		disable := false
		if f := pluginFunction(filepath.Join(dir, e.Name())); f != nil {
			disable = f.Disable
		}
		out = append(out, map[string]interface{}{
			"name":    e.Name(),
			"disable": disable,
		})
	}
	jsonOK(c, out)
}

func getJsContentReal(c *gin.Context) {
	name := filepath.Base(strings.TrimSpace(c.Query("name")))
	if name == "" || name == "." || name == "/" {
		jsonFailMsg(c, "参数错误")
		return
	}
	full := filepath.Join(utils.GetResHome(), "scripts", name)
	data, err := os.ReadFile(full)
	if err != nil {
		jsonFailMsg(c, "读取失败")
		return
	}
	jsonOK(c, string(data))
}

type saveJsContentReq struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Content  string `json:"content"`
}

func saveJsContent(c *gin.Context) {
	var req saveJsContentReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		jsonFailMsg(c, "参数错误")
		return
	}
	name := filepath.Base(strings.TrimSpace(req.Name))
	dir := filepath.Join(utils.GetResHome(), "scripts")
	_ = os.MkdirAll(dir, 0o755)
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte(req.Content), 0o644); err != nil {
		jsonFailMsg(c, "写入失败")
		return
	}
	if err := core.ReloadScript(name); err != nil {
		core.GetMyLog().Warn("插件热加载失败 %s: %v", name, err)
	}
	jsonOK(c, name)
}

func toggleJsPlugin(c *gin.Context) {
	name := filepath.Base(strings.TrimSpace(c.Query("name")))
	full := filepath.Join(utils.GetResHome(), "scripts", name)
	f := pluginFunction(full)
	if f == nil {
		if err := core.ReloadScript(name); err != nil {
			jsonFailMsg(c, "插件加载失败")
			return
		}
		f = pluginFunction(full)
		if f == nil {
			jsonFailMsg(c, "插件不存在")
			return
		}
	}
	f.Disable = !f.Disable
	content, _ := os.ReadFile(full)
	jsonOK(c, gin.H{"name": name, "disable": f.Disable, "content": string(content)})
}

type debugJsReq struct {
	Language   string                 `json:"language"`
	Lang       string                 `json:"lang"`
	Name       string                 `json:"name"`
	Code       string                 `json:"code"`
	PluginType string                 `json:"pluginType"`
	Input      map[string]interface{} `json:"input"`
}

func debugJsPluginPost(c *gin.Context) {
	var req debugJsReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		jsonFailMsg(c, "调试代码为空")
		return
	}
	out, err := core.DebugScript(req.Code)
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	w := c.Writer
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line != "" {
			_, _ = w.WriteString(line + "\n")
		}
	}
	if err != nil {
		_, _ = w.WriteString("调试错误: " + err.Error() + "\n")
	}
	w.Flush()
}

func debugJsPlugin(c *gin.Context) {
	name := filepath.Base(strings.TrimSpace(c.Query("name")))
	if err := core.ReloadScript(name); err != nil {
		jsonOK(c, "调试执行失败: "+err.Error())
		return
	}
	jsonOK(c, "插件已重新执行")
}
