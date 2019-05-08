package numa

import "testing"

func TestELFHash(t *testing.T) {
	var tt = []struct {
		s string
		h uint32
		g uint32
	}{
		{s: "__vdso_gettimeofday", h: 0x315ca59, g: 0xb01bca00},
		{s: "__vdso_clock_gettime", h: 0xd35ec75, g: 0x6e43a318},
		{s: "__vdso_getcpu", h: 0xb01045, g: 0x6562b026},
	}
	for _, v := range tt {
		h := elfHash(v.s)
		if h != v.h {
			t.Errorf("%s got 0x%x", v.s, h)
		}
		g := elfGNUHash(v.s)
		if g != v.g {
			t.Errorf("%s got 0x%x", v.s, g)
		}
	}
}

func TestVdsoSym(t *testing.T) {
	var tt = []struct {
		s string
		v bool
	}{
		{"__vdso_gettimeofday", true},
		{"__vdso_clock_gettime", true},
		{"__vdso_time", true},
		{"__vdso_getcpu", true},
		{"__abc", false},
	}

	for _, v := range tt {
		p := vdsoSym(v.s)
		if x := p != 0; x != v.v {
			t.Errorf("vdsoSym %v got 0x%x", v.s, p)
		}
	}
}
