package numa

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitmask(t *testing.T) {
	assert := require.New(t)
	for i := 0; i < 256; i++ {
		mask := NewBitmask(i)

		assert.True(mask.Len()%8 == 0)
		assert.True(len(mask)*64 >= i)

		for j := 0; j < i; j++ {
			mask.Set(j, true)
			assert.True(mask.Get(j))
			assert.Equal(j+1, mask.OnesCount())
			mask.Set(j, false)
			assert.False(mask.Get(j))
			mask.Set(j, true)

			for k := j + 1; k < i; k++ {
				assert.False(mask.Get(k))
			}
		}

		clone := mask.Clone()
		for j := 0; j < mask.Len(); j++ {
			assert.Equal(mask.Get(j), clone.Get(j))
		}

		if i != 0 {
			assert.Equal(len(mask), strings.Count(mask.String(), ",")+1)
			assert.Equal(mask.OnesCount(), strings.Count(mask.Text(), ",")+1)
		} else {
			assert.Empty(mask.String())
			assert.Empty(mask.Text())
		}

		mask.ClearAll()
		assert.Equal(0, mask.OnesCount())
		mask.SetAll()
		for j := 0; j < i; j++ {
			assert.True(mask.Get(j))
		}
	}
}
