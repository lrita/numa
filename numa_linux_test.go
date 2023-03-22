package numa

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCPUAndNodeShow(t *testing.T) {
	var (
		assert = require.New(t)
		slice  = map[int][]string{}
		nmask  = NodeMask()
	)

	for i := 0; i < nmask.Len(); i++ {
		if !nmask.Get(i) {
			continue
		}
		var cpu []string
		m, err := NodeToCPUMask(i)
		assert.NoError(err, "node %d", i)
		for j := 0; j < m.Len(); j++ {
			if m.Get(j) {
				cpu = append(cpu, fmt.Sprint(j))
			}
		}
		slice[i] = cpu
	}

	for node, cpu := range slice {
		t.Log(fmt.Sprintf("node %d cpus: %s", node, strings.Join(cpu, " ")))
	}
}
