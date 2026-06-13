//go:build darwin

package tel

import (
	"context"
	"errors"
	"unsafe"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/objc"
)

const tracerName = "github.com/deploymenttheory/go-bindings-macosplatform"

// OnException is called by RaiseIfException immediately before panicking with
// an ObjC exception. Wire this to your structured logger at application startup.
// Called from any goroutine; must be goroutine-safe.
var OnException func(reason string)

// OnCallbackPanic is called when a panic is recovered inside a generated
// //export callback (goCallBlock_* or goCallIMP_*). Wire this to your
// structured logger at application startup. Called from any goroutine.
var OnCallbackPanic func(name string, r interface{}, stack []byte)

// Call starts an OTEL span for the given ObjC method invocation and returns
// a cleanup func that ends the span and keeps recv and all args alive past the
// CGo call. Pass nil for recv on class methods and standalone functions. The
// variadic args should be every ObjC object argument whose Go wrapper must be
// prevented from being finalised while the CGo callee holds the raw pointer.
func Call(ctx context.Context, recv objc.Object, spanName string, args ...objc.Object) (context.Context, func()) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, spanName)
	alive := make([]objc.Object, 0, len(args)+1)
	if recv != nil {
		alive = append(alive, recv)
	}
	alive = append(alive, args...)
	if len(alive) == 0 {
		return ctx, func() { span.End() }
	}
	return ctx, func() {
		span.End()
		for _, a := range alive {
			objc.KeepAlive(a)
		}
	}
}

// RaiseIfException converts a caught ObjC exception to a Go panic.
// Calls OnException (if set) then records structured exception fields on the
// active span before panicking. This gives the application a chance to log the
// reason through its structured logger before the panic unwinds the stack.
func RaiseIfException(ctx context.Context, exc unsafe.Pointer) {
	if exc == nil {
		return
	}
	info := objc.ExceptionInfoFromPtr(exc)
	if h := OnException; h != nil {
		h(info.Reason)
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(errors.New(info.Reason))
	span.SetStatus(codes.Error, info.Reason)
	if info.Name != "" {
		span.SetAttributes(
			attribute.String("exception.name", info.Name),
			attribute.String("exception.reason", info.Reason),
		)
	}
	panic("ObjC exception [" + info.Name + "]: " + info.Reason)
}

// NSErrorToError converts an ObjC NSError to a Go error and records it on
// the active span.
func NSErrorToError(ctx context.Context, ptr unsafe.Pointer) error {
	err := objc.NSErrorToError(ptr)
	if err != nil {
		span := trace.SpanFromContext(ctx)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
