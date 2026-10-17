package gin

import (
	"os"
	"path/filepath"

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
	root := filepath.Join(utils.GetResHome(), "scripts")
	sub, ok := safeUploadRel(c.Query("path"))
	if !ok {
		return "", "", false
	}
	cleanName, ok := safeUploadName(name, ".js")
	if !ok {
		return "", "", false
	}
	dir, ok := safeJoinUnder(root, sub)
	if !ok {
		return "", "", false
	}
	full, ok = safeJoinUnder(dir, cleanName)
	if !ok || filepath.Dir(full) != dir {
		return "", "", false
	}
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return "", "", false
	}
	return full, filepath.ToSlash(rel), true
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
	root := filepath.Join(utils.GetResHome(), "scripts")
	sub, ok := safeUploadRel(req.Path)
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	dir, ok := safeJoinUnder(root, sub)
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	name, ok := safeUploadName(req.Filename, ".js")
	if !ok {
		jsonFailMsg(c, "非法文件名")
		return
	}
	if len(req.Content) > maxUploadBytes {
		jsonFailMsg(c, "脚本内容超出限制")
		return
	}
	full, ok := safeJoinUnder(dir, name)
	if !ok || filepath.Dir(full) != dir {
		jsonFailMsg(c, "非法路径")
		return
	}
	rel, err := filepath.Rel(root, full)
	if err != nil {
		jsonFailMsg(c, "非法路径")
		return
	}
	rel = filepath.ToSlash(rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		jsonFailMsg(c, "目录创建失败")
		return
	}
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
	root := filepath.Join(utils.GetResHome(), "scripts")
	sub, ok := safeUploadRel(req.Path)
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	dir, ok := safeJoinUnder(root, sub)
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	name, ok := safeUploadName(req.Filename, ".js")
	if !ok {
		jsonFailMsg(c, "非法文件名")
		return
	}
	full, ok := safeJoinUnder(dir, name)
	if !ok || filepath.Dir(full) != dir {
		jsonFailMsg(c, "非法路径")
		return
	}
	rel, err := filepath.Rel(root, full)
	if err != nil {
		jsonFailMsg(c, "非法路径")
		return
	}
	rel = filepath.ToSlash(rel)
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		jsonFailMsg(c, "删除失败")
		return
	}
	core.UnloadScript(rel)
	jsonOK(c, rel)
}
