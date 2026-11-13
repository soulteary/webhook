//go:build windows

package platform

import "os/exec"

// ConfigureProcessGroup 在 Windows 上保持默认行为（仅终止直接子进程）。
func ConfigureProcessGroup(cmd *exec.Cmd) {}
