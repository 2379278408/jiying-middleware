package gin

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"jiying/middleware/core"
)

func execTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func flattenRecs(list []*Rec) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(list))
	for _, r := range list {
		m := map[string]interface{}{"id": r.ID, "on": r.ON, "position": r.Position}
		for k, v := range r.Fields {
			m[k] = v
		}
		out = append(out, m)
	}
	return out
}

func flattenRec(r *Rec) map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	return flattenRecs([]*Rec{r})[0]
}

func getDependenciesReal(c *gin.Context) {
	deps := subStore("dependencies")
	all := append(flattenRecs(deps.List()), detectEnvDeps()...)
	if sv := strings.TrimSpace(c.Query("searchValue")); sv != "" {
		filtered := make([]map[string]interface{}, 0, len(all))
		for _, row := range all {
			if t, _ := row["type"].(string); t == sv {
				filtered = append(filtered, row)
			}
		}
		all = filtered
	}
	jsonOK(c, all)
}

func depsAddDeps(store *BucketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, _ := c.GetRawData()
		var arrs []map[string]interface{}
		if json.Unmarshal(raw, &arrs) != nil || arrs == nil {
			var one map[string]interface{}
			if json.Unmarshal(raw, &one) != nil {
				jsonFailMsg(c, "参数错误")
				return
			}
			arrs = []map[string]interface{}{one}
		}
		for _, m := range arrs {
			r := recFromMap(m)
			if r.ID > 0 {
				_ = store.Upd(r)
			} else {
				if _, has := r.Fields["status"]; !has {
					r.Fields["status"] = 6
				}
				_ = store.Add(r)
				go installDep(store, r)
			}
		}
		jsonOK(c, flattenRecs(store.List()))
	}
}

func depsUpdateDeps(store *BucketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, _ := c.GetRawData()
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) != nil {
			jsonFailMsg(c, "参数错误")
			return
		}
		r := recFromMap(m)
		if r.ID <= 0 {
			jsonFailMsg(c, "缺少 id")
			return
		}
		if err := store.Upd(r); err != nil {
			jsonFailMsg(c, "未找到")
			return
		}
		jsonOK(c, flattenRecs(store.List()))
	}
}

func recFromMap(m map[string]interface{}) *Rec {
	r := &Rec{Fields: map[string]interface{}{}}
	for k, v := range m {
		if k == "id" {
			if id, ok := v.(float64); ok {
				r.ID = int64(id)
			}
			continue
		}
		if k == "on" {
			if on, ok := v.(bool); ok {
				r.ON = on
			}
			continue
		}
		if k == "position" {
			if p, ok := v.(float64); ok {
				r.Position = int64(p)
			}
			continue
		}
		r.Fields[k] = v
	}
	return r
}

// depNameRe 约束依赖名只能包含包名允许的字符，且不能以 - 开头，
// 避免把请求参数当作命令行选项或注入额外命令。
var depNameRe = regexp.MustCompile(`^[A-Za-z0-9@][A-Za-z0-9._@/+:-]*$`)

func depInstallArgs(typ, name string) []string {
	if !depNameRe.MatchString(name) {
		return nil
	}
	switch typ {
	case "nodejs":
		return []string{"npm", "install", "-g", name}
	case "python":
		return []string{"pip3", "install", name}
	case "golang":
		return []string{"go", "install", name}
	case "linux":
		return []string{"apt-get", "install", "-y", name}
	default:
		return nil
	}
}

func installDep(store *BucketStore, r *Rec) {
	name, _ := r.Fields["name"].(string)
	typ, _ := r.Fields["type"].(string)
	argv := depInstallArgs(typ, name)
	status := 2
	if len(argv) > 0 {
		ctx, cancel := execTimeout(10 * time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		if err := cmd.Run(); err == nil {
			status = 1
		}
	}
	r.Fields["status"] = status
	_ = store.Upd(r)
	core.GetMyLog().Info("依赖安装 %s(%s) 完成，status=%d", name, typ, status)
}

func detectEnvDeps() []map[string]interface{} {
	items := []struct {
		name, typ, cmd string
	}{
		{"node", "nodejs", "node --version"},
		{"npm", "nodejs", "npm --version"},
		{"python3", "python", "python3 --version"},
		{"pip3", "python", "pip3 --version"},
		{"git", "linux", "git --version"},
		{"go", "golang", "go version"},
	}
	id := int64(-1)
	out := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		row := map[string]interface{}{
			"id":        id,
			"name":      it.name,
			"type":      it.typ,
			"version":   "",
			"status":    3,
			"curStatus": "未安装",
			"builtin":   true,
		}
		if v := probeVersion(it.cmd); v != "" {
			row["version"] = v
			row["status"] = 1
			row["curStatus"] = "已安装"
		}
		out = append(out, row)
		id--
	}
	return out
}

func probeVersion(cmdline string) string {
	ctx, cancel := execTimeout(2 * time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmdline).Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSpace(strings.TrimPrefix(v, "git version "))
	v = strings.TrimSpace(strings.TrimPrefix(v, "go version "))
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = v[:i]
	}
	return v
}
