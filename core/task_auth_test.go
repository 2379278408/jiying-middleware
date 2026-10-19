package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTaskFilterScopesByViewer(t *testing.T) {
	resetTaskQueue()
	alice, err := EnqueueTask(Task{Feature: "帮助", UserID: "alice", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := EnqueueTask(Task{Feature: "帮助", UserID: "bob", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(resetTaskQueue)

	own := listTaskItems(TaskFilter{Viewer: "alice", Limit: 10})
	if len(own) != 1 || own[0].ID != alice.ID {
		t.Fatalf("alice 只能看到自己的任务: %+v", own)
	}
	all := listTaskItems(TaskFilter{Viewer: "alice", All: true, Limit: 10})
	if len(all) != 2 {
		t.Fatalf("主控视角应看到全部任务: %+v", all)
	}
	// 显式 user_id 不能突破 Viewer 限制。
	none := listTaskItems(TaskFilter{Viewer: "alice", UserID: bob.ID, Limit: 10})
	if len(none) != 0 {
		t.Fatalf("viewer 过滤应优先于 user_id: %+v", none)
	}
}

func TestTaskActionsRequireOwnership(t *testing.T) {
	resetTaskQueue()
	alice, err := EnqueueTask(Task{Feature: "帮助", UserID: "alice", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := EnqueueTask(Task{Feature: "帮助", UserID: "bob", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(resetTaskQueue)

	if TaskForActor(alice.ID, "bob", false) != nil {
		t.Fatal("bob 不应读取 alice 的任务")
	}
	if err := RetryTaskAs(alice.ID, "bob", false); err == nil {
		t.Fatal("bob 不应重试 alice 的任务")
	}
	if DeleteTaskAs(alice.ID, "bob", false) {
		t.Fatal("bob 不应删除 alice 的任务")
	}
	if TaskForActor(alice.ID, "admin", true) == nil {
		t.Fatal("主控应能读取任意任务")
	}
	if err := RetryTaskAs(alice.ID, "alice", false); err != nil {
		t.Fatalf("alice 应能重试自己的任务: %v", err)
	}
	WaitForTaskDone(alice.ID, 3*time.Second)
	if !DeleteTaskAs(bob.ID, "admin", true) {
		t.Fatal("主控应能删除任意任务")
	}
}

func TestTaskAPIEnforcesOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetTaskQueue()
	alice, err := EnqueueTask(Task{Feature: "帮助", UserID: "alice", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(resetTaskQueue)

	listCtx := func(user string) (*gin.Context, *httptest.ResponseRecorder) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/jiying/api/tasks", nil)
		c.Set(ctxAPIKeyUser, user)
		c.Set(ctxAPIKeyScope, RoleOps)
		return c, rec
	}
	aliceCtx, aliceRec := listCtx("alice")
	jiyingAPITasks(aliceCtx)
	var listed struct {
		Data []*Task `json:"data"`
	}
	if err := json.Unmarshal(aliceRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data) != 1 || listed.Data[0].ID != alice.ID {
		t.Fatalf("接口应只返回当前用户任务: %+v", listed.Data)
	}

	postCtx := func(user string) (*gin.Context, *httptest.ResponseRecorder) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/jiying/api/task/retry", strings.NewReader("id="+alice.ID))
		c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.Set(ctxAPIKeyUser, user)
		c.Set(ctxAPIKeyScope, RoleOps)
		return c, rec
	}
	bobCtx, bobRec := postCtx("bob")
	jiyingAPITaskRetry(bobCtx)
	var denied struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(bobRec.Body.Bytes(), &denied); err != nil {
		t.Fatal(err)
	}
	if denied.Code == 200 {
		t.Fatalf("bob 重试 alice 的任务应被拒绝: %s", bobRec.Body.String())
	}
	adminCtx, adminRec := postCtx("admin")
	adminCtx.Set(ctxAPIKeyScope, RoleMaster)
	jiyingAPITaskRetry(adminCtx)
	var retried struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(adminRec.Body.Bytes(), &retried); err != nil {
		t.Fatal(err)
	}
	if retried.Code != 200 {
		t.Fatalf("主控应能重试任意任务: %s", adminRec.Body.String())
	}
	WaitForTaskDone(alice.ID, 3*time.Second)
}
