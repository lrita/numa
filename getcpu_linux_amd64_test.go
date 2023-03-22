package numa

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetNodeAndCPU2(t *testing.T) {
	if !Available() || !fastway {
		t.Skip("not available or fastwway")
	}

	var (
		nodem  = NewBitmask(NodePossibleCount())
		mu     sync.Mutex
		wg     sync.WaitGroup
		assert = require.New(t)
	)

	fastway = false
	defer func() { fastway = true }()

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
