#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>

// GoBlock is a manually-constructed ObjC block struct whose layout matches
// the ABI that the ObjC runtime and Block_copy/Block_release machinery expect.
// The invoke field holds a C function pointer to the per-signature trampoline.
// The goKey field is the only captured variable: an index into the Go closure
// registry. Trampolines extract goKey and dispatch to the registered Go closure.
struct GoBlock {
    void                    *isa;        // set to _NSConcreteStackBlock
    volatile int32_t         flags;
    int32_t                  reserved;
    void                    *invoke;     // C trampoline function pointer
    struct GoBlockDescriptor *descriptor;
    uint64_t                 goKey;      // Go closure registry key
};

// GoBlockDescriptor carries size and copy/dispose helper pointers per the
// Block ABI. A single shared instance is used for all GoBlock values.
struct GoBlockDescriptor {
    unsigned long reserved;
    unsigned long size;
    void (*copy_helper)(struct GoBlock *dst, struct GoBlock *src);
    void (*dispose_helper)(struct GoBlock *src);
};

// Flag bits relevant to our GoBlock usage.
#define BLOCK_HAS_COPY_DISPOSE  (1 << 25)

// Shared descriptor instance (defined in block_abi.m).
extern struct GoBlockDescriptor _goBlockDescriptor;

// Block lifecycle helpers (defined in block_abi.m).
// Called by the ObjC runtime when a block is copied or released.
void goBlockCopy(struct GoBlock *dst, struct GoBlock *src);
void goBlockDispose(struct GoBlock *src);

// goFreeBlock decrements the Go registry reference count for block->goKey and
// frees the GoBlock heap allocation. Call after the CGo call returns for
// synchronous blocks; not needed for blocks passed asynchronously (their
// lifecycle is managed by ObjC via copy/dispose).
void goFreeBlock(void *block);
