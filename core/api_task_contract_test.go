package core

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestJiyingJSONUsesStableEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/jiying/api/tasks", nil)
	ctx.Request.Header.Set("X-Trace-ID", "trace-contract-1")

	jiyingJSON(ctx, map[string]string{"state": "ready"})

	var body struct {
		OK      bool              `json:"ok"`
		Code    int               `json:"code"`
		Message string            `json:"message"`
		TraceID string            `json:"trace_id"`
		Data    map[string]string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || body.Code != 200 || body.Message != "" || body.TraceID != "trace-contract-1" {
		t.Fatalf("unexpected response envelope: %+v", body)
	}
	if body.Data["state"] != "ready" {
		t.Fatalf("unexpected response data: %+v", body.Data)
	}
}

func TestJiyingErrorUsesStableEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/jiying/api/task/retry", nil)

	jiyingErr(ctx, "任务不存在")

	var body struct {
		OK        bool   `json:"ok"`
		Code      int    `json:"code"`
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
		TraceID   string `json:"trace_id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.OK || body.Code != 1 || body.ErrorCode != "REQUEST_FAILED" || body.Message != "任务不存在" || body.TraceID == "" {
		t.Fatalf("unexpected error envelope: %+v", body)
	}
}

func TestTaskLifecycleStatusesAndFiltering(t *testing.T) {
	resetTaskQueue()
	confirmed, err := EnqueueTask(Task{Feature: "帮助", UserID: "user-1", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != taskWaitingConfirmation {
		t.Fatalf("expected waiting confirmation, got %q", confirmed.Status)
	}

	if err := ConfirmTask(confirmed.ID, false); err != nil {
		t.Fatal(err)
	}
	canceled := listTaskItems(TaskFilter{Status: taskCanceled, Limit: 10})
	if len(canceled) != 1 || canceled[0].ErrorCode != "TASK_CANCELED" {
		t.Fatalf("unexpected canceled task: %+v", canceled)
	}

	failed, err := EnqueueTask(Task{Feature: "失败功能", UserID: "user-2"})
	if err != nil {
		t.Fatal(err)
	}
	tasks.mu.Lock()
	tasks.items[failed.ID].Status = taskFailed
	tasks.items[failed.ID].ErrorCode = "EXECUTION_FAILED"
	tasks.mu.Unlock()
	if err := RetryTask(failed.ID); err != nil {
		t.Fatal(err)
	}
	items := listTaskItems(TaskFilter{Status: taskPending, UserID: "user-2", Limit: 10})
	if len(items) != 1 || items[0].RetryCount != 1 {
		t.Fatalf("unexpected retried task: %+v", items)
	}
}
