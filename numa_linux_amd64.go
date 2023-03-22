package numa

import (
	"github.com/intel-go/cpuid"
)

var fastway = cpuid.HasFeature(cpuid.RDTSCP)

func getcpu()

// GetCPUAndNode returns the node id and cpu id which current caller running on.
// https://man7.org/linux/man-pages/man2/getcpu.2.html
//
// equal:
//
// if fastway {
// 	call RDTSCP
//  The linux kernel will fill the node cpu id in the private data of each cpu.
//  arch/x86/kernel/vsyscall_64.c@vsyscall_set_cpu
// }
// call vdsoGetCPU
func GetCPUAndNode() (cpu int, node int)
