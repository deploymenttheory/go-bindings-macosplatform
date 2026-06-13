// ObjC block ABI implementation for Go-managed blocks.
// Compiled by CGo as part of the internal/blocks package.

#import <Foundation/Foundation.h>
#include "block_abi.h"

// goBlockUnregister and goBlockRetain are //export'd from registry.go;
// the linker resolves these at link time.
extern void goBlockUnregister(uint64_t key);
extern void goBlockRetain(uint64_t key);

// goBlockCopy is called by the ObjC runtime when Block_copy is invoked on a
// GoBlock. The struct content (including goKey) has already been copied by the
// runtime before this helper is called. We increment the Go reference count so
// the closure is not freed until all ObjC copies have been released.
void goBlockCopy(struct GoBlock *dst, struct GoBlock *src) {
    (void)src;
    goBlockRetain(dst->goKey);
}

// goBlockDispose is called by the ObjC runtime when a copied block is released.
// Decrement the Go reference count; when it reaches zero the closure is freed.
void goBlockDispose(struct GoBlock *src) {
    goBlockUnregister(src->goKey);
}

// Shared block descriptor used by all GoBlock values.
struct GoBlockDescriptor _goBlockDescriptor = {
    .reserved     = 0,
    .size         = sizeof(struct GoBlock),
    .copy_helper  = goBlockCopy,
    .dispose_helper = goBlockDispose,
};

// goFreeBlock decrements the Go registry reference count and frees the GoBlock
// heap allocation. For synchronous blocks (comparators, predicates) the
// reference count will reach zero here and the closure is unregistered.
// For async blocks that ObjC has already Block_copy'd, the count drops by one
// but the closure remains live until ObjC's final Block_release fires dispose.
void goFreeBlock(void *block) {
    if (!block) return;
    struct GoBlock *b = (struct GoBlock *)block;
    goBlockDispose(b);
    free(b);
}
