#pragma once

#include <stdint.h>
#include <objc/runtime.h>

// goBridge_Callback_Bind associates a Go callback registry key with an ObjC object
// for a specific selector. Called once per override when a subclass or protocol-
// callback instance is created. The key is stored as a retained associated object
// on obj, keyed by the SEL derived from selName. A key of 0 is a no-op.
void goBridge_Callback_Bind(void* obj, const char* selName, uint64_t key);

// goBridge_Callback_Lookup retrieves the callback key previously stored by
// goBridge_Callback_Bind for (obj, sel). Returns 0 if no binding exists.
// Called from every generated IMP trampoline before dispatching to Go.
uint64_t goBridge_Callback_Lookup(void* obj, SEL sel);
