// dealloc_listener.m — compiled with -fno-objc-arc as part of internal/objc.
//
// GoObjcDeallocListener is a tiny ObjC object stored as an associated object
// on a target. When the target is released and its associated objects are
// cleared, this listener's -dealloc fires, which calls back into Go via the
// stored cgo.Handle so that Go closures can be notified of ObjC object death.

#import <Foundation/Foundation.h>
#include "bridge.h"

// Forward declaration of the Go-exported callback.
extern void goObjcDeallocFired(uintptr_t handle);

@interface GoObjcDeallocListener : NSObject {
    uintptr_t _handle;
}
- (instancetype)initWithHandle:(uintptr_t)handle;
@end

@implementation GoObjcDeallocListener

- (instancetype)initWithHandle:(uintptr_t)handle {
    if ((self = [super init])) {
        _handle = handle;
    }
    return self;
}

- (void)dealloc {
    goObjcDeallocFired(_handle);
    [super dealloc];
}

@end

void *runtime_new_dealloc_listener(uintptr_t handle) {
    // Returns +1-retained; caller must attach as associated object then Release.
    return [[GoObjcDeallocListener alloc] initWithHandle:handle];
}
