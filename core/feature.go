package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"jiying/middleware/utils"
)

type Feature struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Scope    string `json:"scope"`
	Confirm  bool   `json:"confirm"`
	Identity string `json:"identity"`
	Enabled  bool   `json:"enabled"`
}

type featureTable struct {
	mu    sync.RWMutex
	items map[string]*Feature
}

var features = &featureTable{items: map[string]*Feature{}}

func initFeatures() {
	features.load()
	features.ensure("状态", "builtin", "status", "public", false, "app")
	features.ensure("日志", "builtin", "logs", "master", false, "app")
	features.ensure("帮助", "builtin", "help", "public", false, "app")
	features.ensure("插件列表", "builtin", "plugins", "master", false, "app")
	features.ensure("定时列表", "builtin", "cron", "master", false, "app")
	features.ensure("任务列表", "builtin", "tasks", "master", false, "app")
	features.scanScripts()
}

func (t *featureTable) scanScripts() {
	entries, err := os.ReadDir(scriptsDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || scriptLangOf(e.Name()) == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(scriptsDir(), e.Name()))
		if err != nil {
			continue
		}
		name, scope := scriptFeatureNameScope(e.Name(), string(data))
		if name == "" {
			continue
		}
		t.ensure(name, "script", e.Name(), scope, false, "app")
	}
}

func (t *featureTable) ensure(name, kind, target, scope string, confirm bool, identity string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.items[name]; ok {
		return
	}
	t.items[name] = &Feature{Name: name, Kind: kind, Target: target, Scope: scope, Confirm: confirm, Identity: identity, Enabled: true}
	t.saveLocked()
}

func (t *featureTable) load() {
	if BucketApp == nil {
		return
	}
	raw := BucketApp.GetString("jiying_features")
	if raw == "" {
		return
	}
	var list []*Feature
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, f := range list {
		if f != nil && f.Name != "" {
			t.items[f.Name] = f
		}
	}
}

func (t *featureTable) saveLocked() {
	if BucketApp == nil {
		return
	}
	list := make([]*Feature, 0, len(t.items))
	for _, f := range t.items {
		list = append(list, f)
	}
	raw, _ := json.Marshal(list)
	BucketApp.Set("jiying_features", string(raw))
}

func UpsertFeature(f Feature) error {
	if f.Name == "" || f.Kind == "" {
		return fmt.Errorf("功能名和类型不能为空")
	}
	if f.Kind != "builtin" && f.Kind != "http" && f.Kind != "script" {
		return fmt.Errorf("未知功能类型")
	}
	if f.Kind == "http" && !strings.HasPrefix(f.Target, "http://") && !strings.HasPrefix(f.Target, "https://") {
		return fmt.Errorf("HTTP 功能需要 http(s) 地址")
	}
	if f.Scope == "" {
		f.Scope = "master"
	}
	if f.Scope != "public" && f.Scope != "master" {
		return fmt.Errorf("范围只能是 public 或 master")
	}
	if f.Identity == "" {
		f.Identity = "app"
	}
	if f.Identity != "app" && f.Identity != "user" {
		return fmt.Errorf("身份只能是 app 或 user")
	}
	features.mu.Lock()
	features.items[f.Name] = &f
	features.saveLocked()
	features.mu.Unlock()
	return nil
}

func SetFeatureEnabled(name string, enabled bool) error {
	features.mu.Lock()
	defer features.mu.Unlock()
	f, ok := features.items[name]
	if !ok {
		return fmt.Errorf("没有功能 %s", name)
	}
	f.Enabled = enabled
	features.saveLocked()
	return nil
}

func GetFeature(name string) (*Feature, bool) {
	features.mu.RLock()
	defer features.mu.RUnlock()
	if f, ok := features.items[name]; ok && f != nil && f.Enabled {
		cp := *f
		return &cp, true
	}
	// 允许用内部 target 作为别名调用（例如 status 指向「状态」）。
	// 仅当该 target 唯一时才接受，避免歧义。
	var hit *Feature
	for _, f := range features.items {
		if f == nil || !f.Enabled || f.Target != name {
			continue
		}
		if hit != nil {
			return nil, false
		}
		hit = f
	}
	if hit == nil {
		return nil, false
	}
	cp := *hit
	return &cp, true
}

// getFeatureRaw 返回功能记录，不区分是否启用；用于删除等需要完整信息的场景。
func getFeatureRaw(name string) (*Feature, bool) {
	features.mu.RLock()
	defer features.mu.RUnlock()
	f, ok := features.items[name]
	if !ok || f == nil {
		return nil, false
	}
	cp := *f
	return &cp, true
}

// featureTargetInUse reports whether any remaining script feature still points at target.
func featureTargetInUse(target string) bool {
	features.mu.RLock()
	defer features.mu.RUnlock()
	for _, f := range features.items {
		if f != nil && f.Kind == "script" && f.Target == target {
			return true
		}
	}
	return false
}

func ListFeatureItems() []*Feature {
	features.mu.RLock()
	defer features.mu.RUnlock()
	out := make([]*Feature, 0, len(features.items))
	for _, f := range features.items {
		if f != nil {
			cp := *f
			out = append(out, &cp)
		}
	}
	return out
}

func DeleteFeature(name string) error {
	features.mu.Lock()
	defer features.mu.Unlock()
	if _, ok := features.items[name]; !ok {
		return fmt.Errorf("没有功能 %s", name)
	}
	delete(features.items, name)
	features.saveLocked()
	return nil
}

func ListFeatures() string {
	features.mu.RLock()
	defer features.mu.RUnlock()
	if len(features.items) == 0 {
		return "没有功能"
	}
	var b strings.Builder
	for _, f := range features.items {
		if !f.Enabled {
			continue
		}
		fmt.Fprintf(&b, "%s [%s/%s]\n", f.Name, f.Kind, f.Scope)
	}
	if b.Len() == 0 {
		return "没有功能"
	}
	return strings.TrimSpace(b.String())
}

// featureAllowedFor 判断用户是否有权执行功能：主控专属与清单 Admin 标记都要求主控或控制台管理员身份。
// 执行前校验与 Agent 工具裁剪共用同一套规则，避免模型调用后才被拒绝。
func featureAllowedFor(userID string, f *Feature) bool {
	if f == nil {
		return false
	}
	if privilegedUser(userID) {
		return true
	}
	if f.Scope == "master" {
		return false
	}
	if fn := functionByTitle(f.Name); fn != nil && fn.Manifest.Admin != nil && *fn.Manifest.Admin {
		return false
	}
	return true
}

func executeFeature(ctx context.Context, name, args, userID, chatID string) (string, error) {
	if strings.HasPrefix(name, "cron:") {
		label := strings.TrimPrefix(name, "cron:")
		if args != "" && chatID != "" {
			dispatchChatCommand(chatID, args)
		}
		return "定时已触发 " + label, nil
	}
	f, ok := GetFeature(name)
	if !ok {
		return "", fmt.Errorf("功能不可用: %s", name)
	}
	if !featureAllowedFor(userID, f) {
		return "", fmt.Errorf("无权限")
	}
	switch f.Kind {
	case "builtin":
		return runBuiltin(f.Target, args), nil
	case "http":
		traceID := newTraceID()
		body := map[string]any{"feature": name, "args": args, "user_id": userID, "trace_id": traceID}
		if f.Identity == "user" {
			st := UserSessionStatus()
			if !strings.HasPrefix(st, "账号已登录") {
				return "", fmt.Errorf("账号通道未登录")
			}
			body["identity"] = "user"
		}
		out, err := PostJSON(ctx, f.Target, body)
		if err != nil {
			return normalizeFeatureResponse(out, traceID, err), err
		}
		return normalizeFeatureResponse(out, traceID, nil), nil
	case "script":
		return runScriptFileCtx(f.Target, args, chatID, "")
	default:
		return "", fmt.Errorf("未知功能类型 %s", f.Kind)
	}
}

func runBuiltin(target, args string) string {
	switch target {
	case "status":
		return fmt.Sprintf("%s | %s | 任务队列就绪", jiyingState(), UserSessionStatus())
	case "logs":
		return "日志目录 " + filepath.Join(utils.GetDataHome(), "logs")
	case "help":
		return "所有人：帮助、状态、功能列表，以及已授权功能。主控：任务列表、定时列表、插件列表、安装功能、安装脚本、启用功能、停用功能、增加定时、删除定时。触发：跑 <功能名> <参数>"
	case "plugins":
		return listPluginTitles()
	case "cron":
		return listCronTitles()
	case "tasks":
		return ListTasks(20)
	default:
		return "未知内置功能 " + target + " " + args
	}
}

// normalizeFeatureResponse 把 HTTP 功能返回值统一成 {ok,code,message,data,trace_id}。
// 目标若已返回该结构则透传并补齐 trace_id；否则整体包进 data。
func normalizeFeatureResponse(raw []byte, traceID string, callErr error) string {
	envelope := map[string]any{
		"ok":       callErr == nil,
		"code":     0,
		"message":  "",
		"trace_id": traceID,
	}
	if callErr != nil {
		envelope["code"] = 1
		envelope["message"] = callErr.Error()
	}
	var parsed map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &parsed) == nil {
		if _, ok := parsed["ok"]; ok {
			if _, has := parsed["trace_id"]; !has {
				parsed["trace_id"] = traceID
			}
			out, err := json.Marshal(parsed)
			if err == nil {
				return string(out)
			}
		}
		envelope["data"] = parsed
	} else if len(raw) > 0 {
		envelope["data"] = string(raw)
	}
	out, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Sprintf("{\"ok\":false,\"code\":1,\"message\":\"响应序列化失败\",\"trace_id\":%q}", traceID)
	}
	return string(out)
}

func listPluginTitles() string {
	var names []string
	for _, f := range FunctionsSnapshot() {
		if f != nil && !f.Origin {
			st := "启用"
			if f.Disable {
				st = "停用"
			}
			names = append(names, f.Title+"("+st+")")
		}
	}
	if len(names) == 0 {
		return "没有脚本插件"
	}
	return strings.Join(names, "、")
}

func ListCronTitles() string {
	return listCronTitles()
}

func listCronTitles() string {
	if ccs == nil {
		return "没有定时任务"
	}
	ccs.RLock()
	defer ccs.RUnlock()
	if len(ccs.Pts) == 0 {
		return "没有定时任务"
	}
	var b strings.Builder
	for _, c := range ccs.Pts {
		if c == nil {
			continue
		}
		fmt.Fprintf(&b, "%s %s\n", c.Name, c.Cron)
	}
	if b.Len() == 0 {
		return "没有定时任务"
	}
	return strings.TrimSpace(b.String())
}

func runScriptFile(rel, args string) (string, error) {
	return runScriptFileCtx(rel, args, "", "")
}

// runScriptFileCtx 在带会话上下文的运行时里执行脚本，便于脚本回发消息。
func runScriptFileCtx(rel, args, convID, replyTo string) (string, error) {
	full := safeScriptPath(rel)
	if full == "" {
		return "", fmt.Errorf("非法脚本路径")
	}
	cfgObj, cfgEnv := pluginRuntimeConfig(filepath.Base(rel))
	if scriptLangOf(full) == "python" {
		return runPythonScript(full, args, convID, replyTo, cfgEnv)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return evalScriptCtx(string(data), args, convID, replyTo, cfgObj)
}

// evalScriptCtx 在独立 goja 运行时执行脚本，并暴露可选的会话级发送 API。
func evalScriptCtx(code, args, convID, replyTo string, cfg map[string]any) (string, error) {
	rt := goja.New()
	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	reg.Enable(rt)
	console.Enable(rt)
	registerBasicApis(rt)
	registerSendApis(rt, convID, replyTo)
	setupCommonJS(rt)
	_ = rt.Set("args", args)
	_ = rt.Set("result", "")
	if cfg == nil {
		cfg = map[string]any{}
	}
	_ = rt.Set("config", cfg)
	if _, err := runWithTimeout(rt, code, scriptTimeout); err != nil {
		return "", err
	}
	v := rt.Get("result")
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "ok", nil
	}
	return v.String(), nil
}

// setupCommonJS 为 goja 运行时提供 CommonJS 的 module/exports 全局，
// 让脚本与技能可以自然地用 module.exports 导出，返回 module 以便取回导出值。
func setupCommonJS(rt *goja.Runtime) *goja.Object {
	module := rt.NewObject()
	exports := rt.NewObject()
	_ = module.Set("exports", exports)
	_ = rt.Set("module", module)
	_ = rt.Set("exports", exports)
	return module
}

func safeScriptPath(name string) string {
	base := filepath.Base(name)
	if base != name || base == "." || base == ".." || strings.Contains(name, "..") {
		return ""
	}
	if scriptLangOf(base) == "" {
		return ""
	}
	return filepath.Join(scriptsDir(), base)
}

func InstallScript(name, content string) error {
	full := safeScriptPath(name)
	if full == "" {
		return fmt.Errorf("只接受 scripts 目录下的 .js 或 .py 文件名")
	}
	lang := scriptLangOf(full)
	if lang == "js" && strings.Contains(content, "..") && strings.Contains(content, "readFile") {
		return fmt.Errorf("脚本含禁止路径")
	}
	if err := os.MkdirAll(scriptsDir(), 0o755); err != nil {
		return err
	}
	// 第一步：先写临时文件，在临时文件上按语言试跑，失败直接返回，线上目录保持原样。
	tmp, err := os.CreateTemp(scriptsDir(), ".install-*.tmp")
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
	if lang == "python" {
		if err := validatePythonScript(tmpName); err != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("试跑失败: %w", err)
		}
		if _, err := runPythonScript(tmpName, "", "", "", configEnvFor(configDefaults(ParsePluginManifest(content).Config))); err != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("试跑失败: %w", err)
		}
	} else {
		if _, err := trialRunScript(content, ""); err != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("试跑失败: %w", err)
		}
	}
	// 第二步：原子改名落到 scripts 目录，避免中途崩溃留下半个脚本。
	prev, hadPrev := readFileIfExists(full)
	if err := os.Rename(tmpName, full); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	// 第三步：热加载进线上引擎；失败则回滚到安装前的状态。
	if ScriptEngine != nil {
		if err := ReloadScript(filepath.Base(name)); err != nil {
			rollbackScript(full, prev, hadPrev)
			return err
		}
	}
	title, scope := scriptFeatureNameScope(name, content)
	features.ensure(title, "script", filepath.Base(name), scope, false, "app")
	return nil
}

// trialRunScript 在独立 goja 运行时里试跑脚本，不接触线上引擎与命令表。
func trialRunScript(content, args string) (string, error) {
	rt := goja.New()
	reg := require.NewRegistry(require.WithGlobalFolders(scriptFolders()...))
	reg.Enable(rt)
	console.Enable(rt)
	registerBasicApis(rt)
	registerSendApis(rt, "", "")
	setupCommonJS(rt)
	fields := ParsePluginManifest(content).Config
	_ = rt.Set("args", args)
	_ = rt.Set("result", "")
	_ = rt.Set("config", configRuntimeValues(configDefaults(fields), fields))
	if _, err := runWithTimeout(rt, content, scriptTimeout); err != nil {
		return "", err
	}
	v := rt.Get("result")
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return "", nil
	}
	return v.String(), nil
}

func readFileIfExists(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// rollbackScript 恢复安装前的脚本内容；此前不存在则删除，并同步命令表。
func rollbackScript(full string, prev []byte, hadPrev bool) {
	base := filepath.Base(full)
	if hadPrev {
		_ = os.WriteFile(full, prev, 0o644)
		if ScriptEngine != nil {
			_ = ReloadScript(base)
		}
		return
	}
	_ = os.Remove(full)
	UnloadScript(base)
}

func IsMaster(userID string) bool {
	if userID == "" || userID == "system" {
		return userID == "system"
	}
	if BucketApp == nil {
		return false
	}
	raw := BucketApp.GetString("masters")
	if raw == "" {
		return false
	}
	for _, id := range strings.Split(raw, ",") {
		if strings.TrimSpace(id) == userID {
			return true
		}
	}
	return false
}

func ListMasters() []string {
	if BucketApp == nil {
		return []string{}
	}
	seen := map[string]bool{}
	masters := make([]string, 0)
	for _, id := range strings.Split(BucketApp.GetString("masters"), ",") {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			masters = append(masters, id)
		}
	}
	sort.Strings(masters)
	return masters
}

func AddMaster(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("缺少用户 ID")
	}
	masters := ListMasters()
	for _, id := range masters {
		if id == userID {
			return nil
		}
	}
	masters = append(masters, userID)
	sort.Strings(masters)
	BucketApp.Set("masters", strings.Join(masters, ","))
	return nil
}

func RemoveMaster(userID string) error {
	userID = strings.TrimSpace(userID)
	masters := ListMasters()
	kept := make([]string, 0, len(masters))
	found := false
	for _, id := range masters {
		if id == userID {
			found = true
			continue
		}
		kept = append(kept, id)
	}
	if !found {
		return fmt.Errorf("主人不存在")
	}
	BucketApp.Set("masters", strings.Join(kept, ","))
	return nil
}

func ParseAgent(msg string) (feature, args string, ok bool) {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "跑 ") || strings.HasPrefix(msg, "跑\t") {
		rest := strings.TrimSpace(msg[len("跑"):])
		parts := strings.SplitN(rest, " ", 2)
		if parts[0] == "" {
			return "", "", false
		}
		if len(parts) == 2 {
			return parts[0], parts[1], true
		}
		return parts[0], "", true
	}
	if f, exists := GetFeature(msg); exists {
		return f.Name, "", true
	}
	return "", "", false
}
