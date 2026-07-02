// IOReport.h — reconstructed prototypes for /usr/lib/libIOReport.dylib.
//
// Apple ships a linkable stub for this library in the public SDK
// ({SDK}/usr/lib/libIOReport.tbd) but no header: the API is private. These
// declarations are reverse-engineered and match the symbol surface exported
// by the .tbd and the calling conventions established by open-source
// consumers (powermetrics behaviour, socpowerbud, macmon, NeoAsitop).
//
// All object parameters and returns are CoreFoundation references at runtime
// (CFStringRef, CFDictionaryRef, CFMutableDictionaryRef, CFTypeRef). They are
// declared here as opaque pointer typedefs — ABI-identical — so this header
// stays self-contained: the generated CGo bridge compiles without pulling
// CoreFoundation into the C-library type universe, and the Go surface exposes
// unsafe.Pointer values that callers bridge with their own CF helpers.
//
//   IOReportStringRef        = CFStringRef        (Get rule: borrowed)
//   IOReportChannelRef       = CFDictionaryRef    (one channel in a sample)
//   IOReportChannelsRef      = CFMutableDictionaryRef (channel set)
//   IOReportSampleRef        = CFDictionaryRef    (sample; caller releases)
//   IOReportOptionsRef       = CFTypeRef          (pass NULL)
//
// Channels inside a sample dictionary are CFDictionaryRef values held in the
// sample's "IOReportChannels" CFArray; every IOReportChannel*/Simple*/State*
// accessor below takes one of those channel dictionaries.
//
// This header is scanned by Clang like any SDK header (see
// metadata/clibraries.json "ioreport", shim_header) and a copy is shipped
// inside the generated package's bridge/ directory so the CGo bridge
// compiles for module consumers. Private API caveat applies: symbols and
// channel layouts can change between macOS releases; consumers must treat
// every call as fallible.

#pragma once
#include <stdint.h>

typedef struct IOReportSubscription *IOReportSubscriptionRef;
typedef const void *IOReportStringRef;
typedef const void *IOReportChannelRef;
typedef void *IOReportChannelsRef;
typedef const void *IOReportSampleRef;
typedef const void *IOReportOptionsRef;

// ── Channel discovery ────────────────────────────────────────────────────────

// Returns a mutable dictionary describing every channel published on this
// system. Caller owns the result (Create/Copy rule).
IOReportChannelsRef IOReportCopyAllChannels(uint64_t options, uint64_t unused);

// Returns the channels in the named group (e.g. "Energy Model", "CPU Stats",
// "GPU Stats"). subgroup may be NULL for all subgroups. Caller owns the result.
IOReportChannelsRef IOReportCopyChannelsInGroup(IOReportStringRef group, IOReportStringRef subgroup, uint64_t options, uint64_t unused1, uint64_t unused2);

// Merges the channels of toAdd into target (both from IOReportCopyChannels*).
void IOReportMergeChannels(IOReportChannelsRef target, IOReportChannelsRef toAdd, IOReportOptionsRef unused);

// ── Subscription and sampling ────────────────────────────────────────────────

// Subscribes to the given channels. On success *subscribedChannels receives a
// dictionary of the channels actually subscribed (caller owns it). Returns a
// subscription object (caller owns; CFRelease when done) or NULL on failure.
IOReportSubscriptionRef IOReportCreateSubscription(void *allocator, IOReportChannelsRef desiredChannels, IOReportChannelsRef *subscribedChannels, uint64_t channelID, IOReportOptionsRef unused);

// Takes one sample of every subscribed channel. Caller owns the result.
IOReportSampleRef IOReportCreateSamples(IOReportSubscriptionRef subscription, IOReportChannelsRef subscribedChannels, IOReportOptionsRef unused);

// Computes the per-channel delta between two samples. Caller owns the result.
IOReportSampleRef IOReportCreateSamplesDelta(IOReportSampleRef previousSample, IOReportSampleRef currentSample, IOReportOptionsRef unused);

// ── Channel metadata (Get rule: returned strings are borrowed) ───────────────

IOReportStringRef IOReportChannelGetGroup(IOReportChannelRef channel);
IOReportStringRef IOReportChannelGetSubGroup(IOReportChannelRef channel);
IOReportStringRef IOReportChannelGetChannelName(IOReportChannelRef channel);
IOReportStringRef IOReportChannelGetUnitLabel(IOReportChannelRef channel);
uint64_t IOReportChannelGetChannelID(IOReportChannelRef channel);
int32_t IOReportChannelGetFormat(IOReportChannelRef channel);

// ── Simple (scalar) channels ─────────────────────────────────────────────────

int64_t IOReportSimpleGetIntegerValue(IOReportChannelRef channel, int32_t index);

// ── State (residency) channels ───────────────────────────────────────────────

int32_t IOReportStateGetCount(IOReportChannelRef channel);
IOReportStringRef IOReportStateGetNameForIndex(IOReportChannelRef channel, int32_t index);
int64_t IOReportStateGetResidency(IOReportChannelRef channel, int32_t index);

// ── Array channels ───────────────────────────────────────────────────────────

int64_t IOReportArrayGetValueAtIndex(IOReportChannelRef channel, int32_t index);
