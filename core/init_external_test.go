package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jiying/middleware/utils"
)

func TestAddWatchDir(t *testing.T) {
	w := NewAutWatcher()
	if w == nil {
		t.Fatal("NewAutWatcher 失败")
	}
	defer w.watcher.Close()

	tmp := t.TempDir()
	sub := filepath.Join(tmp, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	w.AddWatchDir(tmp)

	list := w.watcher.WatchList()
	if !containsPath(list, tmp) {
		t.Fatalf("根目录未注册: %v", list)
	}
	if !containsPath(list, sub) {
		t.Fatalf("子目录未递归注册: %v", list)
	}
}

func containsPath(list []string, target string) bool {
	for _, p := range list {
		got, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if got == target {
			return true
		}
	}
	return false
}

func TestWatchEventFires(t *testing.T) {
	w := NewAutWatcher()
	if w == nil {
		t.Fatal("NewAutWatcher 失败")
	}
	defer w.watcher.Close()

	tmp := t.TempDir()
	w.AddWatchDir(tmp)

	fired := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		w.WatchFileEvents(func(path string) {
			fired <- path
		})
		close(done)
	}()

	target := filepath.Join(tmp, "demo.js")
	if err := os.WriteFile(target, []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case p := <-fired:
		if !strings.HasSuffix(p, "demo.js") {
			t.Fatalf("回调路径错误: %s", p)
		}
	case <-done:
		t.Fatal("事件循环提前退出")
	}
}

func TestIsExcludeDir(t *testing.T) {
	cases := map[string]bool{
		"/data/plugin/.git/config":  true,
		"/data/plugin/node_modules": true,
		"/data/plugin/@tmpfs":       true,
		"/data/plugin/scripts/x.js": false,
	}
	for p, want := range cases {
		if got := isExcludeDir(p); got != want {
			t.Errorf("isExcludeDir(%q)=%v, want %v", p, got, want)
		}
	}
}

func TestRegisterExternalPlugin(t *testing.T) {
	start := len(Functions)
	defer func() { Functions = Functions[:start] }()

	tmp := t.TempDir()
	dir := filepath.Join(tmp, "plugin", "encrypted_ext")
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "extdemo.js")
	if err := os.WriteFile(p, []byte("cmds=[];"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("JIYING_RES", tmp)

	before := len(Functions)
	loadExternalPlugins()
	if len(Functions) != before+1 {
		t.Fatalf("外部插件未登记: before=%d after=%d", before, len(Functions))
	}
	var found bool
	for _, f := range Functions {
		if f != nil && f.PluginPath == p {
			found = true
			if f.Title != "extdemo" || f.PluginLang != "js" {
				t.Fatalf("外部插件字段错误: %+v", f)
			}
		}
	}
	if !found {
		t.Fatal("Functions 中未找到登记的外部插件")
	}
}

func TestReadFilePaths(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(tmp, "x.js"), []byte{1}, 0o644)
	_ = os.WriteFile(filepath.Join(tmp, "a", "y.py"), []byte{2}, 0o644)

	paths := utils.ReadFilePaths(tmp)
	if len(paths) != 2 {
		t.Fatalf("ReadFilePaths 期望 2 个文件, got %d: %v", len(paths), paths)
	}
	if !utils.IsDir(tmp) || utils.IsDir(filepath.Join(tmp, "x.js")) {
		t.Fatal("IsDir 判定错误")
	}
}
