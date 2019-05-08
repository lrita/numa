#include "textflag.h"

// long vdsoGetCPU(unsigned *, unsigned *, void *)
//
// func getcpu() {
//   vdsoGetCPU(&cpu, &node, NULL)
// }
TEXT ·getcpu(SB), NOSPLIT|NEEDCTXT, $0-0 // this function is running g0 stack, we can overflow safety.
	// We don't know how much stack space the VDSO code will need.
	// In particular, a kernel configured with CONFIG_OPTIMIZE_INLINING=n
	// and hardening can use a full page of stack space in gettime_sym
	// due to stack probes inserted to avoid stack/heap collisions.
	//
	// https://github.com/golang/go/issues/20427#issuecomment-343255844

	MOVQ	SP, BP	        // Save old SP; BP unchanged by C code.

	MOVQ	8(DX), DI       // &cpu
	MOVQ	16(DX), SI      // &node
	MOVQ	$0, DX	        // tcache = NULL

	SUBQ	$16, SP         //
	ANDQ	$~15, SP        // Align for C code

	MOVQ	·vdsoGetCPU(SB), AX
	CALL	AX

	MOVQ	BP, SP		// Restore real SP

	RET

TEXT ·GetCPUAndNode(SB),NOSPLIT,$32-16
	// check support fastway
	CMPB	·fastway(SB), $0
	JE	no_fastway
	// RDTSCP go1.11 support RDTSCP opcode but go1.10 not
	BYTE	$0x0F; BYTE $0x01; BYTE $0xF9
	MOVL	CX, AX
	SHRL	$12, AX
	ANDL	$4095, CX
	MOVQ	CX, cpu+0(FP)
	MOVQ	AX, node+8(FP)
	RET

no_fastway:
	MOVQ    $0, cpu+0(FP)
	MOVQ    $0, node+8(FP)

	LEAQ	·getcpu(SB), AX
	MOVQ	AX, fn-24(SP)
	LEAQ	cpu+0(FP), AX
	MOVQ	AX, cpu-16(SP)
	LEAQ	node+8(FP), AX
	MOVQ	AX, node-8(SP)

	LEAQ	fn-24(SP), AX
	MOVQ	AX, zone-32(SP)

	CALL	runtime·systemstack(SB)

	RET
