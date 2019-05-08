package numa

import (
	"io/ioutil"
	"unsafe"
)

const (
	_AT_NULL         = 0 // End of vector
	_AT_SYSINFO_EHDR = 33

	_PT_LOAD    = 1 /* Loadable program segment */
	_PT_DYNAMIC = 2 /* Dynamic linking information */

	_DT_NULL     = 0          /* Marks end of dynamic section */
	_DT_HASH     = 4          /* Dynamic symbol hash table */
	_DT_STRTAB   = 5          /* Address of string table */
	_DT_SYMTAB   = 6          /* Address of symbol table */
	_DT_GNU_HASH = 0x6ffffef5 /* GNU-style dynamic symbol hash table */
	_DT_VERSYM   = 0x6ffffff0
	_DT_VERDEF   = 0x6ffffffc

	_VER_FLG_BASE = 0x1 /* Version definition of file itself */

	_SHN_UNDEF = 0 /* Undefined section */

	_SHT_DYNSYM = 11 /* Dynamic linker symbol table */

	_STT_FUNC = 2 /* Symbol is a code object */

	_STT_NOTYPE = 0 /* Symbol type is not specified */

	_STB_GLOBAL = 1 /* Global symbol */
	_STB_WEAK   = 2 /* Weak symbol */

	_EI_NIDENT = 16

	// vdsoArrayMax is the byte-size of a maximally sized array on this architecture.
	// See cmd/compile/internal/amd64/galign.go arch.MAXWIDTH initialization.
	vdsoArrayMax = 1<<50 - 1

	// Maximum indices for the array types used when traversing the vDSO ELF structures.
	// Computed from architecture-specific max provided by vdso_linux_*.go
	vdsoSymTabSize     = vdsoArrayMax / unsafe.Sizeof(elfSym{})
	vdsoDynSize        = vdsoArrayMax / unsafe.Sizeof(elfDyn{})
	vdsoSymStringsSize = vdsoArrayMax     // byte
	vdsoVerSymSize     = vdsoArrayMax / 2 // uint16
	vdsoHashSize       = vdsoArrayMax / 4 // uint32

	// vdsoBloomSizeScale is a scaling factor for gnuhash tables which are uint32 indexed,
	// but contain uintptrs
	vdsoBloomSizeScale = unsafe.Sizeof(uintptr(0)) / 4 // uint32
)

type vdsoVersionKey struct {
	version string
	verHash uint32
}

// ELF64 structure definitions for use by the vDSO loader

type elfSym struct {
	st_name  uint32
	st_info  byte
	st_other byte
	st_shndx uint16
	st_value uint64
	st_size  uint64
}

type elfVerdef struct {
	vd_version uint16 /* Version revision */
	vd_flags   uint16 /* Version information */
	vd_ndx     uint16 /* Version Index */
	vd_cnt     uint16 /* Number of associated aux entries */
	vd_hash    uint32 /* Version name hash value */
	vd_aux     uint32 /* Offset in bytes to verdaux array */
	vd_next    uint32 /* Offset in bytes to next verdef entry */
}

type elfEhdr struct {
	e_ident     [_EI_NIDENT]byte /* Magic number and other info */
	e_type      uint16           /* Object file type */
	e_machine   uint16           /* Architecture */
	e_version   uint32           /* Object file version */
	e_entry     uint64           /* Entry point virtual address */
	e_phoff     uint64           /* Program header table file offset */
	e_shoff     uint64           /* Section header table file offset */
	e_flags     uint32           /* Processor-specific flags */
	e_ehsize    uint16           /* ELF header size in bytes */
	e_phentsize uint16           /* Program header table entry size */
	e_phnum     uint16           /* Program header table entry count */
	e_shentsize uint16           /* Section header table entry size */
	e_shnum     uint16           /* Section header table entry count */
	e_shstrndx  uint16           /* Section header string table index */
}

type elfPhdr struct {
	p_type   uint32 /* Segment type */
	p_flags  uint32 /* Segment flags */
	p_offset uint64 /* Segment file offset */
	p_vaddr  uint64 /* Segment virtual address */
	p_paddr  uint64 /* Segment physical address */
	p_filesz uint64 /* Segment size in file */
	p_memsz  uint64 /* Segment size in memory */
	p_align  uint64 /* Segment alignment */
}

type elfShdr struct {
	sh_name      uint32 /* Section name (string tbl index) */
	sh_type      uint32 /* Section type */
	sh_flags     uint64 /* Section flags */
	sh_addr      uint64 /* Section virtual addr at execution */
	sh_offset    uint64 /* Section file offset */
	sh_size      uint64 /* Section size in bytes */
	sh_link      uint32 /* Link to another section */
	sh_info      uint32 /* Additional section information */
	sh_addralign uint64 /* Section alignment */
	sh_entsize   uint64 /* Entry size if section holds table */
}

type elfDyn struct {
	d_tag int64  /* Dynamic entry type */
	d_val uint64 /* Integer value */
}

type elfVerdaux struct {
	vda_name uint32 /* Version or dependency names */
	vda_next uint32 /* Offset in bytes to next verdaux entry */
}

type vdsoInfo struct {
	valid bool

	/* Load information */
	loadAddr   unsafe.Pointer
	loadOffset unsafe.Pointer /* loadAddr - recorded vaddr */

	/* Symbol table */
	symtab     *[vdsoSymTabSize]elfSym
	symstrings *[vdsoSymStringsSize]byte
	chain      []uint32
	bucket     []uint32
	symOff     uint32
	isGNUHash  bool

	/* Version table */
	versym *[vdsoVerSymSize]uint16
	verdef *elfVerdef
}

var (
	vdsoLinuxVersion = vdsoVersionKey{"LINUX_2.6", 0x3ae75f6}
	vdsoinfo         vdsoInfo
	vdsoVersion      int32

	vdsoGetCPU uintptr
)

/* How to extract and insert information held in the st_info field.  */
func _ELF_ST_BIND(val byte) byte { return val >> 4 }
func _ELF_ST_TYPE(val byte) byte { return val & 0xf }

//go:nosplit
func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

//go:linkname gostringnocopy runtime.gostringnocopy
//go:nosplit
func gostringnocopy(str *byte) string

func vdsoInitFromSysinfoEhdr(info *vdsoInfo, hdr *elfEhdr) {
	info.valid = false
	info.loadAddr = unsafe.Pointer(hdr)

	pt := unsafe.Pointer(uintptr(info.loadAddr) + uintptr(hdr.e_phoff))

	// We need two things from the segment table: the load offset
	// and the dynamic table.
	var foundVaddr bool
	var dyn *[vdsoDynSize]elfDyn
	for i := uint16(0); i < hdr.e_phnum; i++ {
		pt := (*elfPhdr)(add(pt, uintptr(i)*unsafe.Sizeof(elfPhdr{})))
		switch pt.p_type {
		case _PT_LOAD:
			if !foundVaddr {
				foundVaddr = true
				info.loadOffset = unsafe.Pointer(uintptr(info.loadAddr) + uintptr(pt.p_offset-pt.p_vaddr))
			}

		case _PT_DYNAMIC:
			dyn = (*[vdsoDynSize]elfDyn)(unsafe.Pointer(uintptr(info.loadAddr) + uintptr(pt.p_offset)))
		}
	}

	if !foundVaddr || dyn == nil {
		return // Failed
	}

	// Fish out the useful bits of the dynamic table.

	var hash, gnuhash *[vdsoHashSize]uint32
	info.symstrings = nil
	info.symtab = nil
	info.versym = nil
	info.verdef = nil
	for i := 0; dyn[i].d_tag != _DT_NULL; i++ {
		dt := &dyn[i]
		p := unsafe.Pointer(uintptr(info.loadOffset) + uintptr(dt.d_val))
		switch dt.d_tag {
		case _DT_STRTAB:
			info.symstrings = (*[vdsoSymStringsSize]byte)(unsafe.Pointer(p))
		case _DT_SYMTAB:
			info.symtab = (*[vdsoSymTabSize]elfSym)(unsafe.Pointer(p))
		case _DT_HASH:
			hash = (*[vdsoHashSize]uint32)(unsafe.Pointer(p))
		case _DT_GNU_HASH:
			gnuhash = (*[vdsoHashSize]uint32)(unsafe.Pointer(p))
		case _DT_VERSYM:
			info.versym = (*[vdsoVerSymSize]uint16)(unsafe.Pointer(p))
		case _DT_VERDEF:
			info.verdef = (*elfVerdef)(unsafe.Pointer(p))
		}
	}

	if info.symstrings == nil || info.symtab == nil || (hash == nil && gnuhash == nil) {
		return // Failed
	}

	if info.verdef == nil {
		info.versym = nil
	}

	if gnuhash != nil {
		// Parse the GNU hash table header.
		nbucket := gnuhash[0]
		info.symOff = gnuhash[1]
		bloomSize := gnuhash[2]
		info.bucket = gnuhash[4+bloomSize*uint32(vdsoBloomSizeScale):][:nbucket]
		info.chain = gnuhash[4+bloomSize*uint32(vdsoBloomSizeScale)+nbucket:]
		info.isGNUHash = true
	} else {
		// Parse the hash table header.
		nbucket := hash[0]
		nchain := hash[1]
		info.bucket = hash[2 : 2+nbucket]
		info.chain = hash[2+nbucket : 2+nbucket+nchain]
	}

	// That's all we need.
	info.valid = true
}

func vdsoFindVersion(info *vdsoInfo, ver *vdsoVersionKey) int32 {
	if !info.valid {
		return 0
	}

	def := info.verdef
	for {
		if def.vd_flags&_VER_FLG_BASE == 0 {
			aux := (*elfVerdaux)(add(unsafe.Pointer(def), uintptr(def.vd_aux)))
			if def.vd_hash == ver.verHash && ver.version == gostringnocopy(&info.symstrings[aux.vda_name]) {
				return int32(def.vd_ndx & 0x7fff)
			}
		}

		if def.vd_next == 0 {
			break
		}
		def = (*elfVerdef)(add(unsafe.Pointer(def), uintptr(def.vd_next)))
	}

	return -1 // cannot match any version
}

func vdsoParseSymbols(name string, info *vdsoInfo, version int32) uintptr {
	if !info.valid {
		return 0
	}

	load := func(symIndex uint32, name string) uintptr {
		sym := &info.symtab[symIndex]
		typ := _ELF_ST_TYPE(sym.st_info)
		bind := _ELF_ST_BIND(sym.st_info)
		// On ppc64x, VDSO functions are of type _STT_NOTYPE.
		if typ != _STT_FUNC && typ != _STT_NOTYPE || bind != _STB_GLOBAL && bind != _STB_WEAK || sym.st_shndx == _SHN_UNDEF {
			return 0
		}
		if name != gostringnocopy(&info.symstrings[sym.st_name]) {
			return 0
		}
		// Check symbol version.
		if info.versym != nil && version != 0 && int32(info.versym[symIndex]&0x7fff) != version {
			return 0
		}

		return uintptr(info.loadOffset) + uintptr(sym.st_value)
	}

	if !info.isGNUHash {
		// Old-style DT_HASH table.
		hash := elfHash(name)
		for chain := info.bucket[hash%uint32(len(info.bucket))]; chain != 0; chain = info.chain[chain] {
			if p := load(chain, name); p != 0 {
				return p
			}
		}
		return 0
	}

	// New-style DT_GNU_HASH table.
	gnuhash := elfGNUHash(name)
	symIndex := info.bucket[gnuhash%uint32(len(info.bucket))]
	if symIndex < info.symOff {
		return 0
	}
	for ; ; symIndex++ {
		hash := info.chain[symIndex-info.symOff]
		if hash|1 == gnuhash|1 {
			// Found a hash match.
			if p := load(symIndex, name); p != 0 {
				return p
			}
		}
		if hash&1 != 0 {
			// End of chain.
			break
		}
	}
	return 0
}

func elfHash(name string) (h uint32) {
	for i := 0; i < len(name); i++ {
		h = h<<4 + uint32(name[i])
		g := h & 0xf0000000
		if g != 0 {
			h ^= g >> 24
		}
		h &= ^g
	}
	return
}

func elfGNUHash(name string) (h uint32) {
	h = 5381
	for i := 0; i < len(name); i++ {
		h = h*33 + uint32(name[i])
	}
	return
}

func init() {
	d, err := ioutil.ReadFile("/proc/self/auxv")
	if err != nil {
		panic(err)
	}
	var base unsafe.Pointer
	auxv := (*(*[128]uintptr)(unsafe.Pointer(&d[0])))[:len(d)/int(unsafe.Sizeof(uintptr(0)))]
	for i := 0; auxv[i] != _AT_NULL; i += 2 {
		tag, val := auxv[i], auxv[i+1]
		if tag != _AT_SYSINFO_EHDR || val == 0 {
			continue
		}
		vdsoInitFromSysinfoEhdr(&vdsoinfo, (*elfEhdr)(unsafe.Pointer(uintptr(base)+val)))
	}
	vdsoVersion = vdsoFindVersion(&vdsoinfo, &vdsoLinuxVersion)
	initVDSOAll()
}

func vdsoSym(name string) uintptr {
	return vdsoParseSymbols(name, &vdsoinfo, vdsoVersion)
}

func initVDSODefault(name string, def uintptr) uintptr {
	if p := vdsoSym(name); p != 0 {
		def = p
	}
	return def
}

func initVDSOAll() {
	vdsoGetCPU = initVDSODefault("__vdso_getcpu", 0xffffffffff600800)
}
