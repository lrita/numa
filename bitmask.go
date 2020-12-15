package gonuma

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// Bitmask is used for syscall or other operation in NUMA API.
type Bitmask []uint64

// Set sets the No.i bit to v.
func (b Bitmask) Set(i int, v bool) {
	if n := i / 64; n < len(b) {
		if v {
			b[n] |= uint64(1) << uint64(i%64)
		} else {
			b[n] &= ^(uint64(1) << uint64(i%64))
		}
	}
}

// Get returns the No.i bit of this bitmask.
func (b Bitmask) Get(i int) bool {
	if n := i / 64; n < len(b) {
		return (b[n]>>uint64(i%64))&0x1 != 0
	}
	return false
}

// ClearAll clear all bits of this bitmask.
func (b Bitmask) ClearAll() {
	for i := range b {
		b[i] = 0
	}
}

// SetAll clear all bits of this bitmask.
func (b Bitmask) SetAll() {
	for i := range b {
		b[i] = ^uint64(0)
	}
}

// OnesCount returns the number of one bits ("population count") in this
// bitmask.
func (b Bitmask) OnesCount() (n int) {
	for _, v := range b {
		n += bits.OnesCount64(v)
	}
	return
}

// String returns the hex string of this bitmask.
func (b Bitmask) String() string {
	s := make([]string, 0, len(b))
	for _, v := range b {
		s = append(s, fmt.Sprintf("%016X", v))
	}
	return strings.Join(s, ",")
}

// Text returns the digital bit text of this bitmask.
func (b Bitmask) Text() string {
	var (
		s   []string
		length = b.Len() * 8
	)
	for i := 0; i < length; i++ {
		if b.Get(i) {
			s = append(s, strconv.Itoa(i))
		}
	}
	return strings.Join(s, ",")
}

// Len returns the bitmask length.
func (b Bitmask) Len() int { return len(b) * 64 }

// Clone return a duplicated bitmask.
func (b Bitmask) Clone() Bitmask {
	bb := make(Bitmask, len(b))
	copy(bb, b)
	return bb
}

// NewBitmask returns a bitmask, which length always rounded to a multiple
// of
// sizeof(uint64). The input param n represents the bit count of this
// bitmask.
func NewBitmask(n int) Bitmask {
	return make(Bitmask, (n+63)/64)
}
