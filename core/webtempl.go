package core

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/utils"
)

var (
	webTemplSources = make(map[int64]string)
	webTemplDatas   = make(map[int64]interface{})
)

func initGlob() {
	loadPageTemplates()
	_dataTpl := DataHome
	_ = _dataTpl
}

func loadPageTemplates() {
	dirs := []string{
		filepath.Join(utils.GetResHome(), "views2", "templates"),
		filepath.Join(utils.GetDataHome(), "templates"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		idx := int64(1)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			webTemplSources[idx] = string(data)
			webTemplDatas[idx] = gin.H{}
			idx++
		}
	}
}

func pageIndex(c *gin.Context) {
	if id := utils.Int64(c.Query("t")); id >= 1 {
		text, ok := webTemplSources[id]
		if !ok {
			GetMyLog().Errorf("页面不存在 t=%d", id)
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "页面不存在"})
			return
		}
		tmpl, err := template.New(fmt.Sprintf("page%d", id)).Parse(text)
		if err != nil {
			GetMyLog().Errorf("模板解析失败 t=%d: %v", id, err)
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板错误"})
			return
		}
		if err := tmpl.Execute(c.Writer, webTemplDatas[id]); err != nil {
			GetMyLog().Errorf("模板渲染失败 t=%d: %v", id, err)
			return
		}
		return
	}
	serveIndex(c)
}

func serveIndex(c *gin.Context) {
	assetRoot := filepath.Join(utils.GetResHome(), "views2", "assets")
	if st, err := os.Stat(filepath.Join(assetRoot, "index.html")); err != nil || st.IsDir() {
		assetRoot = filepath.Join(utils.GetExecPathDir(), "views2", "assets")
		if _, err := os.Stat(filepath.Join(assetRoot, "index.html")); err != nil {
			c.String(http.StatusNotFound, "index.html 不存在")
			return
		}
	}
	data, err := os.ReadFile(filepath.Join(assetRoot, "index.html"))
	if err != nil {
		c.String(http.StatusNotFound, "index.html 读取失败")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
