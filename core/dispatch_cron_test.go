package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDispatchCronFeatureJSON(t *testing.T) {
	name := "测试功能-dispatch"
	_ = DeleteFeature(name)
	if err := UpsertFeature(Feature{Name: name, Kind: "builtin", Target: "help", Scope: "public", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	defer DeleteFeature(name)
	raw, _ := json.Marshal(cronFeatureCommand{Type: "feature", Feature: name})
	out, err := dispatchLocalCommandFor(string(raw), "ops-user")
	if err != nil || !strings.Contains(out, "所有人") {
		t.Fatalf("功能命令执行失败: %q %v", out, err)
	}
}

func TestDispatchCronCommandPermission(t *testing.T) {
	cmdRouter.Store("__admin_test__", func() {})
	defer cmdRouter.Delete("__admin_test__")
	registerAdminSysCommand("测试管理命令", "__admin_test__", 1, func() {})
	if _, err := dispatchLocalCommandFor("__admin_test__", "ops-user"); err == nil {
		t.Fatal("运维账号不应能执行管理命令")
	}
	if _, err := dispatchLocalCommandFor("__admin_test__", "system"); err != nil {
		t.Fatalf("系统身份应能执行管理命令: %v", err)
	}
}

func TestRunCronCmdFailure(t *testing.T) {
	backup := ccs
	ccs = &cronCmdContainer{}
	defer func() { ccs = backup }()
	cmd := &CronCmd{ID: "f1", Name: "失败任务", Command: "__missing_command__", Owner: "ops1", OwnerRole: RoleOps}
	runCronCmd(cmd)
	if cmd.LastStatus != executionFailed || cmd.FailCount != 1 || cmd.LastSuccess != "" {
		t.Fatalf("失败结果未正确回写: %+v", cmd)
	}
	cmd.Command = "__missing_command__"
	runCronCmd(cmd)
	runCronCmd(cmd)
	if cmd.Status || !strings.Contains(cmd.LastError, "已暂停") {
		t.Fatalf("连续失败未熔断: %+v", cmd)
	}
}
