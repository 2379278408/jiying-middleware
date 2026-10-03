package core

import (
	"os"
	"path/filepath"
	"testing"
)

func setupSkills(t *testing.T) {
	t.Helper()
	t.Setenv("JIYING_RES", t.TempDir())
	_ = os.MkdirAll(skillsDir(), 0o755)
}

func TestInstallSkillRejectsBadNames(t *testing.T) {
	setupSkills(t)
	for _, name := range []string{"../evil.js", "sub/dir.js", "note.txt", ""} {
		if err := InstallSkill(name, "// x"); err == nil {
			t.Fatalf("name %q should be rejected", name)
		}
	}
}

func TestInstallAndListSkills(t *testing.T) {
	setupSkills(t)
	if err := InstallSkill("guide.md", "# 指南"); err != nil {
		t.Fatal(err)
	}
	if err := InstallSkill("helper.js", "result = 'ok';"); err != nil {
		t.Fatal(err)
	}
	list := ListSkills()
	if len(list) != 2 {
		t.Fatalf("want 2 skills, got %d: %+v", len(list), list)
	}
	kinds := map[string]string{}
	for _, s := range list {
		kinds[s.Name] = s.Kind
	}
	if kinds["guide.md"] != "markdown" || kinds["helper.js"] != "script" {
		t.Fatalf("unexpected kinds: %+v", kinds)
	}
}

func TestRunSkillScript(t *testing.T) {
	setupSkills(t)
	if err := InstallSkill("calc.js", "result = 'x' + args;"); err != nil {
		t.Fatal(err)
	}
	out, err := RunSkill("calc.js", "1")
	if err != nil {
		t.Fatal(err)
	}
	if out != "x1" {
		t.Fatalf("run output mismatch: %q", out)
	}
	if _, err := RunSkill("guide.md", ""); err == nil {
		t.Fatal("markdown skill must not be runnable")
	}
}

// 技能的自然写法是 module.exports；安装试跑必须提供 CommonJS 全局，否则报 module is not defined。
func TestInstallSkillAllowsModuleExports(t *testing.T) {
	setupSkills(t)
	if err := InstallSkill("lib.js", "module.exports = function (a) { return 'hi:' + a; };"); err != nil {
		t.Fatalf("module.exports 技能应可安装: %v", err)
	}
}

func TestRunSkillCallsExportedFunction(t *testing.T) {
	setupSkills(t)
	if err := InstallSkill("lib.js", "module.exports = function (a) { return 'hi:' + a; };"); err != nil {
		t.Fatal(err)
	}
	out, err := RunSkill("lib.js", "7")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi:7" {
		t.Fatalf("导出函数应被调用并返回结果, got %q", out)
	}
}

// 插件应能 require 复用技能目录下的模块。
func TestScriptRequiresSkillModule(t *testing.T) {
	setupSkills(t)
	// 热加载走全局 ScriptEngine，其 require 注册表绑定创建时的脚本目录；
	// 本测试指向新的临时目录，需重建引擎避免复用其他用例的旧路径。
	prev := ScriptEngine
	ScriptEngine = newTestEngine(t)
	defer func() { ScriptEngine = prev }()
	if err := InstallSkill("lib.js", "module.exports = function (a) { return 'hi:' + a; };"); err != nil {
		t.Fatal(err)
	}
	if err := InstallScript("main2.js", "var lib = require('lib.js'); result = lib('9');"); err != nil {
		t.Fatal(err)
	}
	out, err := evalScriptCtx("var lib = require('lib.js'); result = lib('9');", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi:9" {
		t.Fatalf("require 技能模块失败, got %q", out)
	}
}

func TestDeleteSkill(t *testing.T) {
	setupSkills(t)
	if err := InstallSkill("gone.md", "x"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSkill("gone.md"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSkill("gone.md"); err == nil {
		t.Fatal("deleting a missing skill should error")
	}
	if err := DeleteSkill("../x.js"); err == nil {
		t.Fatal("bad name must be rejected")
	}
	if _, err := os.Stat(filepath.Join(skillsDir(), "gone.md")); !os.IsNotExist(err) {
		t.Fatal("skill file should be gone")
	}
}
