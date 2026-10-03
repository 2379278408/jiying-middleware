package gin

import (
	"reflect"
	"testing"
)

func TestDepInstallArgsRejectsUnsafeName(t *testing.T) {
	for _, name := range []string{
		"",
		"pkg; rm -rf /",
		"pkg && curl http://evil",
		"pkg | sh",
		"pkg`id`",
		"$(whoami)",
		"-rf",
		"--prefix=/tmp",
		"pkg name",
		"pkg\nnext",
	} {
		if got := depInstallArgs("nodejs", name); got != nil {
			t.Errorf("depInstallArgs(nodejs, %q) = %v, want nil", name, got)
		}
	}
}

func TestDepInstallArgsBuildsArgv(t *testing.T) {
	cases := []struct {
		typ, name string
		want      []string
	}{
		{"nodejs", "lodash", []string{"npm", "install", "-g", "lodash"}},
		{"nodejs", "@scope/pkg", []string{"npm", "install", "-g", "@scope/pkg"}},
		{"python", "requests", []string{"pip3", "install", "requests"}},
		{"golang", "github.com/x/y@latest", []string{"go", "install", "github.com/x/y@latest"}},
		{"linux", "curl", []string{"apt-get", "install", "-y", "curl"}},
	}
	for _, c := range cases {
		if got := depInstallArgs(c.typ, c.name); !reflect.DeepEqual(got, c.want) {
			t.Errorf("depInstallArgs(%q, %q) = %v, want %v", c.typ, c.name, got, c.want)
		}
	}
	if got := depInstallArgs("unknown", "pkg"); got != nil {
		t.Errorf("未知类型应返回 nil, got %v", got)
	}
}
