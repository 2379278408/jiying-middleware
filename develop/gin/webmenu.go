package gin

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
	"jiying/middleware/utils"
)

type MenuItem struct {
	Name     string      `json:"name"`
	Mdi      string      `json:"mdi"`
	Link     string      `json:"link,omitempty"`
	Subitems []*MenuItem `json:"subitems,omitempty"`
}

func initWebMenu() {
	core.Server.GET("/webmenu", AuthRequired(), getWebMenu)
}

func getWebMenu(c *gin.Context) {
	jsonOK(c, buildMenu())
}

var menuCats = []struct {
	title, mdi string
	prefix     []string
}{
	{"即应", "mdi mdi-message-text", []string{"aut_jiying_"}},
	{"系统", "mdi mdi-tune", []string{"aut_ui_"}},
	{"插件", "mdi mdi-puzzle", []string{"aut_plugins"}},
	{"用户", "mdi mdi-account-group", []string{"aut_users_"}},
}

func buildMenu() []*MenuItem {
	byCat := map[string]*MenuItem{}
	for _, cat := range menuCats {
		byCat[cat.title] = &MenuItem{Name: cat.title, Mdi: cat.mdi, Subitems: []*MenuItem{}}
	}
	var sys *MenuItem
	var roots []*MenuItem
	for _, cat := range menuCats {
		mi := byCat[cat.title]
		if cat.title == "系统" {
			sys = mi
		}
		roots = append(roots, mi)
	}

	assets := []string{
		filepath.Join(utils.GetResHome(), "views2", "assets"),
		filepath.Join(utils.GetExecPathDir(), "views2", "assets"),
	}
	seen := map[string]bool{}
	for _, dir := range assets {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), "aut_") || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			if strings.HasPrefix(e.Name(), "aut_pages_") || e.Name() == "aut_error.html" {
				continue
			}
			base := strings.TrimSuffix(e.Name(), ".html")
			if seen[base] {
				continue
			}
			seen[base] = true
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, n := range names {
			item := pageItem(n)
			cat := resolveCat(n, byCat, sys)
			cat.Subitems = append(cat.Subitems, item)
		}
	}
	out := roots[:0]
	for _, r := range roots {
		if len(r.Subitems) > 0 {
			out = append(out, r)
		}
	}
	return out
}

func resolveCat(name string, byCat map[string]*MenuItem, sys *MenuItem) *MenuItem {
	for _, cat := range menuCats {
		for _, p := range cat.prefix {
			if strings.HasPrefix(name, p) {
				return byCat[cat.title]
			}
		}
	}
	return sys
}

func pageItem(name string) *MenuItem {
	base := strings.TrimSuffix(strings.TrimPrefix(name, "aut_"), ".html")
	return &MenuItem{
		Name: prettyName(base),
		Mdi:  "mdi mdi-file-document-outline",
		Link: "/assets/" + name,
	}
}

var nameDict = map[string]string{
	"ui_auth":          "插件鉴权",
	"ui_database":      "数据库",
	"ui_instructions":  "说明",
	"ui_logs":          "运行日志",
	"ui_nameids":       "名称映射",
	"users_basic":      "用户管理",
	"users_details":    "用户详情",
	"jiying_dashboard": "即应工作台",
	"jiying_chat":      "消息收发",
	"jiying_scripts":   "插件",
	"jiying_cron":      "定时任务",
	"jiying_tasks":     "任务队列",
	"jiying_logs":      "即应日志",
	"plugins":          "插件",
}

func prettyName(base string) string {
	if n, ok := nameDict[base]; ok {
		return n
	}
	return base
}
