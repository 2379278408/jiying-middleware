package core

import (
	"sync/atomic"
	"testing"

	"github.com/robfig/cron/v3"
)

func TestCleanOnceTasks(t *testing.T) {
	backupCcs := ccs
	ccs = &cronCmdContainer{
		MaxId:     nil,
		Position:  nil,
		cronSched: cron.New(),
	}
	defer func() { ccs = backupCcs }()

	once := &CronCmd{ID: "1", Name: "一次性", IsOnce: true, LastRunningTime: "2026-08-27 10:00:00"}
	recur := &CronCmd{ID: "2", Name: "周期", IsOnce: false}
	ccs.Pts = []*CronCmd{once, recur}

	ccs.cleanOnceTasks()
	if len(ccs.Pts) != 1 || ccs.Pts[0].ID != "2" {
		t.Fatalf("一次性任务应被清理, 留存: %+v", ccs.Pts)
	}
}

func TestCleanOnceTasksRunning(t *testing.T) {
	backupCcs := ccs
	ccs = &cronCmdContainer{cronSched: cron.New()}
	defer func() { ccs = backupCcs }()

	pending := &CronCmd{ID: "3", Name: "未完成", IsOnce: true}
	ccs.Pts = []*CronCmd{pending}

	ccs.cleanOnceTasks()
	if len(ccs.Pts) != 1 {
		t.Fatalf("未完成任务不应清理")
	}
}

func TestCronCmdDispatch(t *testing.T) {
	backupCcs := ccs
	ccs = &cronCmdContainer{cronSched: cron.New()}
	defer func() { ccs = backupCcs }()

	var called atomic.Bool
	cmdRouter.Store("__cron_test__", func() { called.Store(true) })
	defer cmdRouter.Delete("__cron_test__")

	cmd := &CronCmd{ID: "9", Name: "派发", Command: "__cron_test__ job"}
	runCronCmd(cmd)

	waitTrue(&called)
	if cmd.LastRun == "" || cmd.LastSuccess == "" {
		t.Fatalf("执行时间未回写: %+v", cmd)
	}
}
