// +build !linux

package gonuma

import (
	"runtime"
	"syscall"
	"unsafe"
)

func init() {
	// only used for cross-compile
	NUMAnodemax = 1
	memnodes = NewBitmask(NodePossibleCount())
	numanodes = NewBitmask(NodePossibleCount())
	NUMAconfigurednode = setupconfigurednodes()

	memnodes.Set(0, true)
	numanodes.Set(0, true)

	NUMAcpuMax = runtime.NumCPU()
	NUMAconfiguredcpu = runtime.NumCPU()

	cpu2node = make(map[int]int, NUMAcpuMax)
	for i := 0; i < NUMAcpuMax; i++ {
		cpu2node[i] = 0
	}
	cpumask := NewBitmask(NUMAconfiguredcpu)
	for i := 0; i < NUMAconfiguredcpu; i++ {
		cpumask.Set(i, true)
	}
	node2cpu = map[int]Bitmask{0: cpumask}
}

func setupconfigurednodes() (n int) {
	for i := 0; i < NodePossibleCount(); i++ {
		numanodes.Set(i, true)
		memnodes.Set(i, true)
	}
	return NodePossibleCount()
}

// GetMemPolicy retrieves the NUMA policy of the calling process or of a
// memory address, depending on the setting of flags.
func GetMemPolicy(nodemask Bitmask, addr unsafe.Pointer, flags int) (mode int, err error) {
	return 0, syscall.ENOSYS
}

// SetMemPolicy sets the NUMA memory policy of the calling process, which
// consists of a policy mode and zero or more nodes, to the values specified
// by the mode, nodemask and maxnode arguments.
func SetMemPolicy(mode int, nodemask Bitmask) error {
	return syscall.ENOSYS
}

// NodeMemSize64 return the memory total size and free size of given node.
func NodeMemSize64(node int) (total, free int64, err error) {
	return 0, 0, syscall.ENOSYS
}

// MBind sets the NUMA memory policy, which consists of a policy mode and zero
// or more nodes, for the memory range starting with addr and continuing for
// length bytes. The memory policy defines from which node memory is allocated.
// Details to see manpage of mbind.
func MBind(addr unsafe.Pointer, length, mode, flags int, nodemask Bitmask) error {
	return syscall.ENOSYS
}

// GetSchedAffinity writes the affinity mask of the process whose ID is pid
// into the input mask. If pid is zero, then the mask of the calling process
// is returned.
func GetSchedAffinity(pid int, cpumask Bitmask) (int, error) {
	return 0, syscall.ENOSYS
}

// SetSchedAffinity sets the CPU affinity mask of the process whose ID
// is pid to the value specified by mask. If pid is zero, then the calling
// process is used.
func SetSchedAffinity(pid int, cpumask Bitmask) error {
	return syscall.ENOSYS
}

// GetCPUAndNode returns the node id and cpu id which current caller running on.
func GetCPUAndNode() (cpu, node int) {
	cpu = runtime_procPin()
	runtime_procUnpin()
	return cpu % NUMAcpuMax, NUMAnodemax - 1
}

// Implemented in runtime.

//go:linkname runtime_procPin runtime.procPin
//go:nosplit
func runtime_procPin() int

//go:linkname runtime_procUnpin runtime.procUnpin
//go:nosplit
func runtime_procUnpin()
