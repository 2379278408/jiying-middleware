//go:build !windows

package core

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup 让脚本及其子进程进入独立进程组；超时或取消时整组终止，
// 避免插件派生的子进程继续占用资源。
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 2 * 1e9
}
