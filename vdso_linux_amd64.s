#include "textflag.h"

// func getcpu()(cpu int, node int)
// =>
// long getcpu(unsigned *, unsigned *, void *)
TEXT ·getcpu(SB),$4112-16
	// We don't know how much stack space the VDSO code will need.
	// In particular, a kernel configured with CONFIG_OPTIMIZE_INLINING=n
	// and hardening can use a full page of stack space in gettime_sym
	// due to stack probes inserted to avoid stack/heap collisions.
	//
	// https://github.com/golang/go/issues/20427#issuecomment-343255844

	MOVQ	SP, BP	        // Save old SP; BP unchanged by C code.

	MOVQ	$0, cpu+0(FP)
	MOVQ	$0, node+8(FP)
	LEAQ	cpu+0(FP), DI       // &cpu
	LEAQ	node+8(FP), SI      // &node
	MOVQ	$0, DX	            // tcache = NULL

	ADDQ	$4096, SP       // make vdso using the callee stack.
	ANDQ	$~15, SP        // Align for C code

	MOVQ	·vdsoGetCPU(SB), AX
	CALL	AX

	MOVQ	BP, SP		// Restore real SP

	RET
