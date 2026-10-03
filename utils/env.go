package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ExecPath string
	ResPath  string
	dataHome string
	Runnings []*os.Process
)

func GetExecPath() string {
	if ExecPath != "" {
		return ExecPath
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	ExecPath = exe
	return exe
}

func GetExecPathDir() string {
	if ResPath != "" {
		return ResPath
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func GetResHome() string {
	if ResPath != "" {
		return ResPath
	}
	if r := os.Getenv("JIYING_RES"); r != "" {
		return r
	}
	dir := filepath.Dir(GetExecPath())
	return dir
}

func GetDataHome() string {
	if dataHome != "" {
		return dataHome
	}
	if d := os.Getenv("JIYING_DATA"); d != "" {
		dataHome = d
		return d
	}
	return GetResHome()
}

func MkFileDir(file string) error {
	dir := filepath.Dir(file)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o777)
}

func String(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		if s == nil {
			return ""
		}
		return v2String(s)
	}
}
func GetClientIP(c interface{ RemoteIP() string }) string {
	if c == nil {
		return ""
	}
	return c.RemoteIP()
}

func Int64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
