//go:build linux

package repl

import (
	"os"
	"sync"
	"syscall"
	"unsafe"
)

type termios struct {
	Iflag uint32
	Oflag uint32
	Cflag uint32
	Lflag uint32
	Line  byte
	Cc    [32]byte
}

func openKeys() (<-chan key, func(), bool) {
	fd := int(os.Stdin.Fd())
	var old termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x5401, uintptr(unsafe.Pointer(&old))); errno != 0 {
		return nil, func() {}, false
	}
	raw := old
	raw.Lflag &^= 0x0002 | 0x0008 // ICANON | ECHO
	raw.Cc[6] = 1                 // VMIN
	raw.Cc[5] = 0                 // VTIME
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x5402, uintptr(unsafe.Pointer(&raw))); errno != 0 {
		return nil, func() {}, false
	}
	done := make(chan struct{})
	ch := make(chan key, 32)
	go readUnixKeys(done, ch)
	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(done)
			syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0x5402, uintptr(unsafe.Pointer(&old)))
		})
	}
	return ch, stop, true
}

func readUnixKeys(done <-chan struct{}, ch chan<- key) {
	buf := make([]byte, 1)
	for {
		select {
		case <-done:
			return
		default:
		}
		n, err := os.Stdin.Read(buf)
		if n != 1 || err != nil {
			return
		}
		k, ok := unixKey(buf[0], done)
		if !ok {
			continue
		}
		select {
		case <-done:
			return
		case ch <- k:
		}
	}
}

func unixKey(b byte, done <-chan struct{}) (key, bool) {
	if b == '\r' || b == '\n' {
		return key{kind: 'E'}, true
	}
	if b != 0x1b {
		return key{}, false
	}
	var seq [2]byte
	for i := 0; i < 2; i++ {
		select {
		case <-done:
			return key{}, false
		default:
		}
		n, err := os.Stdin.Read(seq[i : i+1])
		if n != 1 || err != nil {
			return key{}, false
		}
	}
	if seq[0] != '[' {
		return key{}, false
	}
	switch seq[1] {
	case 'A':
		return key{kind: 'U'}, true
	case 'B':
		return key{kind: 'D'}, true
	default:
		return key{}, false
	}
}
