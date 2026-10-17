package gin

import (
	"path/filepath"
	"strings"
)

const maxUploadBytes = 4 << 20

// safeUploadRel 规范化用户提交的相对路径，拒绝绝对路径、上跳与 NUL。
func safeUploadRel(raw string) (string, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" || raw == "." {
		return "", true
	}
	if strings.ContainsRune(raw, 0) {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." {
		return "", true
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", false
	}
	return clean, true
}

// safeJoinUnder 把相对路径拼到根目录下，并确认结果仍在根目录内。
func safeJoinUnder(root, rel string) (string, bool) {
	clean, ok := safeUploadRel(rel)
	if !ok {
		return "", false
	}
	full := root
	if clean != "" {
		full = filepath.Join(root, clean)
	}
	back, err := filepath.Rel(root, full)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", false
	}
	return full, true
}

// safeUploadName 只接受纯文件名并校验扩展名，避免通过表单文件名写入任意路径。
func safeUploadName(raw string, allowed ...string) (string, bool) {
	name := filepath.Base(strings.TrimSpace(raw))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, want := range allowed {
		if ext == want {
			return name, true
		}
	}
	return "", false
}
