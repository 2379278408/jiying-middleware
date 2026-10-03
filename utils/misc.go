package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func TrimObject(v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return
	}
	t := elem.Type()
	for i := 0; i < t.NumField(); i++ {
		f := elem.Field(i)
		if f.Kind() != reflect.String {
			continue
		}
		elem.Field(i).SetString(strings.TrimSpace(f.String()))
	}
}

func Uint64(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func UnmarshalUInt64Array(b []byte) []uint64 {
	var arr []uint64
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil
	}
	return arr
}

func v2String(s interface{}) string {
	if b, err := json.Marshal(s); err == nil {
		return string(b)
	}
	return fmt.Sprint(s)
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func ReadFilePaths(dir string) []string {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil
	}
	return paths
}

func TrimHiddenCharacter(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x200b || r == 0x200c || r == 0x200d || r == 0x2060 || r == 0xfeff:
			continue
		case r < 0x20 || r == 0x7f:
			continue
		case r >= 0xe000 && r <= 0xf8ff:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

var DefaultSeparators = []string{" ", "\n", "\t", "，", ",", "。", ".", "！", "!", "？", "?", "；", ";", "："}

func Split(s string) []string {
	var parts []string
	var cur strings.Builder
	for _, r := range s {
		if isSeparator(r) {
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func isSeparator(r rune) bool {
	for _, sp := range DefaultSeparators {
		if string(r) == sp {
			return true
		}
	}
	return r == ' ' || r == '\u3000'
}

func ExpandChineseRegexSpace(s string) string {
	return strings.ReplaceAll(s, "\u3000", "[ \\t\u3000]")
}
