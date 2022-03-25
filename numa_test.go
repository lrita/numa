package numa

import (
	"runtime"
	"sync"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestNotAvailable(t *testing.T) {
	if Available() {
		t.Skip("TestNotAvailable")
	}
	assert := require.New(t)
	_, err := GetMemPolicy(nil, nil, 0)
	assert.Equal(syscall.ENOSYS, err)
	assert.Equal(syscall.ENOSYS, SetMemPolicy(MPOL_DEFAULT, nil))

	assert.Equal(syscall.ENOSYS, Bind(nil))
	assert.Equal(syscall.ENOSYS, MBind(nil, 0, 0, 0, nil))

	_, err = GetSchedAffinity(0, nil)
	assert.Equal(syscall.ENOSYS, err)
	assert.Equal(syscall.ENOSYS, SetSchedAffinity(0, nil))

	assert.Equal(syscall.ENOSYS, RunOnNode(-1))
	assert.Equal(syscall.ENOSYS, RunOnNode(0))
	assert.Error(RunOnNode(NodePossibleCount() + 1))
	assert.Error(RunOnNode(-2))

	for i := 0; i < CPUCount()+10; i++ {
		node, err := CPUToNode(i)
		if i < CPUCount() {
			assert.NoError(err)
			assert.Equal(0, node)
		} else {
			assert.Error(err)
		}
	}

	_, err = NodeToCPUMask(NodePossibleCount() + 1)
	assert.Error(err)
	_, err = NodeToCPUMask(NodePossibleCount() + 1)
	assert.Error(err)

	_, err = RunningNodesMask()
	assert.Error(err)

	_, err = RunningCPUMask()
	assert.Error(err)

	assert.Equal(syscall.ENOSYS, RunOnNodeMask(NodeMask()))
}

func TestNodeMemSize64(t *testing.T) {
	var (
		assert   = require.New(t)
		nodemask = NodeMask()
	)
	if !Available() {
		for i := 0; i < nodemask.Len(); i++ {
			_, _, err := NodeMemSize64(i)
			assert.Equal(syscall.ENOSYS, err)
		}
	} else {
		for i := 0; i < nodemask.Len(); i++ {
			if !nodemask.Get(i) {
				continue
			}
			total, freed, err := NodeMemSize64(i)
			assert.NoError(err)
			assert.True(total > 0)
			assert.True(freed >= 0)
		}
	}
}

func TestNUMAAPI(t *testing.T) {
	assert := require.New(t)
	assert.True(MaxNodeID() >= 0, "MaxNodeID() >= 0")
	assert.True(MaxPossibleNodeID() >= 0, "MaxPossibleNodeID() >= 0")
	assert.True(MaxPossibleNodeID() >= MaxNodeID())
	assert.True(NodeCount() > 0, "NodeCount() > 0")
	assert.True(NodePossibleCount() > 0, "NodePossibleCount() > 0")
	assert.True(NodePossibleCount() >= NodeCount())
	assert.True(CPUCount() > 0)
}

func TestMemPolicy(t *testing.T) {
	if !Available() {
		t.Skip()
	}
	assert := require.New(t)

	t.Log("nnodemask = ", nnodemax)
	t.Log("nconfigurednode =", nconfigurednode)
	t.Log("ncpumask =", ncpumax)
	t.Log("nconfiguredcpu =", nconfiguredcpu)

	mode, err := GetMemPolicy(nil, nil, 0)
	assert.NoError(err)
	assert.True(mode >= 0 && mode < MPOL_MAX, "%#v", mode)
	assert.NoError(SetMemPolicy(MPOL_DEFAULT, nil))
}

func TestGetMemAllowedNodeMaskAndBind(t *testing.T) {
	assert := require.New(t)
	mask, err := GetMemAllowedNodeMask()
	if Available() {
		assert.NoError(err)
		assert.True(mask.OnesCount() > 0)
		assert.NoError(Bind(mask))
	} else {
		assert.Equal(syscall.ENOSYS, err)
	}
}

func TestRunOnNodeAndRunningNodesMask(t *testing.T) {
	if !Available() {
		t.Skip()
	}
	assert := require.New(t)
	mask, err := RunningNodesMask()
	assert.NoError(err)
	assert.True(mask.OnesCount() > 0)
	for i := 0; i < mask.Len(); i++ {
		if !mask.Get(i) {
			continue
		}
		assert.NoError(RunOnNode(i), "run on node %d", i)

		cpumask, err := NodeToCPUMask(i)
		assert.NoError(err)
		assert.True(cpumask.OnesCount() > 0)

		gotmask, err := RunningCPUMask()
		assert.NoError(err)
		assert.Equal(cpumask, gotmask)

		for j := 0; j < cpumask.Len(); j++ {
			if !cpumask.Get(j) {
				continue
			}
			node, err := CPUToNode(j)
			assert.NoError(err)
			assert.Equal(i, node)
		}
	}

	assert.NoError(RunOnNode(-1))
	assert.Error(RunOnNode(-2))
	assert.Error(RunOnNode(1 << 20))

	_, err = CPUToNode(CPUPossibleCount())
	assert.Error(err)
}

func TestMBind(t *testing.T) {
	if !Available() {
		t.Skip()
	}
	assert := require.New(t)

	assert.Equal(syscall.EINVAL,
		MBind(unsafe.Pointer(t), 100, MPOL_DEFAULT, 0, nil))
}

func TestGetNodeAndCPU(t *testing.T) {
	if !Available() {
		t.Skip("not available")
	}
	var (
		nodem  = NewBitmask(NodePossibleCount())
		mu     sync.Mutex
		wg     sync.WaitGroup
		assert = require.New(t)
	)
	cpum := make([]Bitmask, NodePossibleCount())
	for i := 0; i < len(cpum); i++ {
		cpum[i] = NewBitmask(CPUPossibleCount())
	}
	for i := 0; i < CPUCount(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				cpu, node := GetCPUAndNode()
				mu.Lock()
				cpum[node].Set(cpu, true)
				nodem.Set(node, true)
				mu.Unlock()
				runtime.Gosched()
			}
		}()
	}
	wg.Wait()

	nmask := NodeMask()
	for i := 0; i < nodem.Len(); i++ {
		if !nodem.Get(i) {
			continue
		}
		assert.True(nmask.Get(i), "node %d", i)
		cpumask, err := NodeToCPUMask(i)
		assert.NoError(err)
		cmask := cpum[i]
		for j := 0; j < cmask.Len(); j++ {
			if !cmask.Get(j) {
				continue
			}
			assert.True(cpumask.Get(j), "cpu %d @ node %d", j, i)
		}
	}
}

func BenchmarkGetCPUAndNode(b *testing.B) {
	b.RunParallel(func(bp *testing.PB) {
		for bp.Next() {
			GetCPUAndNode()
		}
	})
}
