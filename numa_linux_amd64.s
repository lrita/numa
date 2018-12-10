#include "textflag.h"

TEXT ·fastcpuandnode(SB),NOSPLIT,$0-16
	BYTE $0x0F; BYTE $0x01; BYTE $0xF9 // RDTSCP go1.11 support RDTSCP opcode but go1.10 not
	MOVL CX, AX
	SHRL $12, AX
	ANDL $4095, CX
	MOVQ CX, cpu+0(FP)
	MOVQ AX, node+8(FP)
	RET
