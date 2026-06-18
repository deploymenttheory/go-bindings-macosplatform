//go:build darwin

// Package extension implements LuLu's network-extension side: the
// NEFilterDataProvider subclass that judges every new flow, process attribution
// via libproc, and the XPC daemon the app talks to. Mirrors LuLu's Extension/.
package extension

import (
	"unsafe"

	"github.com/deploymenttheory/go-bindings-macosplatform/opinionated/idiomatic/libraries/libproc"
)

// procPidPathInfoMaxSize matches PROC_PIDPATHINFO_MAXSIZE (4 * PATH_MAX).
const procPidPathInfoMaxSize = 4096

// ProcessPath returns the executable path for pid via libproc's proc_pidpath,
// or "" when it cannot be determined. This is the Go equivalent of LuLu's
// Process/Binary path resolution.
func ProcessPath(pid int32) string {
	if pid <= 0 {
		return ""
	}
	buf := make([]byte, procPidPathInfoMaxSize)
	n := libproc.Pidpath(pid, unsafe.Pointer(&buf[0]), uint32(len(buf)))
	if n <= 0 {
		return ""
	}
	return string(buf[:n])
}
