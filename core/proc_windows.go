//go:build windows

package core

import "os/exec"

func configureProcessGroup(cmd *exec.Cmd) {}
