package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/gin-gonic/gin"
)

// skillsDir 与 scripts 同级，用于存放可复用的技能模块。
func skillsDir() string {
	return filepath.Join(filepath.Dir(scriptsDir()), "skills")
}

// scriptFolders 返回 goja require 的全局搜索目录：脚本与技能。
func scriptFolders() []string {
	return []string{scriptsDir(), skillsDir()}
}

// safeSkillPath 校验技能文件名，拒绝路径穿越。技能支持 .js 与 .md。
func safeSkillPath(name string) string {
	base := filepath.Base(name)
	if base != name || strings.Contains(name, "..") || base == "." || base == ".." {
		return ""
	}
	if !strings.HasSuffix(base, ".js") && !strings.HasSuffix(base, ".md") {
		return ""
	}
	return filepath.Join(skillsDir(), base)
}

// SkillItem 是技能列表项。
type SkillItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Kind string `json:"kind"`
}

func ListSkills() []SkillItem {
	entries, err := os.ReadDir(skillsDir())
	if err != nil {
		return nil
	}
	out := make([]SkillItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		kind := "markdown"
		if strings.HasSuffix(e.Name(), ".js") {
			kind = "script"
		} else if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		size := int64(0)
		if fi, err := e.Info(); err == nil {
			size = fi.Size()
		}
		out = append(out, SkillItem{Name: e.Name(), Size: size, Kind: kind})
	}
	return out
}

// InstallSkill 原子写入技能文件；.js 先隔离试跑，失败不改动线上目录。
func InstallSkill(name, content string) error {
	full := safeSkillPath(name)
	if full == "" {
		return fmt.Errorf("只接受 skills 目录下的 .js 或 .md 文件名")
	}
	if strings.HasSuffix(name, ".js") {
		if strings.Contains(content, "..") && strings.Contains(content, "readFile") {
			return fmt.Errorf("技能含禁止路径")
		}
		if _, err := trialRunScript(content, ""); err != nil {
			return fmt.Errorf("试跑失败: %w", err)
		}
	}
	if err := os.MkdirAll(skillsDir(), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(skillsDir(), ".skill-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, full); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

func DeleteSkill(name string) error {
	full := safeSkillPath(name)
	if full == "" {
		return fmt.Errorf("非法技能名")
	}
	return os.Remove(full)
}

// RunSkill 执行 .js 技能并返回结果字符串。
// 技能以 CommonJS 语义加载：若导出的是函数则用参数调用，否则回退到脚本的 result。
func RunSkill(name, args string) (string, error) {
	full := safeSkillPath(name)
	if full == "" {
		return "", fmt.Errorf("非法技能名")
	}
	if !strings.HasSuffix(name, ".js") {
		return "", fmt.Errorf("只有 .js 技能可以执行")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	rt := goja.New()
	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	reg.Enable(rt)
	console.Enable(rt)
	registerBasicApis(rt)
	registerSendApis(rt, "", "")
	module := setupCommonJS(rt)
	_ = rt.Set("args", args)
	_ = rt.Set("result", "")
	if _, err := runWithTimeout(rt, string(data), scriptTimeout); err != nil {
		return "", err
	}
	if fn, ok := goja.AssertFunction(module.Get("exports")); ok {
		v, err := fn(goja.Undefined(), rt.ToValue(args))
		if err != nil {
			return "", err
		}
		return valueString(v), nil
	}
	return valueString(rt.Get("result")), nil
}

// valueString 把 goja 返回值转成字符串，空值统一为 "ok"，与脚本执行保持一致。
func valueString(v goja.Value) string {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "ok"
	}
	return v.String()
}

func registerSkillsAPI() {
	g := Server.Group("/jiying/api", requireLogin)
	g.GET("/skills", func(c *gin.Context) {
		jiyingJSON(c, ListSkills())
	})
	g.POST("/skill/install", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		code := c.PostForm("code")
		if name == "" || code == "" {
			jiyingErr(c, "缺少文件名或内容")
			return
		}
		if err := InstallSkill(name, code); err != nil {
			RecordAudit(consoleUser(c), "skill.install", name, "", "失败: "+err.Error())
			jiyingErr(c, "安装失败: "+err.Error())
			return
		}
		RecordAudit(consoleUser(c), "skill.install", name, fmt.Sprintf("%d 字节", len(code)), "成功")
		jiyingJSON(c, "技能已安装 "+filepath.Base(name))
	})
	g.POST("/skill/delete", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if err := DeleteSkill(name); err != nil {
			RecordAudit(consoleUser(c), "skill.delete", name, "", "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "skill.delete", name, "", "成功")
		jiyingJSON(c, "ok")
	})
	g.POST("/skill/run", requireRole(RoleOps), func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		args := c.PostForm("args")
		out, err := RunSkill(name, args)
		if err != nil {
			RecordAudit(consoleUser(c), "skill.run", name, args, "失败: "+err.Error())
			jiyingErr(c, err.Error())
			return
		}
		RecordAudit(consoleUser(c), "skill.run", name, args, "成功")
		jiyingJSON(c, out)
	})
}
