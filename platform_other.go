//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// hideConsoleWindow 在非 Windows 平台是空操作。
// 存在的意义是让 main.go 能在 Linux 构建节点上完成 wails 的绑定生成
// （那一步会按**宿主机**平台编译一遍 main.go，而不是按交叉编译的目标平台）。
func hideConsoleWindow(_ *exec.Cmd) {}

// killProcessTree 在非 Windows 平台退化为只终止父进程。
// Unix 上父进程退出时子进程会被 init 收养并继续运行，因此这里尽力递归杀一次子进程。
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	// Unix 上没有 taskkill /T 的等价物；先记录子进程再杀，避免子进程被 init 收养后失控
	if children, err := childPIDs(cmd.Process.Pid); err == nil {
		_ = cmd.Process.Kill()
		for _, pid := range children {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		return
	}
	_ = cmd.Process.Kill()
}

// childPIDs 列出直接子进程;拿不到时返回错误，由调用方退化为只杀父进程。
func childPIDs(pid int) ([]int, error) {
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		return nil, err
	}
	var out2 []int
	for _, line := range strings.Fields(string(out)) {
		if n, err := strconv.Atoi(line); err == nil {
			out2 = append(out2, n)
		}
	}
	return out2, nil
}

// acquireSingleInstance 在非 Windows 平台不做限制，始终认为当前是唯一实例。
func acquireSingleInstance() (func(), bool) {
	return func() {}, true
}

// showAlreadyRunningDialog 在非 Windows 平台没有可弹的原生对话框。
func showAlreadyRunningDialog() {}

// launchInstallerElevated 应用内更新仅支持 Windows。
func launchInstallerElevated(_ string) error {
	return errors.New("当前平台不支持应用内更新")
}
