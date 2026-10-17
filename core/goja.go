package core

import (
	"fmt"
	"jiying/middleware/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/robfig/cron/v3"
)

type Function struct {
	PluginPath    string
	PluginContent string
	PluginLang    string
	Manifest      PluginManifest
	Routers       []string
	Methods       []string
	Events        []string
	DisTool       bool
	Rules         []string
	FindAll       bool
	Admin         bool
	Handle        string
	Cron          string
	CronId        int
	Priority      int
	Disable       bool
	Hash          string
	Bypass        bool
	Origin        bool
	Title         string
	OpenSource    bool
	IsPinned      bool
	Icon          string
	Params        []string
	Class         string
	Author        string
	CreatedAt     time.Time
	Version       string
	Download      string
	Platform      bool
	Public        bool
	Price         int
	Service       string
	Description   string
	Previews      []string
	Rating        int
	ImTypeWhite   []string
	ImTypeBlack   []string
	UserIdWhite   []string
	UserIdBlack   []string
	GroupIdWhite  []string
	GroupIdBlack  []string
	rulesRe       []*regexp.Regexp
}

var (
	Functions    []*Function
	ScriptEngine *goja.Runtime
	ScriptCron   *cron.Cron
	scriptMu     sync.Mutex
)

func InitGoja() {
	GetMyLog().Warn("goja引擎初始化中...")
	DelOriginCommand()
	DelOriginCommand()

	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	runtime := goja.New()
	reg.Enable(runtime)
	console.Enable(runtime)
	ScriptEngine = runtime

	ScriptCron = cron.New()
	ScriptCron.Start()

	registerBasicApis(runtime)
	registerSendApis(runtime, "", "")
	if err := os.MkdirAll(scriptsDir(), 0o750); err != nil {
		GetMyLog().Warn("创建脚本目录失败: %v", err)
	}
	loadScripts()
}

func scriptsDir() string {
	return filepath.Join(utils.GetResHome(), "scripts")
}

func DelOriginCommand() {
	funcLock.Lock()
	defer funcLock.Unlock()
	kept := Functions[:0]
	for _, f := range Functions {
		if f == nil {
			continue
		}
		if f.Origin {
			if f.CronId > 0 && ScriptCron != nil {
				ScriptCron.Remove(cron.EntryID(f.CronId))
			}
			continue
		}
		kept = append(kept, f)
	}
	Functions = kept
}

func registerBasicApis(rt *goja.Runtime) {
	_ = rt.Set("log", func(args ...goja.Value) {
		GetMyLog().Info("%s", formatGojaArgs(args))
	})
}

func formatGojaArgs(args []goja.Value) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if s, ok := a.Export().(string); ok {
			parts = append(parts, s)
			continue
		}
		parts = append(parts, fmt.Sprintf("%v", a.Export()))
	}
	return strings.Join(parts, " ")
}

func DebugScript(code string) (string, error) {
	rt := goja.New()
	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	reg.Enable(rt)
	console.Enable(rt)
	registerBasicApis(rt)
	registerSendApis(rt, "", "")
	var buf strings.Builder
	_ = rt.Set("log", func(args ...goja.Value) {
		buf.WriteString(formatGojaArgs(args) + "\n")
	})
	v, err := runWithTimeout(rt, code, scriptTimeout)
	if err != nil {
		return buf.String(), err
	}
	if v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		buf.WriteString(fmt.Sprintf("%v", v.Export()) + "\n")
	}
	return buf.String(), nil
}

func loadScripts() {
	dir := scriptsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		GetMyLog().Warn("读取脚本目录失败 %s: %v", dir, err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lang := scriptLangOf(name)
		if lang == "" {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			GetMyLog().Warn("读取脚本失败 %s: %v", path, err)
			continue
		}
		fn := &Function{
			PluginPath:    path,
			PluginContent: string(data),
			PluginLang:    lang,
			Title:         scriptTitle(name),
			Origin:        false,
		}
		applyFunctionManifest(fn, string(data))
		appendFunction(fn)
		_ = runScript(fn)
	}
}

func runScript(f *Function) error {
	if ScriptEngine == nil {
		return fmt.Errorf("引擎未初始化")
	}
	if isPythonLang(f.PluginLang) {
		// Python 脚本不在启动时执行，改由 runScriptFileCtx 在子进程中按需运行。
		return nil
	}
	// goja Runtime 不是 goroutine-safe 的，命令并发执行时必须串行进入同一个运行时。
	scriptMu.Lock()
	defer scriptMu.Unlock()
	_ = ScriptEngine.Set("args", "")
	_ = ScriptEngine.Set("result", "")
	cfgObj, _ := pluginRuntimeConfig(filepath.Base(f.PluginPath))
	_ = ScriptEngine.Set("config", cfgObj)
	setupCommonJS(ScriptEngine)
	_, err := runWithTimeout(ScriptEngine, f.PluginContent, scriptTimeout)
	if err != nil {
		GetMyLog().Warn("脚本执行失败 %s: %v", f.PluginPath, err)
		return err
	}
	GetMyLog().Warn("脚本加载成功 %s", f.PluginPath)
	return nil
}

const scriptTimeout = 15 * time.Second

// runWithTimeout 在 goja 运行时上执行脚本，超时后中断执行。
func runWithTimeout(rt *goja.Runtime, code string, timeout time.Duration) (goja.Value, error) {
	if rt == nil {
		return nil, fmt.Errorf("引擎未初始化")
	}
	var fired atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		fired.Store(true)
		rt.Interrupt("脚本执行超时")
	})
	defer func() {
		timer.Stop()
		if fired.Load() {
			// 中断可能落在 RunString 返回之后，清掉挂起标志以免污染下一次执行。
			rt.ClearInterrupt()
		}
	}()
	return rt.RunString(code)
}

func scriptAbsPath(relPath string) string {
	clean := filepath.Clean("/" + relPath)
	full := filepath.Join(scriptsDir(), clean)
	rel, err := filepath.Rel(scriptsDir(), full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return full
}

func ReloadScript(relPath string) error {
	full := scriptAbsPath(relPath)
	if full == "" {
		return fmt.Errorf("非法脚本路径 %s", relPath)
	}
	lang := scriptLangOf(filepath.Base(full))
	if lang == "" {
		return fmt.Errorf("不支持的脚本类型 %s", relPath)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	for _, f := range FunctionsSnapshot() {
		if f != nil && f.PluginPath == full {
			prev, prevLang := f.PluginContent, f.PluginLang
			f.PluginContent = string(data)
			f.PluginLang = lang
			applyFunctionManifest(f, string(data))
			if err := runScript(f); err != nil {
				// 保留上一次可用内容，避免热加载失败后命令表里留着坏脚本。
				f.PluginContent = prev
				f.PluginLang = prevLang
				return err
			}
			return nil
		}
	}
	fn := &Function{
		PluginPath:    full,
		PluginContent: string(data),
		PluginLang:    lang,
		Title:         scriptTitle(full),
		Origin:        false,
	}
	applyFunctionManifest(fn, string(data))
	appendFunction(fn)
	if err := runScript(fn); err != nil {
		removeFunctionByPath(full)
		return err
	}
	return nil
}

// removeFunctionByPath 从命令表中移除指定脚本，用于热加载失败后的回滚。
func removeFunctionByPath(full string) {
	funcLock.Lock()
	defer funcLock.Unlock()
	out := Functions[:0]
	for _, f := range Functions {
		if f != nil && f.PluginPath == full {
			continue
		}
		out = append(out, f)
	}
	Functions = out
}

func UnloadScript(relPath string) {
	full := scriptAbsPath(relPath)
	if full == "" {
		return
	}
	funcLock.Lock()
	defer funcLock.Unlock()
	out := Functions[:0]
	for _, f := range Functions {
		if f != nil && f.PluginPath == full {
			continue
		}
		out = append(out, f)
	}
	Functions = out
}
