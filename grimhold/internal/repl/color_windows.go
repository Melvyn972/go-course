//go:build windows

package repl

import (
	"os"
	"syscall"
	"unsafe"
)

func enableVT() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	h := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	r, _, _ := getMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return
	}
	const enableVirtualTerminalProcessing = 0x0004
	_, _, _ = setMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
}
