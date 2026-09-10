//go:build windows

package tunnel

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// On Windows, background SSH tunnels run without Unix pgid.
}

func killProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}

func killPID(pid int) {
	if pid <= 0 {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}

func isProcessAlive(proc *os.Process) bool {
	if proc == nil || proc.Pid <= 0 {
		return false
	}
	const stillActive = 259
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}
