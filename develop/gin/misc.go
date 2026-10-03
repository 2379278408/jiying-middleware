package gin

import (
	"encoding/json"
	"fmt"
	"io"
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
			!strings.HasSuffix(name, ".bmp") {
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
	dir := filepath.Join(utils.GetResHome(), "images", "gallery")
	_ = os.MkdirAll(dir, 0o755)
	dst := filepath.Join(dir, filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, dst); err != nil {
		jsonFailMsg(c, "保存失败")
		return
	}
	jsonOK(c, "/imgs/static/"+filepath.Base(file.Filename))
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
	path := strings.TrimSpace(c.PostForm("path"))
	dir := filepath.Join(utils.GetResHome(), "scripts")
	if sub == "/skill" {
		dir = filepath.Join(utils.GetResHome(), "skills")
	}
	if path != "" && path != "/" {
		dir = filepath.Join(dir, path)
	}
	_ = os.MkdirAll(dir, 0o755)
	dst := filepath.Join(dir, filepath.Base(file.Filename))
	if err := c.SaveUploadedFile(file, dst); err != nil {
		jsonFailMsg(c, "保存失败")
		return
	}
	jsonOK(c, file.Filename)
}

func getCronCmds(c *gin.Context) {
	jsonOK(c, core.CronCmdsSnapshot())
}

func postCronCmds(c *gin.Context) {
	fields := bodyMap(c)
	cmd := core.AddCronCmdFrom(fields)
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
	cmd := core.UpdateCronCmdFrom(id, fields)
	jsonOK(c, cmd)
}

func deleteCronCmds(c *gin.Context) {
	raw, _ := c.GetRawData()
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil || len(ids) == 0 {
		ids = splitStr(c.PostForm("ids"))
	}
	n := core.DeleteCronCmdsByIds(ids)
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
