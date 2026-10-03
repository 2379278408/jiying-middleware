package core

// 消息触发的执行（命令与 Agent 回复）并发上限：上游应用通道可能瞬间推入大量消息，
// 每条都派生 goroutine 执行脚本或调用模型，不设上限会把上游与本地资源一起打爆。
// 取不到槽位时立即回「系统繁忙」并确认消息，避免消息在通道里堆积重放。
const defaultMsgExecConcurrency = 8

// 阈值可用 JIYING_MSG_EXEC_CONCURRENCY 覆盖，便于按部署规模调整。
var msgExecSem = make(chan struct{}, envInt("JIYING_MSG_EXEC_CONCURRENCY", defaultMsgExecConcurrency))

func tryAcquireMsgExec() bool {
	select {
	case msgExecSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseMsgExec() {
	<-msgExecSem
}
