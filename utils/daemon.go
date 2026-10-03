package utils

import (
	"os"
	"os/exec"
)

func Daemon(uid int) {
	if len(Runnings) > 0 {
		for _, p := range Runnings {
			if p != nil {
				p.Kill()
			}
		}
		Runnings = Runnings[:0]
	}
	exe := GetExecPath()
	if exe == "" {
		return
	}
	if _, err := os.Stat(exe); err != nil {
		return
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err == nil {
		Runnings = append(Runnings, cmd.Process)
	}
}
