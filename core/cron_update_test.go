package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func TestUpdateCronCmdReschedulesAndValidates(t *testing.T) {
	withCronTestStore(t)
	ccs.cronSched = cron.New(cron.WithParser(cronSpecParser))
	ccs.cronSched.Start()
	t.Cleanup(func() { ccs.cronSched.Stop() })

	cmd := AddCronCmdFrom(map[string]interface{}{
		"name": "巡检", "command": "status", "cron": "0 */5 * * * *",
	}, "ops1", RoleOps)
	if cmd == nil {
		t.Fatal("创建定时任务失败")
	}
	oldEntry := cmd.CronEntityId
	if oldEntry == 0 || len(ccs.cronSched.Entries()) != 1 {
		t.Fatalf("初始调度未挂载: entry=%d", oldEntry)
	}

	updated, err := UpdateCronCmd(cmd.ID, map[string]interface{}{
		"name": "巡检2", "command": "help", "cron": "0 */10 * * * *",
	}, "ops1", RoleOps)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "巡检2" || updated.Command != "help" || updated.Cron != "0 */10 * * * *" {
		t.Fatalf("字段未更新: %+v", updated)
	}
	if updated.CronEntityId == 0 || updated.CronEntityId == oldEntry {
		t.Fatalf("更新后应重新挂载调度: old=%d new=%d", oldEntry, updated.CronEntityId)
	}
	if len(ccs.cronSched.Entries()) != 1 {
		t.Fatalf("旧调度应被移除，剩余 %d 条", len(ccs.cronSched.Entries()))
	}

	if _, err := UpdateCronCmd(cmd.ID, map[string]interface{}{"cron": "bad spec"}, "ops1", RoleOps); err == nil {
		t.Fatal("非法表达式应被拒绝")
	}
	if _, err := UpdateCronCmd(cmd.ID, map[string]interface{}{"command": `{"type":"feature","feature":"不存在"}`}, "ops1", RoleOps); err == nil {
		t.Fatal("不存在的功能应被拒绝")
	}
	if _, err := UpdateCronCmd(cmd.ID, map[string]interface{}{"command": "help"}, "ops2", RoleOps); err == nil {
		t.Fatal("其他运维不应修改他人定时任务")
	}
	if cur := findCronByIDLocked(cmd.ID); cur == nil || cur.Name != "巡检2" || cur.Cron == "bad spec" {
		t.Fatalf("失败更新不应改动原任务: %+v", cur)
	}
}

func TestCronSnapshotScopesByOwner(t *testing.T) {
	withCronTestStore(t)
	AddCronCmdFrom(map[string]interface{}{"name": "a", "command": "status", "cron": "0 * * * *"}, "ops1", RoleOps)
	AddCronCmdFrom(map[string]interface{}{"name": "b", "command": "status", "cron": "0 * * * *"}, "ops2", RoleOps)
	AddCronCmdFrom(map[string]interface{}{"name": "c", "command": "status", "cron": "0 * * * *"}, "admin", RoleMaster)

	if got := CronCmdsSnapshotFor("ops1", RoleOps); len(got) != 1 || got[0].Name != "a" {
		t.Fatalf("ops1 只应看到自己的任务: %+v", got)
	}
	if got := CronCmdsSnapshotFor("admin", RoleMaster); len(got) != 3 {
		t.Fatalf("主控应看到全部任务: %+v", got)
	}
}

func TestCronUpdateAPIPartialAndOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withCronTestStore(t)
	cmd := AddCronCmdFrom(map[string]interface{}{
		"name": "巡检", "command": "status", "cron": "0 */5 * * * *",
	}, "ops1", RoleOps)
	if cmd == nil {
		t.Fatal("创建定时任务失败")
	}
	ccs.Lock()
	cmd.GroupManager = []string{"conv-1"}
	ccs.Unlock()

	post := func(user, scope, body string) (int, string) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/jiying/api/cron/update", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set(ctxAPIKeyUser, user)
		c.Set(ctxAPIKeyScope, scope)
		jiyingAPICronUpdate(c)
		var resp struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp.Code, rec.Body.String()
	}

	if code, body := post("ops2", RoleOps, "id="+cmd.ID+"&name=劫持"); code == 200 {
		t.Fatalf("其他运维不应修改他人任务: %s", body)
	}
	if code, body := post("ops1", RoleOps, "id="+cmd.ID+"&name=巡检2"); code != 200 {
		t.Fatalf("本人应能只改名称: %s", body)
	}
	got := findCronByIDLocked(cmd.ID)
	if got == nil || got.Name != "巡检2" || got.Command != "status" || got.Cron != "0 */5 * * * *" {
		t.Fatalf("未传字段应保持原值: %+v", got)
	}
	if code, body := post("ops1", RoleOps, "id="+cmd.ID+"&chat_id="); code != 200 {
		t.Fatalf("应能清空会话 ID: %s", body)
	}
	if got = findCronByIDLocked(cmd.ID); got == nil || len(got.GroupManager) != 0 {
		t.Fatalf("会话 ID 未清空: %+v", got)
	}
}
