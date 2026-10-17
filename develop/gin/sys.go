package gin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/develop/users"
	"jiying/middleware/utils"
)

func initSystem() {
	g := core.Server.Group("/")
	g.Use(AuthRequired())
	g.GET("/system/params", getSystemParams)
	g.POST("/system/params", requireConsoleMaster(), saveSystemParams)
	g.GET("/system/:key", requireConsoleMaster(), getSystemKey)
	g.GET("/logs/menu", requireConsoleMaster(), getLogsMenu)
	g.GET("/logs/:name", requireConsoleMaster(), getLogFile)
	g.GET("/logs", requireConsoleMaster(), getLogs)
	g.DELETE("/logs", requireConsoleMaster(), deleteLogFile)
	g.GET("/realtime", getRealtimeReal)
	g.GET("/statistics", getStatisticsReal)
	g.GET("/theme", getTheme)
	g.POST("/restart", requireConsoleMaster(), doRestart)
	g.GET("/buckets", requireConsoleMaster(), getBuckets)
	g.GET("/notice/config", getNoticeConfig)
	g.POST("/notice/read-all", readAllNotices)
	g.POST("/notice/delete-all", deleteAllNotices)
	g.GET("/notice", getNotices)
}

func getSystemParams(c *gin.Context) {
	params := map[string]interface{}{
		"bucket": "middleware",
		"key":    "config",
	}
	for _, k := range []string{
		"appName", "frontendWeb", "webSite", "backPage", "drivingSchool",
		"register_open", "icp", "login_logo", "header_bg", "default_avatar",
		"grant_plugin_default", "stream_broadcast", "sf", "make", "data_version",
		"cache_clear", "croncmds_autoremove_days", "varlink_regs",
	} {
		if v := core.BucketApp.GetString(k); v != "" {
			params[k] = v
		}
	}
	jsonOK(c, params)
}

func getSystemKey(c *gin.Context) {
	bucket := c.Param("key")
	if sensitiveBucketKey(bucket) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "该配置项不可读取"})
		return
	}
	key := c.Query("key")
	body := core.BucketApp.GetString(bucket)
	if key != "" {
		_ = bucket
		if sensitiveBucketKey(key) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "该配置项不可读取"})
			return
		}
		bk, err := core.MakeBucket(bucket, 0)
		if err == nil && bk != nil {
			body = bk.GetString(key)
		}
	}
	jsonOK(c, body)
}

func saveSystemParams(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		jsonFailMsg(c, "参数错误")
		return
	}
	saved := 0
	for k, vs := range c.Request.PostForm {
		if len(vs) == 0 || k == "" {
			continue
		}
		if protectedSystemKey(k) {
			continue
		}
		core.BucketApp.Set(k, vs[0])
		saved++
	}
	if saved == 0 {
		jsonFailMsg(c, "未收到参数")
		return
	}
	core.GetMyLog().Info("系统参数保存 %d 项", saved)
	jsonOK(c, gin.H{"saved": saved})
}

func getBuckets(c *gin.Context) {
	bucket := c.Query("bucket")
	key := c.Query("key")
	if bucket == "" {
		bucket = "middleware"
	}
	var body string
	if key == "" {
		bk, err := core.MakeBucket(bucket, 0)
		if err != nil || bk == nil {
			jsonOK(c, "")
			return
		}
		body = bucketSummary(bk)
	} else {
		if sensitiveBucketKey(key) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "该配置项不可读取"})
			return
		}
		bk, err := core.MakeBucket(bucket, 0)
		if err != nil || bk == nil {
			jsonOK(c, "")
			return
		}
		body = bk.GetString(key)
	}
	jsonOK(c, body)
}

func bucketSummary(bk core.Bucket) string {
	ks, ok := bk.(interface{ Keys() []string })
	if !ok {
		return fmt.Sprintf("%s", bk)
	}
	var b strings.Builder
	for _, k := range ks.Keys() {
		if sensitiveBucketKey(k) {
			continue
		}
		v := bk.GetString(k)
		if len(v) > 120 {
			v = v[:120] + "..."
		}
		b.WriteString(fmt.Sprintf("%s = %s\n", k, v))
	}
	return b.String()
}

// sensitiveBucketKey 报告桶键是否承载凭据类数据：这类值不通过通用配置/桶读取接口回传，
// 避免会话签名密钥、插件密钥等被任意账号读出。
func sensitiveBucketKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	switch k {
	case "session_key", "secret_key", "apikey", "api_key", "access_token", "refresh_token":
		return true
	}
	return strings.HasPrefix(k, "plugincfg:")
}

// protectedSystemKey 报告键是否只能由专用接口写入：通用系统参数保存必须跳过
// 会话密钥、插件密钥以及主控/角色表，避免越权改写权限与凭据。
func protectedSystemKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	return sensitiveBucketKey(k) || k == "masters" || k == "jiying_roles"
}

func logHomeDir() string {
	return filepath.Join(utils.GetResHome(), "logs")
}

func getLogsMenu(c *gin.Context) {
	dir := logHomeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonOK(c, []interface{}{})
		return
	}
	var logs []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		logs = append(logs, map[string]interface{}{"name": e.Name(), "is_directory": false})
	}
	sort.Sort(sort.Reverse(sort.StringSlice(func() []string {
		names := make([]string, 0, len(logs))
		for _, m := range logs {
			names = append(names, m["name"].(string))
		}
		return names
	}())))
	if len(logs) > 30 {
		logs = logs[:30]
	}
	jsonOK(c, logs)
}

func getLogFile(c *gin.Context) {
	name := filepath.Base(strings.TrimSpace(c.Param("name")))
	if !strings.HasSuffix(name, ".log") {
		jsonFailMsg(c, "仅支持 .log 文件")
		return
	}
	full := filepath.Join(logHomeDir(), name)
	data, err := readTail(full, 256*1024)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": string(data)})
}

func deleteLogFile(c *gin.Context) {
	var body struct {
		Filename string `json:"filename"`
		Path     string `json:"path"`
	}
	_ = c.ShouldBindJSON(&body)
	name := strings.TrimSpace(body.Filename)
	if name == "" {
		jsonFailMsg(c, "参数错误")
		return
	}
	name = filepath.Base(name)
	if !strings.HasSuffix(name, ".log") {
		jsonFailMsg(c, "仅支持 .log 文件")
		return
	}
	if err := os.Remove(filepath.Join(logHomeDir(), name)); err != nil {
		jsonFailMsg(c, "删除失败")
		return
	}
	jsonOK(c, name)
}

func readTail(full string, max int64) ([]byte, error) {
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > max {
		if _, err := f.Seek(-max, io.SeekEnd); err != nil {
			return nil, err
		}
	}
	return io.ReadAll(f)
}

func getLogs(c *gin.Context) {
	dir := logHomeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": ""})
		return
	}
	var target string
	var newest int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		if st, err := e.Info(); err == nil && st.ModTime().Unix() >= newest {
			newest = st.ModTime().Unix()
			target = e.Name()
		}
	}
	if target == "" {
		c.JSON(http.StatusOK, gin.H{"code": 200, "data": ""})
		return
	}
	data, err := readTail(filepath.Join(dir, target), 256*1024)
	if err != nil {
		data = nil
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": string(data)})
}

func joinPath(dir, name string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}

func usersCount() int {
	return len(users.Users)
}

func getTheme(c *gin.Context) {
	theme := core.BucketApp.GetString("theme", "web")
	if theme == "" {
		theme = "default"
	}
	jsonOK(c, gin.H{
		"site_theme": theme,
		"theme":      theme,
		"logo_bg":    "default",
		"header_bg":  "default",
		"sidebar_bg": "default",
	})
}

func doRestart(c *gin.Context) {
	core.GetMyLog().Info("收到重启指令")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "重启中..."})
	time.AfterFunc(200*time.Millisecond, func() {
		os.Exit(0)
	})
}

func getNoticeConfig(c *gin.Context) {
	jsonOK(c, gin.H{"switch": true})
}

func getNotices(c *gin.Context) {
	jsonOK(c, []interface{}{})
}

func readAllNotices(c *gin.Context)   { jsonOK(c, nil) }
func deleteAllNotices(c *gin.Context) { jsonOK(c, nil) }
