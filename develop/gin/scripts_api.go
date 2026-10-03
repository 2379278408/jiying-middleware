package gin

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/utils"
)

func initScriptsAPI() {
	g := core.Server.Group("/scripts")
	g.Use(AuthRequired(), requireConsoleRole(core.RoleOps))

	g.GET("/:name", getScriptContent)
	g.POST("", saveScript)
	g.DELETE("", deleteScript)
}

func scriptTarget(c *gin.Context, name string) (full, rel string, ok bool) {
	sub := strings.TrimSpace(c.Query("path"))
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.Contains(sub, "..") {
		return "", "", false
	}
	rel = name
	if sub != "" && sub != "/" {
		rel = sub + "/" + name
	}
	full = filepath.Join(utils.GetResHome(), "scripts", filepath.FromSlash(rel))
	return full, rel, true
}

func getScriptContent(c *gin.Context) {
	full, _, ok := scriptTarget(c, c.Param("name"))
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		jsonFailMsg(c, "读取失败")
		return
	}
	jsonOK(c, string(data))
}

type saveScriptReq struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

func saveScript(c *gin.Context) {
	var req saveScriptReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Filename == "" {
		jsonFailMsg(c, "参数错误")
		return
	}
	if strings.Contains(req.Filename, "..") || strings.Contains(req.Path, "..") {
		jsonFailMsg(c, "非法路径")
		return
	}
	dir := filepath.Join(utils.GetResHome(), "scripts")
	rel := req.Filename
	if p := strings.TrimSpace(req.Path); p != "" && p != "/" {
		dir = filepath.Join(dir, filepath.FromSlash(p))
		rel = p + "/" + req.Filename
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		jsonFailMsg(c, "目录创建失败")
		return
	}
	full := filepath.Join(dir, req.Filename)
	if err := os.WriteFile(full, []byte(req.Content), 0o644); err != nil {
		jsonFailMsg(c, "写入失败")
		return
	}
	if err := core.ReloadScript(rel); err != nil {
		core.GetMyLog().Warn("脚本热加载失败 %s: %v", rel, err)
	}
	jsonOK(c, rel)
}

func deleteScript(c *gin.Context) {
	var req saveScriptReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Filename == "" {
		jsonFailMsg(c, "参数错误")
		return
	}
	if strings.Contains(req.Filename, "..") || strings.Contains(req.Path, "..") {
		jsonFailMsg(c, "非法路径")
		return
	}
	rel := req.Filename
	if p := strings.TrimSpace(req.Path); p != "" && p != "/" {
		rel = p + "/" + req.Filename
	}
	full := filepath.Join(utils.GetResHome(), "scripts", filepath.FromSlash(rel))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		jsonFailMsg(c, "删除失败")
		return
	}
	core.UnloadScript(rel)
	jsonOK(c, rel)
}
