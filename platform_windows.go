//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32DLL      = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32DLL.NewProc("CreateMutexW")
	user32DLL        = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW  = user32DLL.NewProc("MessageBoxW")
)

// hideConsoleWindow 让子进程不弹出控制台窗口。
// yt-dlp / ffmpeg 都是控制台程序，直接 exec 会闪出一个黑框，
// 这里挂上 Windows 的 CREATE_NO_WINDOW 语义（SysProcAttr.HideWindow）。
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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
	text, _ := syscall.UTF16PtrFromString("DownTube 已经在运行了。\n请切换到任务栏中已打开的窗口。")
	title, _ := syscall.UTF16PtrFromString("DownTube")
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x40)
}
