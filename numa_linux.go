// +build linux

package gonuma

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/intel-go/cpuid"
)

func init() {
	_, _, e1 := syscall.Syscall6(
		syscall.SYS_GET_MEMPOLICY,
		0,
		0,
		0,
		0,
		0,
		0,
	)
	available = e1 != syscall.ENOSYS
	NUMAnodemax = setupnodemask() // max nodes
	memnodes = NewBitmask(NodePossibleCount())
	numanodes = NewBitmask(NodePossibleCount())
	NUMAconfigurednode = setupconfigurednodes() // configured nodes
	NUMAcpuMax = setupncpu()                    // max cpu
	NUMAconfiguredcpu = setupnconfiguredcpu()   // configured cpu
	setupconstraints()
}

// GetMemPolicy retrieves the NUMA policy of the calling process or of a
// memory address, depending on the setting of flags.
// Details to see manpage of get_mempolicy.
//
// If flags is specified as 0, then information about the calling process's
// default policy (as set by set_mempolicy(2)) is returned. The policy
// returned [mode and nodemask] may be used to restore the process's policy
// to its state at the time of the call to get_mempolicy() using
// set_mempolicy(2).
//
// If flags specifies MPolFMemsAllowed (available since Linux 2.6.24),
// the mode argument is ignored and the set of nodes [memories] that the
// process is allowed to specify in subsequent calls to mbind(2) or
// set_mempolicy(2) [in the absence of any mode flags] is returned in
// nodemask. It is not permitted to combine MPolFMemsAllowed with
// either MPolFAddr or MPolFNode.
//
// If flags specifies MPolFAddr, then information is returned about the
// policy governing the memory address given in addr. This policy may be
// different from the process's default policy if mbind(2) or one of the
// helper functions described in numa(3) has been used to establish a policy
// for the memory range containing addr.
//
// If flags specifies both MPolFNode and MPolFAddr, get_mempolicy() will
// return the node ID of the node on which the address addr is allocated
// into
// the location pointed to by mode. If no page has yet been allocated for
// the
// specified address, get_mempolicy() will allocate a page as if the process
// had performed a read [load] access to that address, and return the ID of
// the node where that page was allocated.
//
// If flags specifies MPolFNode, but not MPolFAddr, and the process's
// current policy is MPolInterleave, then get_mempolicy() will return in
// the location pointed to by a non-NULL mode argument, the node ID of the
// next node that will be used for interleaving of internal kernel pages
// allocated on behalf of the process. These allocations include pages for
// memory mapped files in process memory ranges mapped using the mmap(2)
// call with the MAP_PRIVATE flag for read accesses, and in memory ranges
// mapped with the MAP_SHARED flag for all accesses.
func GetMemPolicy(
	nodemask Bitmask,
	addr unsafe.Pointer,
	flags int,
) (mode int, err error) {
	var mask, maxnode uintptr
	if maxnode = uintptr(nodemask.Len()); maxnode != 0 {
		mask = uintptr(unsafe.Pointer(&nodemask[0]))
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_GET_MEMPOLICY,
		uintptr(unsafe.Pointer(&mode)), mask, maxnode,
		uintptr(addr), uintptr(flags), 0)
	if errno != 0 {
		err = errno
	}
	return
}

// SetMemPolicy sets the NUMA memory policy of the calling process, which
// consists of a policy mode and zero or more nodes, to the values specified
// by the mode, nodemask and maxnode arguments.
// Details to see manpage of set_mempolicy.
//
// A NUMA machine has different memory controllers with different distances
// to specific CPUs. The memory policy defines from which node memory is
// allocated for the process.

// SetMemPolicy defines the default policy for the process. The process
// policy governs allocation of pages in the process's address space
// outside of memory ranges controlled by a more specific policy set by
// mbind(2). The process default policy also controls allocation of any
// pages for memory mapped files mapped using the mmap(2) call with the
// MAP_PRIVATE flag and that are only read [loaded] from by the process
// and of memory mapped files mapped using the mmap(2) call with the
// MAP_SHARED flag, regardless of the access type. The policy is applied
// only when a new page is allocated for the process. For anonymous memory
// this is when the page is first touched by the application. The mode
// argument must specify one of MPolDefault, MPolBind, MPolInterleave or
// PolPreferred. All modes except MPolDefault require the caller to
// specify via the nodemask argument one or more nodes. The mode argument
// may also include an optional mode flag. The supported mode flags are:
// MPolFStaticNodes and MPolFRelativeNodes. Where a nodemask is required,
// it must contain at least one node that is on-line, allowed by the process
// current cpuset context, [unless the MPolFStaticNodes mode flag is
// specified], and contains memory. If the MPolFStaticNodes is set in mode
// and a required nodemask contains no nodes that are allowed by the
// process current cpuset context, the memory policy reverts to local
// allocation. This effectively overrides the specified policy until the
// process cpuset context includes one or more of the nodes specified
// by nodemask.
func SetMemPolicy(mode int, nodemask Bitmask) (err error) {
	var mask, maxnode uintptr
	if maxnode = uintptr(nodemask.Len()); maxnode != 0 {
		mask = uintptr(unsafe.Pointer(&nodemask[0]))
	}
	_, _, errno := syscall.Syscall(syscall.SYS_SET_MEMPOLICY,
		uintptr(mode), mask, maxnode)
	if errno != 0 {
		err = errno
	}
	return
}

// MBind sets the NUMA memory policy, which consists of a policy mode
// and zero or more nodes, for the memory range starting with addr
// and continuing for length bytes. The memory policy defines from
// which node memory is allocated. Details to see manpage of mbind.
// If the memory range specified by the addr and length arguments
// includes an "anonymous" region of memory that is a region of memory
// created using the mmap(2) system call with the MAP_ANONYMOUS or a
// memory mapped file, mapped using the mmap(2) system call with the
// MAP_PRIVATE flag, pages will be allocated only according to the
// specified policy when the application writes [stores] to the page.
// For anonymous regions, an initial read access will use a shared page
// in the kernel containing all zeros. For a file mapped with MAP_PRIVATE,
// an initial read access will allocate pages according to the process
// policy of the process that causes the page to be allocated. This may
// not be the process that called mbind(). The specified policy will be
// ignored for any MAP_SHARED mappings in the specified memory range.
// Rather the pages will be allocated according to the process policy of
// the process that caused the page to be allocated. Again, this may not
// be the process that called mbind(). If the specified memory range
// includes a shared memory region created using the shmget(2) system call
// and attached using the shmat(2) system call, pages allocated for the
// anonymous or shared memory region will be allocated according to the
// policy specified, regardless which process attached to the shared memory
// segment causes the allocation. If, however, the shared memory region was
// created with the SHM_HUGETLB flag, the huge pages will be allocated
// according to the policy specified only if the page allocation is caused by
// the process that calls mbind() for that region. By default, mbind() has an
// effect only for new allocations; if the pages inside the range have been
// already touched before setting the policy, then the policy has no effect.
// This default behavior may be overridden by the MPolMFMove and MPolMFMoveAll
// flags described below.
func MBind(
	addr unsafe.Pointer,
	length, mode, flags int,
	nodemask Bitmask,
) (err error) {
	var mask, maxnode uintptr
	if maxnode = uintptr(nodemask.Len()); maxnode != 0 {
		mask = uintptr(unsafe.Pointer(&nodemask[0]))
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_MBIND, uintptr(addr),
		uintptr(length), uintptr(mode), mask, maxnode, uintptr(flags))
	if errno != 0 {
		err = errno
	}
	return
}

// GetSchedAffinity writes the affinity mask of the process whose ID is pid
// into the input mask. If pid is zero, then the mask of the calling process
// is returned.
func GetSchedAffinity(pid int, cpumask Bitmask) (int, error) {
	var mask, maxnode uintptr
	if maxnode = uintptr(cpumask.Len() / 8); maxnode != 0 {
		mask = uintptr(unsafe.Pointer(&cpumask[0]))
	}
	length, _, e1 := syscall.Syscall(syscall.SYS_SCHED_GETAFFINITY,
		uintptr(pid), maxnode, mask)
	if e1 != 0 {
		return 0, e1
	}
	return int(length), nil
}

// SetSchedAffinity sets the CPU affinity mask of the process whose ID
// is pid to the value specified by mask. If pid is zero, then the calling
// process is used.
func SetSchedAffinity(pid int, cpumask Bitmask) error {
	var mask, maxnode uintptr
	if maxnode = uintptr(cpumask.Len() / 8); maxnode != 0 {
		mask = uintptr(unsafe.Pointer(&cpumask[0]))
	}
	_, _, e1 := syscall.Syscall(syscall.SYS_SCHED_SETAFFINITY,
		uintptr(pid), maxnode, mask)
	if e1 != 0 {
		return e1
	}
	return nil
}

// We do this the way Paul Jackson's libcpuset does it. The nodemask
// values in /proc/self/status are in an ASCII format that uses nine
// characters for each 32 bits of mask. This could also be used to
// find the cpumask size.
func setupnodemask() (n int) {
	d, err := ioutil.ReadFile("/proc/self/status")
	if err == nil {
		const stp = "Mems_allowed:\t"
		for _, line := range strings.Split(string(d), "\n") {
			if !strings.HasPrefix(line, stp) {
				continue
			}
			n = (len(line) - len(stp) + 1) * 32 / 9
		}
	}
	if n == 0 {
		n = 16
		for n < 4096*8 {
			n <<= 1
			mask := NewBitmask(n)
			if _, err := GetMemPolicy(mask, nil, 0); err != nil &&
				err != syscall.EINVAL {
				break
			}
		}
	}
	return
}

func setupconfigurednodes() (n int) {
	files, err := ioutil.ReadDir("/sys/devices/system/node")
	if err != nil {
		return 1
	}
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), "node") {
			continue
		}
		i, _ := strconv.Atoi(f.Name()[4:])
		if n < i {
			n = i // maybe some node absence
		}
		numanodes.Set(i, true)
		if _, _, err := NodeMemSize64(i); err == nil {
			memnodes.Set(i, true)
		}
	}
	n++
	return
}

func setupncpu() (n int) {
	length := 4096
	for {
		mask := NewBitmask(length)
		nn, err := GetSchedAffinity(0, mask)
		if err == nil {
			return nn * 8
		}
		if err != syscall.EINVAL {
			return 128
		}
		length *= 2
	}
}

func setupnconfiguredcpu() (n int) {
	// sysconf(_SC_NPROCESSORS_CONF)
	files, err := ioutil.ReadDir("/sys/devices/system/cpu")
	if err == nil {
		for _, f := range files {
			if !f.IsDir() || !strings.HasPrefix(f.Name(), "cpu") {
				continue
			}
			if _, err := strconv.Atoi(f.Name()[3:]); err == nil {
				n++
			}
		}
		return
	}
	// fallback
	d, _ := ioutil.ReadFile("/proc/cpuinfo")
	for _, line := range strings.Split(string(d), "\n") {
		if strings.HasPrefix(line, "processor") {
			n++
		}
	}
	if n == 0 {
		n = 1
	}
	return
}

func setupconstraints() {
	node2cpu = make(map[int]Bitmask)
	cpu2node = make(map[int]int)
	for i := 0; i < numanodes.Len(); i++ {
		if !numanodes.Get(i) {
			continue
		}
		fname := fmt.Sprintf("/sys/devices/system/node/node%d/cpumap", i)
		d, err := ioutil.ReadFile(fname)
		if err != nil {
			continue
		}
		cpumask := NewBitmask(CPUCount())
		tokens := strings.Split(strings.TrimSpace(string(d)), ",")
		for j := 0; j < len(tokens); j++ {
			mask, _ := strconv.ParseUint(tokens[len(tokens)-1-j], 16, 64)
			nn := 64
			if runtime.GOARCH == "386" {
				nn = 32
			}
			for k := 0; k < nn; k++ {
				if (mask>>uint64(k))&0x01 != 0 {
					cpumask.Set(k+j*nn, true)
				}
			}
		}
		node2cpu[i] = cpumask
		for j := 0; j < cpumask.Len(); j++ {
			if cpumask.Get(j) {
				cpu2node[j] = i
			}
		}
	}
}

// NodeMemSize64 return the memory total size and free size of given node.
func NodeMemSize64(node int) (total, free int64, err error) {
	var (
		d     []byte
		fname = fmt.Sprintf("/sys/devices/system/node/node%d/meminfo", node)
	)
	d, err = ioutil.ReadFile(fname)
	if err != nil {
		return
	}
	split := func(s, d string) string {
		return strings.TrimFunc(
			s[strings.Index(s, d)+len(d):], func(x rune) bool {
				return x < '0' || x > '9'
			})
	}
	for _, line := range strings.Split(string(d), "\n") {
		if !strings.HasSuffix(line, "kB") {
			continue
		}
		switch {
		case strings.Contains(line, "MemTotal"):
			total, err = strconv.ParseInt(split(line, "MemTotal"), 10, 64)
			if err != nil {
				return
			}
			total *= 1024
		case strings.Contains(line, "MemFree"):
			free, err = strconv.ParseInt(split(line, "MemFree:"), 10, 64)
			if err != nil {
				return
			}
			free *= 1024
		}
	}
	return
}

// NUMAfastway ...
var NUMAfastway = cpuid.HasFeature(cpuid.RDTSCP)

func getcpu()

// GetCPUAndNode returns the node and cpu which current caller is running on.
func GetCPUAndNode() (cpu, node int)
