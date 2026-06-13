// Weak reference bridge — compiled with -fno-objc-arc (package default).
// objc_storeWeak and objc_loadWeak are available in <objc/runtime.h> without ARC.
// We explicitly retain the loaded object so the caller receives a +1-owned pointer.

#import <Foundation/Foundation.h>
#include <objc/runtime.h>
#include "bridge.h"

void runtime_weak_store(void **slot, void *obj) {
    objc_storeWeak((id *)slot, (id)obj);
}

void *runtime_weak_load(void **slot) {
    // objc_loadWeak returns an autoreleased id (or nil if deallocated).
    // We explicitly retain it inside an autorelease pool so the caller gets
    // a +1-owned pointer that outlives the pool drain.
    @autoreleasepool {
        id obj = objc_loadWeak((id *)slot);
        if (!obj) return NULL;
        [obj retain];   // balance: caller must Release when done
        return (void *)obj;
    }
}
