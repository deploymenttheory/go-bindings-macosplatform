// ObjC associated-object runtime for Go IMP callback keys.
// Compiled by CGo as part of the internal/callbacks package.
// ARC is disabled; all retain/release is explicit.

#import <Foundation/Foundation.h>
#include "callbacks_abi.h"

// GoBridgeCallbackKey is a minimal NSObject wrapper that carries a uint64 key.
// It is stored as a retained associated object on each subclass/protocol-impl
// instance, keyed by the SEL of the method it handles.
@interface GoBridgeCallbackKey : NSObject {
@public
    uint64_t value;
}
@end

@implementation GoBridgeCallbackKey
- (void)dealloc {
    [super dealloc]; // non-ARC explicit dealloc
}
@end

void goBridge_Callback_Bind(void* obj, const char* selName, uint64_t key) {
    if (!obj || !selName || !key) return;
    SEL sel = sel_registerName(selName);
    GoBridgeCallbackKey* holder = [[GoBridgeCallbackKey alloc] init];
    holder->value = key;
    objc_setAssociatedObject((id)obj, (void*)sel, holder,
                             OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    [holder release];
}

uint64_t goBridge_Callback_Lookup(void* obj, SEL sel) {
    if (!obj || !sel) return 0;
    GoBridgeCallbackKey* holder =
        (GoBridgeCallbackKey*)objc_getAssociatedObject((id)obj, (void*)sel);
    if (!holder) return 0;
    return holder->value;
}
