//go:build !windows

package main

import (
	"errors"
	"os/exec"
)

// hideConsoleWindow 在非 Windows 平台是空操作。
// 存在的意义是让 main.go 能在 Linux 构建节点上完成 wails 的绑定生成
// （那一步会按**宿主机**平台编译一遍 main.go，而不是按交叉编译的目标平台）。
func hideConsoleWindow(_ *exec.Cmd) {}

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
