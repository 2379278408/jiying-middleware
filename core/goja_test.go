package core

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

func TestGojaEval(t *testing.T) {
	if ScriptEngine == nil {
		ScriptEngine = newTestEngine(t)
	}
	v, err := ScriptEngine.RunString(`(function(){ return 1 + 2 * 3 })()`)
	if err != nil {
		t.Fatalf("执行脚本失败: %v", err)
	}
	if got := v.ToInteger(); got != 7 {
		t.Fatalf("脚本结果应为 7，实际 %d", got)
	}
}

func TestDelOriginCommand(t *testing.T) {
	backup := Functions
	Functions = []*Function{
		{Title: "origin-cmd", Origin: true},
		{Title: "user-script"},
		{Title: "plugin", Origin: false},
	}
	DelOriginCommand()
	if len(Functions) != 2 {
		t.Fatalf("期望保留 2 个命令，实际 %d", len(Functions))
	}
	for _, f := range Functions {
		if f.Origin {
			t.Fatalf("Origin 命令应被移除: %s", f.Title)
		}
	}
	Functions = backup
}

func TestAddOriginCommand(t *testing.T) {
	backup := Functions
	Functions = nil
	t.Cleanup(func() { Functions = backup })
	AddOriginCommand(&Function{Title: "t:command", Origin: true})
	if len(Functions) != 1 {
		t.Fatalf("注册后函数表应有 1 个命令，实际 %d", len(Functions))
	}
	DelOriginCommand()
	if len(Functions) != 0 {
		t.Fatalf("清除 Origin 后应为空，实际 %d", len(Functions))
	}
}

func newTestEngine(t *testing.T) *goja.Runtime {
	t.Helper()
	rt := goja.New()
	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	reg.Enable(rt)
	console.Enable(rt)
	registerBasicApis(rt)
	registerSendApis(rt, "", "")
	setupCommonJS(rt)
	return rt
}
