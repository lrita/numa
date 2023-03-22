//go:build !amd64
// +build !amd64

package numa

import (
	"syscall"
	"unsafe"
)

// GetCPUAndNode returns the node id and cpu id which current caller running on.
// https://man7.org/linux/man-pages/man2/getcpu.2.html
func GetCPUAndNode() (cpu int, node int) {
	_, _, errno := syscall.RawSyscall(syscall.SYS_GETCPU,
		uintptr(unsafe.Pointer(&cpu)),
		uintptr(unsafe.Pointer(&node)),
		0)
	if errno != 0 {
		cpu = 0
		node = 0
	}
	return
}
