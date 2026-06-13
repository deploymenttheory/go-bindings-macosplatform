#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>

/* ObjC runtime retain / release used by all generated bridge files. */
void  runtime_objc_retain(void *obj);
void  runtime_objc_release(void *obj);

/* NSString <-> C string conversions. */
void *runtime_nsstring_from_cstring(const char *s);
char *runtime_nsstring_to_cstring(void *nsstr); /* caller must free() */

/* NSError localizedDescription as a C string. */
char *runtime_nserror_description(void *err); /* caller must free() */

/* NSException fields for panic messages. */
char *runtime_nsexception_reason(void *ex); /* caller must free() */
char *runtime_nsexception_name(void *ex);   /* caller must free() */

/* Weak references — ObjC runtime equivalents of objc_storeWeak / objc_loadWeakRetained.
   Using bridge wrappers avoids CGo confusion with ObjC runtime C-only headers. */
void  runtime_weak_store(void **slot, void *obj);   /* objc_storeWeak wrapper */
void *runtime_weak_load(void **slot);               /* objc_loadWeakRetained wrapper; +1 retain */

/* Allocate an uninitialized ObjC object of the named class (+alloc). */
void *runtime_objc_alloc_class(const char *className);

/* Returns 1 if the caller is on the macOS main thread, 0 otherwise. */
int runtime_is_main_thread(void);

/* Associated objects — attach arbitrary data to ObjC objects. */
void  runtime_set_associated_object(void *obj, const void *key, void *value, uintptr_t policy);
void *runtime_get_associated_object(void *obj, const void *key);
void  runtime_remove_associated_objects(void *obj);

/* Autorelease pool — required for ObjC work outside the main run loop. */
void *runtime_autorelease_pool_push(void);
void  runtime_autorelease_pool_pop(void *pool);

/* Dealloc listener — fires a Go callback when an ObjC object is deallocated. */
void *runtime_new_dealloc_listener(uintptr_t handle);

/* Selector registration. */
void *runtime_sel_register_name(const char *name);

/* ObjC runtime class creation and method installation. */
void *runtime_get_class(const char *name);
void *runtime_get_protocol(const char *name);
void *runtime_class_alloc_pair(void *superCls, const char *name);
void  runtime_class_register_pair(void *cls);
void  runtime_class_dispose_pair(void *cls);
bool  runtime_class_add_method(void *cls, void *sel, void *imp, const char *types);
bool  runtime_class_add_protocol(void *cls, void *protocol);
void *runtime_imp_from_block(void *block);

/* Send -init to an already-allocated object; returns the (possibly new) pointer. */
void *runtime_send_init(void *obj);
