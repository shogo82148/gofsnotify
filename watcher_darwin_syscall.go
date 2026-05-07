//go:build darwin

package fsnotify

import (
	"reflect"
	"syscall"
	"unsafe"
	_ "unsafe"
)

func darwinKqueue() (int, error) {
	r0, _, e1 := libcSyscall(reflect.ValueOf(libc_kqueue_trampoline).Pointer(), 0, 0, 0)
	fd := int(r0)
	if e1 != 0 {
		return fd, e1
	}
	return fd, nil
}

func darwinKevent(kq int, changes []syscall.Kevent_t, events []syscall.Kevent_t, timeout *syscall.Timespec) (int, error) {
	var pChanges uintptr
	if len(changes) > 0 {
		pChanges = uintptr(unsafe.Pointer(&changes[0]))
	}
	var pEvents uintptr
	if len(events) > 0 {
		pEvents = uintptr(unsafe.Pointer(&events[0]))
	}
	r0, _, e1 := libcSyscall6(
		reflect.ValueOf(libc_kevent_trampoline).Pointer(),
		uintptr(kq),
		pChanges,
		uintptr(len(changes)),
		pEvents,
		uintptr(len(events)),
		uintptr(unsafe.Pointer(timeout)),
	)
	n := int(r0)
	if e1 != 0 {
		return n, e1
	}
	return n, nil
}

//go:linkname libcSyscall syscall.syscall
func libcSyscall(fn, a1, a2, a3 uintptr) (r1, r2 uintptr, err syscall.Errno)

//go:linkname libcSyscall6 syscall.syscall6
func libcSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

func libc_kqueue_trampoline()

//go:cgo_import_dynamic libc_kqueue kqueue "/usr/lib/libSystem.B.dylib"

func libc_kevent_trampoline()

//go:cgo_import_dynamic libc_kevent kevent "/usr/lib/libSystem.B.dylib"
