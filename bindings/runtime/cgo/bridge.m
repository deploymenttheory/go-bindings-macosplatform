// This file is compiled by CGo as part of the internal/objc package.

#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#include "bridge.h"

void runtime_objc_retain(void *obj) {
    if (obj) {
        CFRetain(obj);
    }
}

void runtime_objc_release(void *obj) {
    if (obj) {
        CFRelease(obj);
    }
}

void *runtime_nsstring_from_cstring(const char *s) {
    if (!s) return nil;
    @autoreleasepool {
        NSString *str = [NSString stringWithUTF8String:s];
        return (__bridge void *)[str retain];
    }
}

char *runtime_nsstring_to_cstring(void *nsstr) {
    if (!nsstr) return NULL;
    @autoreleasepool {
        NSString *s = (__bridge NSString *)nsstr;
        const char *utf8 = [s UTF8String];
        if (!utf8) return NULL;
        return strdup(utf8);
    }
}

char *runtime_nserror_description(void *err) {
    if (!err) return NULL;
    @autoreleasepool {
        NSError *e = (__bridge NSError *)err;
        const char *desc = [[e localizedDescription] UTF8String];
        if (!desc) return NULL;
        return strdup(desc);
    }
}

void *runtime_objc_alloc_class(const char *className) {
    Class cls = objc_getClass(className);
    if (!cls) return nil;
    return [cls alloc]; // +1 retained; caller must call an init method
}

int runtime_is_main_thread(void) {
    return [NSThread isMainThread] ? 1 : 0;
}

char *runtime_nsexception_reason(void *ex) {
    if (!ex) return NULL;
    @autoreleasepool {
        NSException *e = (__bridge NSException *)ex;
        const char *reason = [[e reason] UTF8String];
        if (!reason) return NULL;
        return strdup(reason);
    }
}

char *runtime_nsexception_name(void *ex) {
    if (!ex) return NULL;
    @autoreleasepool {
        NSException *e = (__bridge NSException *)ex;
        const char *name = [[e name] UTF8String];
        if (!name) return NULL;
        return strdup(name);
    }
}


/* ── Associated objects ─────────────────────────────────────────────────── */

void runtime_set_associated_object(void *obj, const void *key, void *value, uintptr_t policy) {
    objc_setAssociatedObject((__bridge id)obj, key,
                             (__bridge id)value,
                             (objc_AssociationPolicy)policy);
}

void *runtime_get_associated_object(void *obj, const void *key) {
    id v = objc_getAssociatedObject((__bridge id)obj, key);
    return (__bridge void *)v;
}

void runtime_remove_associated_objects(void *obj) {
    objc_removeAssociatedObjects((__bridge id)obj);
}

/* ── Autorelease pool ───────────────────────────────────────────────────── */

void *runtime_autorelease_pool_push(void) {
    return [[NSAutoreleasePool alloc] init];
}

void runtime_autorelease_pool_pop(void *pool) {
    [(NSAutoreleasePool *)pool release];
}

/* ── Selector registration ──────────────────────────────────────────────── */

void *runtime_sel_register_name(const char *name) {
    return sel_registerName(name);
}

/* ── Class creation ─────────────────────────────────────────────────────── */

void *runtime_get_class(const char *name) {
    return objc_getClass(name);
}

void *runtime_get_protocol(const char *name) {
    return objc_getProtocol(name);
}

void *runtime_class_alloc_pair(void *superCls, const char *name) {
    return objc_allocateClassPair((Class)superCls, name, 0);
}

void runtime_class_register_pair(void *cls) {
    objc_registerClassPair((Class)cls);
}

void runtime_class_dispose_pair(void *cls) {
    objc_disposeClassPair((Class)cls);
}

bool runtime_class_add_method(void *cls, void *sel, void *imp, const char *types) {
    return class_addMethod((Class)cls, (SEL)sel, (IMP)imp, types);
}

bool runtime_class_add_protocol(void *cls, void *protocol) {
    return class_addProtocol((Class)cls, (Protocol *)protocol);
}

void *runtime_imp_from_block(void *block) {
    return imp_implementationWithBlock((__bridge id)block);
}

void *runtime_send_init(void *obj) {
    return [(id)obj init];
}
