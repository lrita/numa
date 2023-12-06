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

func TestCPUMaskParse(t *testing.T) {
	cpumask1 := parseCPUMapMask("000f,ff000fff", 32, 64)
	if cpumask1.String() != "0000000FFF000FFF" {
		t.Fatalf("Failed to parse CPU Mask 1 correctly, Expected %v but got %v", "0000000FFF000FFF", cpumask1.String())
	}

	cpumask2 := parseCPUMapMask("fff0,00fff000", 32, 64)
	if cpumask2.String() != "0000FFF000FFF000" {
		t.Fatalf("Failed to parse CPU Mask 2 correctly")
	}

	cpumask3 := parseCPUMapMask("00000000,00000000,00000000,00000000,00000000,ffffffff,00000000,ffffffff", 32, 128)
	if cpumask3.String() != "00000000FFFFFFFF,00000000FFFFFFFF" {
		t.Fatalf("Failed to parse CPU Mask 3 correctly, Expected %v but got %v", "00000000FFFFFFFF,00000000FFFFFFFF", cpumask3.String())
	}
}
