//go:build darwin

// Package oslog provides hand-crafted helpers for emitting messages to the
// macOS unified logging system, mirroring Swift's os.Logger(subsystem:category:).
//
// The generated bindings cannot cover emission: the OSLog ObjC framework
// (bindings/frameworks/oslog) only exposes the read-side API (OSLogStore,
// OSLogEntry), because Apple ships log emission exclusively as the C os_log
// macros. Those macros expand to _os_log_impl with a compiler-packed argument
// buffer and the caller image's __dso_handle, so the only correct way to call
// them from Go is through this C shim, which lets the C compiler do the
// packing with a fixed "%{public}s" format.
package oslog

/*
#include <os/log.h>
#include <stdint.h>
#include <stdlib.h>

static os_log_t orin_os_log_create(const char *subsystem, const char *category) {
	return os_log_create(subsystem, category);
}

static void orin_os_log(os_log_t log, uint8_t type, const char *message) {
	os_log_with_type(log, type, "%{public}s", message);
}
*/
import "C"

import "unsafe"

// LogType mirrors os_log_type_t.
type LogType uint8

const (
	LogTypeDefault LogType = 0x00
	LogTypeInfo    LogType = 0x01
	LogTypeDebug   LogType = 0x02
	LogTypeError   LogType = 0x10
	LogTypeFault   LogType = 0x11
)

// Logger mirrors Swift's os.Logger(subsystem:category:). Loggers are cheap,
// safe for concurrent use, and live for the process lifetime (os_log_t
// objects are never released, matching Apple's guidance).
type Logger struct {
	log C.os_log_t
}

// NewLogger creates a unified-logging logger for the given subsystem and
// category, mirroring os.Logger(subsystem:category:).
func NewLogger(subsystem string, category string) *Logger {
	cSubsystem := C.CString(subsystem)
	defer C.free(unsafe.Pointer(cSubsystem))
	cCategory := C.CString(category)
	defer C.free(unsafe.Pointer(cCategory))

	return &Logger{log: C.orin_os_log_create(cSubsystem, cCategory)}
}

// Log emits message at the given level.
func (l *Logger) Log(logType LogType, message string) {
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	C.orin_os_log(l.log, C.uint8_t(logType), cMessage)
}

// Default emits message at the default level (Swift Logger.log).
func (l *Logger) Default(message string) { l.Log(LogTypeDefault, message) }

// Info emits message at the info level (Swift Logger.info).
func (l *Logger) Info(message string) { l.Log(LogTypeInfo, message) }

// Debug emits message at the debug level (Swift Logger.debug).
func (l *Logger) Debug(message string) { l.Log(LogTypeDebug, message) }

// Error emits message at the error level (Swift Logger.error).
func (l *Logger) Error(message string) { l.Log(LogTypeError, message) }

// Fault emits message at the fault level (Swift Logger.fault).
func (l *Logger) Fault(message string) { l.Log(LogTypeFault, message) }
