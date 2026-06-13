//go:build darwin

package cgo

// #cgo CFLAGS: -fno-objc-arc -x objective-c
// #cgo LDFLAGS: -framework Foundation
// #include <Foundation/Foundation.h>
// #include <stdio.h>
//
// static void cUncaughtExceptionHandler(NSException *exc) {
//     fprintf(stderr, "\n=== ObjC uncaught exception ===\n");
//     fprintf(stderr, "Name:   %s\n", [[exc name]   UTF8String] ?: "<nil>");
//     fprintf(stderr, "Reason: %s\n", [[exc reason] UTF8String] ?: "<nil>");
//     fprintf(stderr, "Stack:\n");
//     for (NSString *frame in [exc callStackSymbols]) {
//         fprintf(stderr, "  %s\n", [frame UTF8String]);
//     }
//     fprintf(stderr, "===============================\n");
//     fflush(stderr);
//     // Return; the process terminates immediately after this handler returns.
// }
//
// static void installUncaughtExceptionHandler(void) {
//     NSSetUncaughtExceptionHandler(&cUncaughtExceptionHandler);
// }
import "C"

// InstallCrashHandlers installs an ObjC uncaught-exception handler that writes
// the exception name, reason, and full call-stack symbols to stderr before the
// process terminates. Call this as early as possible in main().
func InstallCrashHandlers() {
	C.installUncaughtExceptionHandler()
}
