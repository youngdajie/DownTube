//go:build windows

package main

import (
	"errors"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	kernel32DLL      = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32DLL.NewProc("CreateMutexW")
	user32DLL        = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW  = user32DLL.NewProc("MessageBoxW")
	shell32DLL       = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32DLL.NewProc("ShellExecuteW")
)

const (
	swShowNormal   = 1
	shellExecError = 32 // ShellExecuteW 返回值 <= 32 表示失败
)

// launchInstallerElevated 以管理员权限启动安装包(会弹 UAC 确认),
// 用 ShellExecuteW + "runas" 实现 —— exec.Command 无法触发提权。
func launchInstallerElevated(path string) error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	// 静默安装:不弹安装向导,安装完成后由 NSIS 脚本自动重启应用。
	params, err := syscall.UTF16PtrFromString("/S")
	if err != nil {
		return err
	}
	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		swShowNormal,
	)
	if ret <= shellExecError {
		return errors.New("启动安装程序失败，可能取消了管理员授权")
	}
	return nil
}

// hideConsoleWindow 让子进程不弹出控制台窗口。
// yt-dlp / ffmpeg 都是控制台程序，直接 exec 会闪出一个黑框，
// 这里挂上 Windows 的 CREATE_NO_WINDOW 语义（SysProcAttr.HideWindow）。
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// killProcessTree 终结 cmd 及其所有后代进程。
//
// 为什么不能只靠 exec.CommandContext:它只终止直接子进程。Windows 上父进程被杀后
// 子进程会继续存活 —— yt-dlp 在分片下载(DASH/HLS)或合并时会拉起 ffmpeg,真正在下载的
// 往往正是那个 ffmpeg。更糟的是它继承了 stdout/stderr 管道写端,父进程死了管道仍不 EOF,
// cmd.Wait() 会一直阻塞,任务并发槽位再也释放不了,表现就是「清空队列后还在后台下载,
// 直到关闭程序才停」。
//
// taskkill /T 的树终止同时解决两件事:杀掉子进程,以及让管道写端全部关闭。
func killProcessTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	// 先尝试连树一起杀;失败(进程已退出/taskkill 不可用)再退回只杀父进程
	if err := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run(); err == nil {
		return
	}
	_ = cmd.Process.Kill()
}

// acquireSingleInstance 通过命名互斥量保证同一时间只运行一个 DownTube 实例，
// 避免多实例读写同一份 state.json 导致任务与设置互相覆盖。
//
// 返回一个释放函数：进程正常退出时由调用方 defer 调用。
// 注意**不要**在互斥量还持有时就释放 —— CreateMutexW 的句柄由内核按进程
// 引用计数维护，进程退出会自动释放，这里只是显式收一下。
func acquireSingleInstance() (func(), bool) {
	name, err := syscall.UTF16PtrFromString("Local\\DownTube-SingleInstance")
	if err != nil {
		return func() {}, true
	}
	handle, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return func() {}, true
	}
	if errno, ok := lastErr.(syscall.Errno); ok && errno == syscall.ERROR_ALREADY_EXISTS {
		syscall.CloseHandle(syscall.Handle(handle))
		return func() {}, false
	}
	h := syscall.Handle(handle)
	return func() { syscall.CloseHandle(h) }, true
}

func showAlreadyRunningDialog() {
	text, _ := syscall.UTF16PtrFromString("DownTube 已经在运行了。\n请切换到任务栏中已打开的窗口")
	title, _ := syscall.UTF16PtrFromString("DownTube")
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x40)
}
