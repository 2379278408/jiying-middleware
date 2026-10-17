package core

import (
	"context"
	"testing"
)

func withCronTestStore(t *testing.T) {
	t.Helper()
	backup := ccs
	ccs = &cronCmdContainer{}
	t.Cleanup(func() { ccs = backup })
}

func TestCronIdentity(t *testing.T) {
	cases := []struct {
		name string
		cmd  *CronCmd
		want string
	}{
		{"主控任务以系统身份执行", &CronCmd{Owner: "admin", OwnerRole: RoleMaster}, "system"},
		{"运维任务以本人身份执行", &CronCmd{Owner: "ops1", OwnerRole: RoleOps}, "ops1"},
		{"只读任务以本人身份执行", &CronCmd{Owner: "viewer", OwnerRole: RoleReadonly}, "viewer"},
		{"旧数据无角色按归属执行", &CronCmd{Owner: "ops1"}, "ops1"},
		{"旧系统任务保留系统身份", &CronCmd{Owner: "system"}, "system"},
		{"空缺归属回落系统", &CronCmd{}, "system"},
	}
	for _, tc := range cases {
		if got := cronIdentity(tc.cmd); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestAddCronCmdFromBindsSessionIdentity(t *testing.T) {
	withCronTestStore(t)
	cmd := AddCronCmdFrom(map[string]interface{}{
		"name":       "巡检",
		"command":    "status",
		"cron":       "0 * * * *",
		"owner":      "system",
		"owner_role": RoleMaster,
	}, "ops1", RoleOps)
	if cmd == nil {
		t.Fatal("创建定时任务失败")
	}
	if cmd.Owner != "ops1" || cmd.OwnerRole != RoleOps {
		t.Fatalf("归属未绑定会话身份: %+v", cmd)
	}
	if got := cronIdentity(cmd); got != "ops1" {
		t.Fatalf("执行身份应绑定创建者，got %q", got)
	}
	if AddCronCmdFrom(map[string]interface{}{"name": "x"}, "", "") != nil {
		t.Fatal("缺少操作者时不应创建定时任务")
	}
}

func TestCronManagePermission(t *testing.T) {
	withCronTestStore(t)
	cmd := AddCronCmdFrom(map[string]interface{}{
		"name": "巡检", "command": "status", "cron": "0 * * * *",
	}, "ops1", RoleOps)
	if cmd == nil {
		t.Fatal("创建定时任务失败")
	}
	if UpdateCronCmdFrom(cmd.ID, map[string]interface{}{"command": "help"}, "ops2", RoleOps) != nil {
		t.Fatal("其他运维不应修改他人定时任务")
	}
	if DeleteCronCmdsByIds([]string{cmd.ID}, "ops2", RoleOps) != 0 {
		t.Fatal("其他运维不应删除他人定时任务")
	}
	if SetCronCmdStatus(cmd.ID, false, "ops2", RoleOps) {
		t.Fatal("其他运维不应停用他人定时任务")
	}
	if _, err := executeFeature(context.Background(), "cron:巡检", "", "ops2", ""); err == nil {
		t.Fatal("非归属用户不应触发他人定时任务")
	}
	if UpdateCronCmdFrom(cmd.ID, map[string]interface{}{"command": "help"}, "admin", RoleMaster) == nil {
		t.Fatal("主控应能修改任意定时任务")
	}
	if !SetCronCmdStatus(cmd.ID, false, "admin", RoleMaster) {
		t.Fatal("主控应能停用任意定时任务")
	}
	if DeleteCronCmdsByIds([]string{cmd.ID}, "admin", RoleMaster) != 1 {
		t.Fatal("主控应能删除任意定时任务")
	}
}

func TestExecuteFeatureCronRequiresOwner(t *testing.T) {
	withCronTestStore(t)
	cmd := AddCronCmdFrom(map[string]interface{}{
		"name": "巡检", "command": "status", "cron": "0 * * * *",
	}, "ops1", RoleOps)
	if cmd == nil {
		t.Fatal("创建定时任务失败")
	}
	if _, err := executeFeature(context.Background(), "cron:巡检", "", "ops1", ""); err != nil {
		t.Fatalf("归属用户应能触发: %v", err)
	}
	if _, err := executeFeature(context.Background(), "cron:不存在", "", "ops1", ""); err == nil {
		t.Fatal("不存在的定时任务不应被触发")
	}
	if _, err := executeFeature(context.Background(), "cron:巡检", "", "system", ""); err != nil {
		t.Fatalf("系统身份应能触发: %v", err)
	}
}
