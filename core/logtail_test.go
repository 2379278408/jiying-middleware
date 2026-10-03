package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadTailLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "day.log")
	var sb []byte
	for i := 1; i <= 5000; i++ {
		sb = append(sb, []byte(fmt.Sprintf("line-%04d\n", i))...)
	}
	if err := os.WriteFile(path, sb, 0o644); err != nil {
		t.Fatal(err)
	}

	// 完整读取（上限远大于文件）应返回最后 tail 行且顺序正确。
	full, err := readTailLines(path, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"line-4998", "line-4999", "line-5000"}; !reflect.DeepEqual(full, want) {
		t.Fatalf("full tail = %v, want %v", full, want)
	}

	// 上限小于文件时应只扫描末尾字节，丢弃被截断的首行残段。
	partial, err := readTailLines(path, 2, 64)
	if err != nil {
		t.Fatal(err)
	}
	if len(partial) != 2 || partial[0] != "line-4999" || partial[1] != "line-5000" {
		t.Fatalf("partial tail = %v, want [line-4999 line-5000]", partial)
	}

	// 尾部无换行结束也要能取到完整行。
	if err := os.WriteFile(path, []byte("a\nb\nc"), 0o644); err != nil {
		t.Fatal(err)
	}
	nobreak, err := readTailLines(path, 2, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"b", "c"}; !reflect.DeepEqual(nobreak, want) {
		t.Fatalf("no-break tail = %v, want %v", nobreak, want)
	}

	// 空文件返回空切片。
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err := readTailLines(path, 10, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty = %v, want []", empty)
	}
}
