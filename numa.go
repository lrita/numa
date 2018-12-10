package numa

import (
	"fmt"
)

var (
	available bool
	// The max possible node count, which represents the node count of local
	// platform supporting.
	// nnodemax =@nodemask_sz+1
	nnodemax int
	// The max configured(enabled/setuped) node, which represents the
	// available node count of local platform.
	// nconfigurednode =@maxconfigurednode+1
	nconfigurednode int
	// The max possible cpu count, which represents the cpu count of local
	// platform supporting.
	// ncpumax =@cpumask_sz+1
	ncpumax int
	// nconfiguredcpu =@maxconfiguredcpu
	nconfiguredcpu int

	memnodes  Bitmask
	numanodes Bitmask

	cpu2node map[int]int
	node2cpu map[int]Bitmask
)

const (
	// The memory policy of GetMemPolicy/SetMemPolicy.
	MPOL_DEFAULT = iota
	MPOL_PREFERRED
	MPOL_BIND
	MPOL_INTERLEAVE
	MPOL_LOCAL
	MPOL_MAX

	// MPOL_F_STATIC_NODES since Linux 2.6.26
	// A nonempty nodemask specifies physical node ids. Linux does will not
	// remap the nodemask when the process moves to a different cpuset context,
	// nor when the set of nodes allowed by the process's current cpuset
	// context changes.
	MPOL_F_STATIC_NODES = 1 << 15

	// MPOL_F_RELATIVE_NODES since Linux 2.6.26
	// A nonempty nodemask specifies node ids that are relative to the set
	// of node ids allowed by the process's current cpuset.
	MPOL_F_RELATIVE_NODES = 1 << 14

	// MPOL_MODE_FLAGS is the union of all possible optional mode flags passed
	// to either SetMemPolicy() or mbind().
	MPOL_MODE_FLAGS = MPOL_F_STATIC_NODES | MPOL_F_RELATIVE_NODES
)

const (
	// Flags for get_mem_policy
	// return next IL node or node of address
	// Warning: MPOL_F_NODE is unsupported and subject to change. Don't use.
	MPOL_F_NODE = 1 << iota
	// look up vma using address
	MPOL_F_ADDR
	// query nodes allowed in cpuset
	MPOL_F_MEMS_ALLOWED
)

const (
	// Flags for mbind
	// Verify existing pages in the mapping
	MPOL_MF_STRICT = 1 << iota
	// Move pages owned by this process to conform to mapping
	MPOL_MF_MOVE
	// Move every page to conform to mapping
	MPOL_MF_MOVE_ALL
	// Modifies '_MOVE: lazy migrate on fault
	MPOL_MF_LAZY
	// Internal flags start here
	POL_MF_INTERNAL

	MPOL_MF_VALID = MPOL_MF_STRICT | MPOL_MF_MOVE | MPOL_MF_MOVE_ALL
)

// Available returns current platform is whether support NUMA.
// @ int numa_available(void)
func Available() bool {
	return available
}

// MaxNodeID returns the max id of current configured NUMA nodes.
// @numa_max_node_int
func MaxNodeID() int {
	return nconfigurednode - 1
}

// MaxPossibleNodeID returns the max possible node id of this platform supported.
// The possible node id always larger than max node id.
func MaxPossibleNodeID() int {
	return nnodemax - 1
}

// NodeCount returns the count of current configured NUMA nodes.
//
// NOTE: this function's behavior matches the documentation (ie: it
// returns a count of nodes with memory) despite the poor function
// naming.  We also cannot use the similarly poorly named
// numa_all_nodes_ptr as it only tracks nodes with memory from which
// the calling process can allocate.  Think sparse nodes, memory-less
// nodes, cpusets...
// @numa_num_configured_nodes
func NodeCount() int {
	return memnodes.OnesCount()
}

// NodeMask returns the mask of current configured nodes.
func NodeMask() Bitmask {
	return memnodes.Clone()
}

// NodePossibleCount returns the possible NUMA nodes count of current platform
// supported.
func NodePossibleCount() int {
	return nnodemax
}

// CPUPossibleCount returns the possible cpu count of current platform supported.
func CPUPossibleCount() int {
	return ncpumax
}

// CPUCount returns the current configured(enabled/detected) cpu count, which
// is different with runtime.NumCPU().
func CPUCount() int {
	return nconfiguredcpu
}

// RunningNodesMask return the bitmask of current process using NUMA nodes.
// @numa_get_run_node_mask_v2
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
	return cpumask, nil
}

// NodeToCPUMask returns the cpumask of given node id.
// @numa_node_to_cpus_v2
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
// The special node -1 will set current process on all available nodes.
// @numa_run_on_node
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
		return fmt.Errorf("invalided node %d", node)
	}
	return SetSchedAffinity(0, cpumask)
}

// GetMemAllowedNodeMask returns the bitmask of current process allowed running
// nodes.
// @numa_get_mems_allowed
func GetMemAllowedNodeMask() (Bitmask, error) {
	mask := NewBitmask(NodePossibleCount())
	if _, err := GetMemPolicy(mask, nil, MPOL_F_MEMS_ALLOWED); err != nil {
		return nil, err
	}
	return mask, nil
}

// RunOnNodeMask run current process to the given nodes.
// @numa_run_on_node_mask_v2
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
// @numa_bind_v2
func Bind(mask Bitmask) error {
	if err := RunOnNodeMask(mask); err != nil {
		return err
	}
	return SetMemPolicy(MPOL_BIND, mask)
}
