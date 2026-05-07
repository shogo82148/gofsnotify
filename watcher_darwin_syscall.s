//go:build darwin

#include "textflag.h"

TEXT ·libc_kqueue_trampoline(SB),NOSPLIT,$0-0
JMP libc_kqueue(SB)

TEXT ·libc_kevent_trampoline(SB),NOSPLIT,$0-0
JMP libc_kevent(SB)
