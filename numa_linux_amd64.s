#include "textflag.h"

TEXT ·fastcpuandnode(SB),NOSPLIT,$0-16
	RDTSCP
	MOVL CX, AX
	SHRL $12, AX
	ANDL $4095, CX
	MOVQ CX, cpu+0(FP)
	MOVQ AX, node+8(FP)
	RET
