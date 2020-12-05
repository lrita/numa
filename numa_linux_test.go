// +build linux

package gonuma_test

import (
	"runtime"
	"sync"
	"testing"

	gonuma "github.com/johnsonjh/gonuma"
	"github.com/stretchr/testify/require"
)

func TestGetNodeAndCPU2(t *testing.T) {
	if !gonuma.NUMAavailable() || !gonuma.NUMAfastway {
		t.Skip("not available or fastwway")
	}

	var (
		nodem  = gonuma.NewBitmask(gonuma.NodePossibleCount())
		mu     sync.Mutex
		wg     sync.WaitGroup
		assert = require.New(t)
	)

	gonuma.NUMAfastway = false
	defer func() { gonuma.NUMAfastway = true }()

	cpum := make([]gonuma.Bitmask, gonuma.NodePossibleCount())
	for i := 0; i < len(cpum); i++ {
		cpum[i] = gonuma.NewBitmask(gonuma.CPUPossibleCount())
	}
	for i := 0; i < gonuma.CPUCount(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				cpu, node := gonuma.GetCPUAndNode()
				mu.Lock()
				cpum[node].Set(cpu, true)
				nodem.Set(node, true)
				mu.Unlock()
				runtime.Gosched()
			}
		}()
	}
	wg.Wait()

	for i := 0; i < (2 * gonuma.CPUCount()); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cpu, node := gonuma.GetCPUAndNode()
			t.Logf("TestGetNodeAndCPU2 cpu %v node %v", cpu, node)
		}()
	}
	wg.Wait()

	nmask := gonuma.NodeMask()
	for i := 0; i < nodem.Len(); i++ {
		if !nodem.Get(i) {
			continue
		}
		assert.True(nmask.Get(i), "node %d", i)
		cpumask, err := gonuma.NodeToCPUMask(i)
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
