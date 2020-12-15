package gonuma

import (
	"fmt"
)

var (
	available bool

	// NUMAnodemax is the maximum possible node count. It represents
	// the absolute highest node count supported on the local platform.
	// NUMAnodemax =@nodemask_sz+1
	NUMAnodemax int

	// NUMAconfigurednode represents the maximum possible number of
	// configured or enabled nodes supported on the local platform.
	// NUMAconfigurednode =@maxconfigurednode+1
	NUMAconfigurednode int

	// NUMAcpuMax is the maximum possible CPU count, which represents
	// the absolute highest CPU count supported on the local platform.
	// NUMAcpuMax =@cpumask_sz+1
	NUMAcpuMax int

	// NUMAconfiguredcpu is the number of currently configured CPUs.
	// NUMAconfiguredcpu =@maxconfiguredcpu
	NUMAconfiguredcpu int

	memnodes  Bitmask
	numanodes Bitmask

	cpu2node map[int]int
	node2cpu map[int]Bitmask
)

// const block
const (
	MPolDefault = iota
	MPolPreferred
	MPolBind
	MPolInterleave
	MPolLocal
	MPolMax

	// MPolFStaticNodes since Linux 2.6.26 ....
	// A nonempty nodemask specifies physical node ids. Linux does will
	// not remap the nodemask when the process moves to a different cpuset
	// context, nor when the set of nodes allowed by the process current
	// cpuset context changes.
	MPolFStaticNodes = 1 << 15

	// MPolFRelativeNodes since Linux 2.6.26
	// A nonempty nodemask specifies node ids that are relative to the set
	// of node ids allowed by the process's current cpuset.
	MPolFRelativeNodes = 1 << 14

	// MPolModeFlags is the union of all possible optional mode flags passed
	// to either SetMemPolicy() or mbind().
	MPolModeFlags = MPolFStaticNodes | MPolFRelativeNodes
)

const (
	// MPolFNode is unsupported and subject to change.
	// Flags for get_mem_policy return next IL node or node of address
	MPolFNode = 1 << iota
	// MPolFAddr looks up vma using address
	MPolFAddr
	// MPolFMemsAllowed queries nodes allowed in cpuset
	MPolFMemsAllowed
)

const (
	// MPolMFStrict verifies existing pages in the mapping Flags for mbind
	MPolMFStrict = 1 << iota
	// MPolMFMove moves pages owned by this process to conform to mapping
	MPolMFMove
	// MPolMFMoveAll moves every page to conform to mapping
	MPolMFMoveAll
	// MpolMfLazy modifies '_MOVE: lazy migrate on fault
	MpolMfLazy
	// PolMfInternal is for internal flags starting here
	PolMfInternal
	// MPolMFValid = ...
	MPolMFValid = MPolMFStrict | MPolMFMove | MPolMFMoveAll
)

// NUMAavailable returns current platform is whether support NUMA.
func NUMAavailable() bool {
	return available
}

// MaxNodeID returns the max id of current configured NUMA nodes.
func MaxNodeID() int {
	return NUMAconfigurednode - 1
}

// MaxPossibleNodeID returns the max possible node id this platform supports
// The possible node id always larger than max node id.
func MaxPossibleNodeID() int {
	return NUMAnodemax - 1
}

// NodeCount returns the count of current configured NUMA nodes.
// NOTE: this function's behavior matches the documentation (ie: it
// returns a count of nodes with memory) despite the poor function
// naming.  We also cannot use the similarly poorly named
// numa_all_nodes_ptr as it only tracks nodes with memory from which
// the calling process can allocate.  Think sparse nodes, memory-less
// nodes, cpusets.
func NodeCount() int {
	return memnodes.OnesCount()
}

// NodeMask returns the mask of current configured nodes.
func NodeMask() Bitmask {
	return memnodes.Clone()
}

// NodePossibleCount returns the possible NUMA nodes count of current
// platform supported.
func NodePossibleCount() int {
	return NUMAnodemax
}

// CPUPossibleCount returns the possible cpu count current platform supports
func CPUPossibleCount() int {
	return NUMAcpuMax
}

// CPUCount returns the current configured(enabled/detected) cpu count,
// which is different with runtime.NumCPU().
func CPUCount() int {
	return NUMAconfiguredcpu
}

// RunningNodesMask return the bitmask of current process using NUMA nodes.
func RunningNodesMask() (Bitmask, error) {
	nodemask := NewBitmask(NodePossibleCount())
	cpumask := NewBitmask(CPUPossibleCount())
	if _, err := GetSchedAffinity(0, cpumask); err != nil {
		return nil, err
	}
	for i := 0; i < cpumask.Len(); i++ {
		if !cpumask.Get(i) {
			continue
		}
		n, err := CPUToNode(i)
		if err != nil {
			return nil, err
		}
		nodemask.Set(n, true)
	}
	return nodemask, nil
}

// RunningCPUMask return the cpu bitmask of current process running on.
func RunningCPUMask() (Bitmask, error) {
	cpumask := NewBitmask(CPUPossibleCount())
	if _, err := GetSchedAffinity(0, cpumask); err != nil {
		return nil, err
	}
	return cpumask[:len(NewBitmask(CPUCount()))], nil
}

// NodeToCPUMask returns the cpumask of given node id.
func NodeToCPUMask(node int) (Bitmask, error) {
	if node > MaxPossibleNodeID() {
		return nil, fmt.Errorf("node %d is out of range", node)
	}
	cpumask, ok := node2cpu[node]
	if !ok {
		return nil, fmt.Errorf("node %d not found", node)
	}
	return cpumask.Clone(), nil
}

// CPUToNode returns the node id by given cpu id.
func CPUToNode(cpu int) (int, error) {
	node, ok := cpu2node[cpu]
	if !ok {
		return 0, fmt.Errorf("cpu %d not found", cpu)
	}
	return node, nil
}

// RunOnNode set current process run on given node.
// The special node "-1" will set current process on all available nodes.
func RunOnNode(node int) (err error) {
	var cpumask Bitmask
	switch {
	case node == -1:
		cpumask = NewBitmask(CPUPossibleCount())
		cpumask.SetAll()
	case node >= 0:
		cpumask, err = NodeToCPUMask(node)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid node %d", node)
	}
	return SetSchedAffinity(0, cpumask)
}

// GetMemAllowedNodeMask returns the bitmask of current process allowed
// running nodes.
func GetMemAllowedNodeMask() (Bitmask, error) {
	mask := NewBitmask(NodePossibleCount())
	if _, err := GetMemPolicy(mask, nil, MPolFMemsAllowed); err != nil {
		return nil, err
	}
	return mask, nil
}

// RunOnNodeMask run current process to the given nodes.
func RunOnNodeMask(mask Bitmask) error {
	cpumask := NewBitmask(CPUPossibleCount())
	m := mask.Clone()
	for i := 0; i < mask.Len(); i++ {
		if !m.Get(i) {
			continue
		}
		if !memnodes.Get(i) {
			continue
		}
		cpu, err := NodeToCPUMask(i)
		if err != nil {
			return err
		}
		for j := 0; j < cpu.Len(); j++ {
			cpumask.Set(j, true)
		}
	}
	return SetSchedAffinity(0, cpumask)
}

// Bind bind current process on those nodes which given by a bitmask.
func Bind(mask Bitmask) error {
	if err := RunOnNodeMask(mask); err != nil {
		return err
	}
	return SetMemPolicy(MPolBind, mask)
}
