//go:build windows

package repl

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

func openKeys() (<-chan key, func(), bool) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	readIn := kernel32.NewProc("ReadConsoleInputW")
	cancel := kernel32.NewProc("CancelIoEx")

	h := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	r, _, _ := getMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return nil, func() {}, false
	}
	const (
		enableProcessedInput = 0x0001
		enableLineInput      = 0x0002
		enableEchoInput      = 0x0004
		enableMouseInput     = 0x0010
		enableExtendedFlags  = 0x0080
	)
	next := (mode &^ (enableLineInput | enableEchoInput | enableMouseInput)) | enableProcessedInput | enableExtendedFlags
	if r, _, _ = setMode.Call(uintptr(h), uintptr(next)); r == 0 {
		return nil, func() {}, false
	}

	done := make(chan struct{})
	ch := make(chan key, 32)
	go readWindowsKeys(h, readIn, done, ch)

	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(done)
			cancel.Call(uintptr(h), 0)
			setMode.Call(uintptr(h), uintptr(mode))
		})
	}
	return ch, stop, true
}

func readWindowsKeys(h syscall.Handle, readIn *syscall.LazyProc, done <-chan struct{}, ch chan<- key) {
	type rec struct {
		EventType uint16
		_         uint16
		KeyDown   int32
		Repeat    uint16
		VK        uint16
		Scan      uint16
		Char      uint16
		Control   uint32
	}
	for {
		select {
		case <-done:
			return
		default:
		}
		var ev rec
		var n uint32
		r, _, _ := readIn.Call(uintptr(h), uintptr(unsafe.Pointer(&ev)), 1, uintptr(unsafe.Pointer(&n)))
		if r == 0 {
			return
		}
		if n == 0 || ev.EventType != 1 || ev.KeyDown == 0 {
			continue
		}
		var k key
		switch ev.VK {
		case 0x26:
			k.kind = 'U'
		case 0x28:
			k.kind = 'D'
		case 0x0D:
			k.kind = 'E'
		default:
			continue
		}
		select {
		case <-done:
			return
		case ch <- k:
		}
	}
}
