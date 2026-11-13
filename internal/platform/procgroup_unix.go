//go:build !windows

package platform

import (
	"os/exec"
	"syscall"
)

// ConfigureProcessGroup 让 cmd 在独立进程组中运行，并在 context 取消（如 hook 超时）时
// 终止整个进程组，而不只是直接子进程。否则 shell 脚本派生的子进程会在超时后继续运行，
// 并因持有 stdout/stderr 而让 Wait 一直阻塞、占住执行槽位。
// 必须在 cmd.Start 之前调用，且 cmd 需由 exec.CommandContext 创建。
func ConfigureProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// 负 PID 表示向整个进程组发送信号；进程组不存在时退回到只杀直接子进程。
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
