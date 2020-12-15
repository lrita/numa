package gonuma

import (
	"io/ioutil"
	"unsafe"
)

const (
	AtNull        = 0 // End of vector
	AtSysInfoEHdr = 33

	PtLoad    = 1 /* Loadable program segment */
	PtDynamic = 2 /* Dynamic linking information */

	DtNull    = 0          /* Marks end of dynamic section */
	DtHash    = 4          /* Dynamic symbol hash table */
	DtStrTab  = 5          /* Address of string table */
	DtSymTab  = 6          /* Address of symbol table */
	DtGNUHash = 0x6ffffef5 /* GNU-style dynamic symbol hash table */
	DtVerSym  = 0x6ffffff0
	DtVerDef  = 0x6ffffffc

	VerFlagBase = 0x1 /* Version definition of file itself */

	ShnUndef = 0 /* Undefined section */

	ShtDynSym = 11 /* Dynamic linker symbol table */

	SttFunc = 2 /* Symbol is a code object */

	SttNoType = 0 /* Symbol type is not specified */

	StbGlobal = 1 /* Global symbol */
	StbWeak   = 2 /* Weak symbol */

	EINIdent = 16

	// vdsoArrayMax is the byte-size of a maximally sized array on this
	// architecture.
	// See cmd/compile/internal/amd64/galign.go arch.MAXWIDTH
	// initialization.
	vdsoArrayMax = 1<<50 - 1

	// Maximum indices for the array types used when traversing the vDSO ELF
	// structures.
	// Computed from architecture-specific max provided by vdso_linux_*.go
	VdsoSymTabSize     = vdsoArrayMax / unsafe.Sizeof(elfSym{})
	vdsoDynSize        = vdsoArrayMax / unsafe.Sizeof(elfDyn{})
	VdsoSymStringsSize = vdsoArrayMax     // byte
	vdsoVerSymSize     = vdsoArrayMax / 2 // uint16
	vdsoHashSize       = vdsoArrayMax / 4 // uint32

	// vdsoBloomSizeScale is a scaling factor for gnuhash tables which are
	// uint32 indexed,
	// but contain uintptrs
	vdsoBloomSizeScale = unsafe.Sizeof(uintptr(0)) / 4 // uint32
)

type vdsoVersionKey struct {
	version string
	verHash uint32
}

// ELF64 structure definitions for use by the vDSO loader

type elfSym struct {
	stName   uint32
	stInfo   byte
	st_other byte
	stShndx  uint16
	stValue  uint64
	st_size  uint64
}

type elfVerdef struct {
	vdVersion uint16 /* Version revision */
	vdFlags   uint16 /* Version information */
	vdNdx     uint16 /* Version Index */
	vdCnt     uint16 /* Number of associated aux entries */
	vdHash    uint32 /* Version name hash value */
	vdAux     uint32 /* Offset in bytes to verdaux array */
	vdNext    uint32 /* Offset in bytes to next verdef entry */
}

type elfEhdr struct {
	eIdent     [EINIdent]byte /* Magic number and other info */
	eType      uint16         /* Object file type */
	eMachine   uint16         /* Architecture */
	eVersion   uint32         /* Object file version */
	eEntry     uint64         /* Entry point virtual address */
	ePhoff     uint64         /* Program header table file offset */
	eShoff     uint64         /* Section header table file offset */
	eFlags     uint32         /* Processor-specific flags */
	eEhsize    uint16         /* ELF header size in bytes */
	ePhentsize uint16         /* Program header table entry size */
	ePhnum     uint16         /* Program header table entry count */
	eShentsize uint16         /* Section header table entry size */
	eShnum     uint16         /* Section header table entry count */
	eShstrndx  uint16         /* Section header string table index */
}

type elfPhdr struct {
	pType   uint32 /* Segment type */
	pFlags  uint32 /* Segment flags */
	pOffset uint64 /* Segment file offset */
	pVaddr  uint64 /* Segment virtual address */
	pPaddr  uint64 /* Segment physical address */
	pFilesz uint64 /* Segment size in file */
	pMemsz  uint64 /* Segment size in memory */
	pAlign  uint64 /* Segment alignment */
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
	dTag int64  /* Dynamic entry type */
	dVal uint64 /* Integer value */
}

type elfVerdaux struct {
	vdaName uint32 /* Version or dependency names */
	vdaNext uint32 /* Offset in bytes to next verdaux entry */
}

type vdsoInfo struct {
	valid bool

	/* Load information */
	loadAddr   unsafe.Pointer
	loadOffset unsafe.Pointer /* loadAddr - recorded vaddr */

	/* Symbol table */
	symtab     *[VdsoSymTabSize]elfSym
	symstrings *[VdsoSymStringsSize]byte
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

/* How to extract and insert information held in the stInfo field.  */
func ELFstBind(val byte) byte { return val >> 4 }
func ELFstType(val byte) byte { return val & 0xf }

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

	pt := unsafe.Pointer(uintptr(info.loadAddr) + uintptr(hdr.ePhoff))

	// We need two things from the segment table: the load offset
	// and the dynamic table.
	var foundVaddr bool
	var dyn *[vdsoDynSize]elfDyn
	for i := uint16(0); i < hdr.ePhnum; i++ {
		pt := (*elfPhdr)(add(pt, uintptr(i)*unsafe.Sizeof(elfPhdr{})))
		switch pt.pType {
		case PtLoad:
			if !foundVaddr {
				foundVaddr = true
				info.loadOffset = unsafe.Pointer(
					uintptr(info.loadAddr) + uintptr(pt.pOffset-pt.pVaddr),
				)
			}

		case PtDynamic:
			dyn = (*[vdsoDynSize]elfDyn)(
				unsafe.Pointer(
					uintptr(info.loadAddr) + uintptr(pt.pOffset),
				),
			)
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
	for i := 0; dyn[i].dTag != DtNull; i++ {
		dt := &dyn[i]
		p := unsafe.Pointer(uintptr(info.loadOffset) + uintptr(dt.dVal))
		switch dt.dTag {
		case DtStrTab:
			info.symstrings = (*[VdsoSymStringsSize]byte)(unsafe.Pointer(p))
		case DtSymTab:
			info.symtab = (*[VdsoSymTabSize]elfSym)(unsafe.Pointer(p))
		case DtHash:
			hash = (*[vdsoHashSize]uint32)(unsafe.Pointer(p))
		case DtGNUHash:
			gnuhash = (*[vdsoHashSize]uint32)(unsafe.Pointer(p))
		case DtVerSym:
			info.versym = (*[vdsoVerSymSize]uint16)(unsafe.Pointer(p))
		case DtVerDef:
			info.verdef = (*elfVerdef)(unsafe.Pointer(p))
		}
	}

	if info.symstrings == nil || info.symtab == nil ||
		(hash == nil && gnuhash == nil) {
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
		if def.vdFlags&VerFlagBase == 0 {
			aux := (*elfVerdaux)(
				add(unsafe.Pointer(def), uintptr(def.vdAux)),
			)
			if def.vdHash == ver.verHash &&
				ver.version == gostringnocopy(
					&info.symstrings[aux.vdaName],
				) {
				return int32(def.vdNdx & 0x7fff)
			}
		}

		if def.vdNext == 0 {
			break
		}
		def = (*elfVerdef)(add(unsafe.Pointer(def), uintptr(def.vdNext)))
	}

	return -1 // cannot match any version
}

func vdsoParseSymbols(name string, info *vdsoInfo, version int32) uintptr {
	if !info.valid {
		return 0
	}

	load := func(symIndex uint32, name string) uintptr {
		sym := &info.symtab[symIndex]
		typ := ELFstType(sym.stInfo)
		bind := ELFstBind(sym.stInfo)
		// On ppc64x, VDSO functions are of type SttNoType.
		if typ != SttFunc && typ != SttNoType || bind != StbGlobal && bind != StbWeak ||
			sym.stShndx == ShnUndef {
			return 0
		}
		if name != gostringnocopy(&info.symstrings[sym.stName]) {
			return 0
		}
		// Check symbol version.
		if info.versym != nil && version != 0 &&
			int32(info.versym[symIndex]&0x7fff) != version {
			return 0
		}

		return uintptr(info.loadOffset) + uintptr(sym.stValue)
	}

	if !info.isGNUHash {
		// Old-style DT_HASH table.
		hash := ELFHash(name)
		for chain := info.bucket[hash%uint32(len(info.bucket))]; chain != 0; chain = info.chain[chain] {
			if p := load(chain, name); p != 0 {
				return p
			}
		}
		return 0
	}

	// New-style DT_GNU_HASH table.
	gnuhash := ELFGNUHash(name)
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

func ELFHash(name string) (h uint32) {
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

func ELFGNUHash(name string) (h uint32) {
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
	for i := 0; auxv[i] != AtNull; i += 2 {
		tag, val := auxv[i], auxv[i+1]
		if tag != AtSysInfoEHdr || val == 0 {
			continue
		}
		vdsoInitFromSysinfoEhdr(
			&vdsoinfo,
			(*elfEhdr)(unsafe.Pointer(uintptr(base)+val)),
		)
	}
	vdsoVersion = vdsoFindVersion(&vdsoinfo, &vdsoLinuxVersion)
	initVDSOAll()
}

func VdsoSym(name string) uintptr {
	return vdsoParseSymbols(name, &vdsoinfo, vdsoVersion)
}

func initVDSODefault(name string, def uintptr) uintptr {
	if p := VdsoSym(name); p != 0 {
		def = p
	}
	return def
}

func initVDSOAll() {
	vdsoGetCPU = initVDSODefault("__vdso_getcpu", 0xffffffffff600800)
}
