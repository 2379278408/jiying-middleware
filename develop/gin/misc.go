package gin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/utils"
)

func initMisc() {
	g := core.Server.Group("/")
	g.Use(AuthRequired())

	g.GET("/instructions", getInstructionsReal)

	g.GET("/gallery", getGallery)
	g.POST("/gallery", requireConsoleRole(core.RoleOps), uploadGallery)
	g.DELETE("/gallery", requireConsoleRole(core.RoleOps), deleteGallery)
	core.Server.Static("/imgs/static", filepath.Join(utils.GetResHome(), "images", "gallery"))

	g.GET("/nameids", getNameIds)
	g.POST("/bindingEmail", bindingEmail)

	g.POST("/scripts/upload", requireConsoleRole(core.RoleOps), uploadScriptFile)
	g.POST("/skills/upload", requireConsoleRole(core.RoleOps), uploadSkillFile)

	g.GET("/croncmds", getCronCmds)
	g.POST("/croncmds", requireConsoleRole(core.RoleOps), postCronCmds)
	g.PUT("/croncmds", requireConsoleRole(core.RoleOps), putCronCmds)
	g.DELETE("/croncmds", requireConsoleRole(core.RoleOps), deleteCronCmds)

	deps := subStore("dependencies")
	g.GET("/dependencies", getDependenciesReal)
	g.POST("/dependencies", requireConsoleRole(core.RoleOps), depsAddDeps(deps))
	g.PUT("/dependencies", requireConsoleRole(core.RoleOps), depsUpdateDeps(deps))
	g.DELETE("/dependencies", requireConsoleRole(core.RoleOps), depsDelete(deps))

	subs := subStore("subscriptions")
	g.GET("/subscriptions", func(c *gin.Context) { jsonOK(c, flattenRecs(subs.List())) })
	g.GET("/subscriptions/:id", func(c *gin.Context) {
		ids := parseIDs(c)
		if len(ids) == 1 {
			if r := subs.Get(ids[0]); r != nil {
				jsonOK(c, flattenRec(r))
				return
			}
		}
		jsonFailMsg(c, "未找到")
	})
	g.POST("/subscriptions", requireConsoleRole(core.RoleOps), depsAddDeps(subs))
	g.PUT("/subscriptions", requireConsoleRole(core.RoleOps), depsUpdateDeps(subs))
	g.DELETE("/subscriptions", requireConsoleRole(core.RoleOps), depsDelete(subs))
	g.GET("/market/subs", func(c *gin.Context) { jsonOK(c, flattenRecs(subs.List())) })
}

func getGallery(c *gin.Context) {
	dir := filepath.Join(utils.GetResHome(), "images", "gallery")
	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonOK(c, []interface{}{})
		return
	}
	var imgs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".png") &&
			!strings.HasSuffix(name, ".jpeg") && !strings.HasSuffix(name, ".gif") &&
			!strings.HasSuffix(name, ".bmp") && !strings.HasSuffix(name, ".webp") {
			continue
		}
		imgs = append(imgs, "/imgs/static/"+name)
	}
	if imgs == nil {
		imgs = []string{}
	}
	jsonOK(c, imgs)
}

func uploadGallery(c *gin.Context) {
	file, err := c.FormFile("imgfile")
	if err != nil {
		jsonFailMsg(c, "未收到文件")
		return
	}
	if file.Size <= 0 || file.Size > 10<<20 {
		jsonFailMsg(c, "图片大小需在 10MB 以内")
		return
	}
	base := filepath.Base(file.Filename)
	stem, ext := strings.TrimSuffix(base, filepath.Ext(base)), strings.ToLower(filepath.Ext(base))
	if stem == "" || stem == "." || !galleryExtAllowed(ext) {
		jsonFailMsg(c, "只支持 jpg、png、gif、bmp、webp 图片")
		return
	}
	opened, err := file.Open()
	if err != nil {
		jsonFailMsg(c, "读取文件失败")
		return
	}
	head := make([]byte, 512)
	n, _ := opened.Read(head)
	_ = opened.Close()
	contentType := http.DetectContentType(head[:n])
	if !galleryContentMatches(ext, contentType) {
		jsonFailMsg(c, "文件内容不是有效图片")
		return
	}
	dir := filepath.Join(utils.GetResHome(), "images", "gallery")
	_ = os.MkdirAll(dir, 0o755)
	name := stem + ext
	dst := filepath.Join(dir, name)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		jsonFailMsg(c, "保存失败")
		return
	}
	jsonOK(c, "/imgs/static/"+name)
}

func galleryExtAllowed(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

func galleryContentMatches(ext, contentType string) bool {
	switch ext {
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg"
	case ".png":
		return contentType == "image/png"
	case ".gif":
		return contentType == "image/gif"
	case ".bmp":
		return contentType == "image/bmp"
	case ".webp":
		return contentType == "image/webp"
	}
	return false
}

func deleteGallery(c *gin.Context) {
	raw, _ := c.GetRawData()
	var names []string
	if json.Unmarshal(raw, &names) != nil || len(names) == 0 {
		jsonFailMsg(c, "参数错误")
		return
	}
	dir := filepath.Join(utils.GetResHome(), "images", "gallery")
	for _, n := range names {
		name := filepath.Base(n)
		if name == "" || name == "." || name == "/" {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
	jsonOK(c, names)
}

func getNameIds(c *gin.Context) {
	jsonOK(c, gin.H{
		"total": 0,
		"rows":  []interface{}{},
	})
}

func bindingEmail(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	if email == "" {
		jsonFailMsg(c, "邮箱不能为空")
		return
	}
	u := CurrentUser(c)
	if u == nil {
		jsonFailMsg(c, "请先登录")
		return
	}
	u.Email = email
	if err := u.Save(); err != nil {
		jsonFailMsg(c, "保存失败")
		return
	}
	jsonOK(c, nil)
}

func uploadScriptFile(c *gin.Context) {
	uploadTextFile(c, "/varscript")
}

func uploadSkillFile(c *gin.Context) {
	uploadTextFile(c, "/skill")
}

func uploadTextFile(c *gin.Context, sub string) {
	file, err := c.FormFile("file")
	if err != nil {
		jsonFailMsg(c, "未收到文件")
		return
	}
	if file.Size <= 0 || file.Size > maxUploadBytes {
		jsonFailMsg(c, "文件大小超出限制")
		return
	}
	root := filepath.Join(utils.GetResHome(), "scripts")
	if sub == "/skill" {
		root = filepath.Join(utils.GetResHome(), "skills")
	}
	dir, ok := safeJoinUnder(root, c.PostForm("path"))
	if !ok {
		jsonFailMsg(c, "非法路径")
		return
	}
	name, ok := safeUploadName(file.Filename, ".js", ".md", ".json", ".txt", ".yaml", ".yml")
	if !ok {
		jsonFailMsg(c, "不支持的文件类型")
		return
	}
	dst, ok := safeJoinUnder(dir, name)
	if !ok || filepath.Dir(dst) != dir {
		jsonFailMsg(c, "非法文件名")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		jsonFailMsg(c, "目录创建失败")
		return
	}
	if err := c.SaveUploadedFile(file, dst); err != nil {
		jsonFailMsg(c, "保存失败")
		return
	}
	jsonOK(c, name)
}

func getCronCmds(c *gin.Context) {
	actor, role := core.ConsoleActor(c)
	jsonOK(c, core.CronCmdsSnapshotFor(actor, role))
}

func postCronCmds(c *gin.Context) {
	fields := bodyMap(c)
	actor, role := core.ConsoleActor(c)
	cmd := core.AddCronCmdFrom(fields, actor, role)
	jsonOK(c, cmd)
}

func putCronCmds(c *gin.Context) {
	fields := bodyMap(c)
	id := ""
	if v, ok := fields["id"].(string); ok {
		id = v
	} else if v, ok := fields["id"].(float64); ok {
		id = fmt.Sprintf("%.0f", v)
	}
	actor, role := core.ConsoleActor(c)
	cmd := core.UpdateCronCmdFrom(id, fields, actor, role)
	jsonOK(c, cmd)
}

func deleteCronCmds(c *gin.Context) {
	raw, _ := c.GetRawData()
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil || len(ids) == 0 {
		ids = splitStr(c.PostForm("ids"))
	}
	actor, role := core.ConsoleActor(c)
	n := core.DeleteCronCmdsByIds(ids, actor, role)
	jsonOK(c, gin.H{"deleted": n})
}

func bodyMap(c *gin.Context) map[string]interface{} {
	raw, _ := c.GetRawData()
	m := map[string]interface{}{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func splitStr(s string) []string {
	if s == "" {
		return nil
	}
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n'
	})
}

var subStores = map[string]*BucketStore{}

func subStore(name string) *BucketStore {
	if s, ok := subStores[name]; ok {
		return s
	}
	s := NewBucketStore(name, "list")
	subStores[name] = s
	return s
}

func depsDelete(store *BucketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, _ := c.GetRawData()
		var ids []int64
		if json.Unmarshal(raw, &ids) != nil || len(ids) == 0 {
			jsonFailMsg(c, "参数错误")
			return
		}
		for _, id := range ids {
			_ = store.Del(id)
		}
		jsonOK(c, store.List())
	}
}

func rawBody(c *gin.Context) ([]byte, error) {
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	_ = c.Request.Body.Close()
	return b, nil
}

var _ = fmt.Sprintf
