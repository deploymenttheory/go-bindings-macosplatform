# macOS 27 degradation review

Each entry below is new relative to the SDK 26.5 baseline. Existing entries that
disappeared are removed from the baseline; no ratchet is disabled.

| Diagnostic | Reason for retaining the fallback |
| --- | --- |
| `AVFoundation: "struct OpaqueVTCompressionSession *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `AudioToolbox: "union (unnamed union at /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/System/Library/Frameworks/AudioToolbox.framework/Headers/AUComponent.h:984:2)" → unsafe.Pointer (unresolved named type)` | Existing fallback; SDK header line shifted. |
| `ComputeGraph: "float __attribute__((ext_vector_type(3)))" → unsafe.Pointer (unresolved named type)` | SIMD vector ABI has no supported direct Go representation; retain the existing opaque fallback policy. |
| `CoreGraphics: "CGPDFMarkedContentItem" → unsafe.Pointer (unresolved named type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `CoreGraphics: "CGPDFStructureElement" → unsafe.Pointer (unresolved named type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `CoreGraphics: "struct CGPDFMarkedContentItem *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `CoreGraphics: "struct CGPDFStructureElement *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `HIServices: "AXObserverRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `HIServices: "AXUIElementRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `HIServices: "CFTypeRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `HIServices: "PasteboardRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `MPSFunctions: "NSZone *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MPSFunctions: "const struct CGColorConversionInfo *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MPSFunctions: "float __attribute__((ext_vector_type(4)))" → unsafe.Pointer (unresolved named type)` | SIMD vector ABI has no supported direct Go representation; retain the existing opaque fallback policy. |
| `MPSFunctions: "struct CGColorSpace *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MPSFunctions: "struct _NSZone" → unsafe.Pointer (unresolved named type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MPSImage: MPSBinaryImageKernel.encodeToCommandBuffer:inPlacePrimaryTexture:secondaryTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `MPSImage: MPSBinaryImageKernel.encodeToCommandBuffer:primaryTexture:inPlaceSecondaryTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `MPSImage: MPSUnaryImageKernel.encodeToCommandBuffer:inPlaceTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `MediaAccessibility: "CFErrorRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `MediaAccessibility: "struct __CFError *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MediaToolbox: "const struct opaqueCMFormatDescription *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MetalPerformanceShaders: "struct CGColorSpace *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `MetalPerformanceShaders: MPSBinaryImageKernel.encodeToCommandBuffer:inPlacePrimaryTexture:secondaryTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `MetalPerformanceShaders: MPSBinaryImageKernel.encodeToCommandBuffer:primaryTexture:inPlaceSecondaryTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `MetalPerformanceShaders: MPSUnaryImageKernel.encodeToCommandBuffer:inPlaceTexture:fallbackCopyAllocator: param copyAllocator → objc.Block (block return type "metal.MTLTexture" not bridgeable)` | Existing callback return is now parsed correctly as MTLTexture; retain the explicit objc.Block fallback for object returns. |
| `PencilKit: "const struct CGPath *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `PencilKit: PKStrokePath.initWithBezierPath:creationDate:pointProvider: param pointProvider → objc.Block (block return type "*PKStrokePoint" not bridgeable)` | New object-returning callback; callers use the explicit objc.Block representation. |
| `Security: "CFDateRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CFDictionaryRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CFStringRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CFTypeRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CFURLRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CMSDecoderRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "CMSEncoderRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SSLContextRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecACLRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecAccessRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecCertificateRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecCodeRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecIdentityRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecIdentitySearchRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecKeyRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecKeychainItemRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecKeychainRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecKeychainSearchRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecPolicyRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecPolicySearchRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecRequirementRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecStaticCodeRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecTrustRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Security: "SecTrustedApplicationRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `SpeechSynthesis: "CFStringRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `SpeechSynthesis: "CFTypeRef *" → unsafe.Pointer (unresolved pointer type)` | Existing pointer fallback; the scanner now removes ownership annotations from the type. |
| `Virtualization: "struct __SecCertificate *" → unsafe.Pointer (unresolved pointer type)` | Opaque C handle or pointer introduced on this surface; unsafe.Pointer preserves the pointer ABI without inventing a Go value layout. |
| `vecLib: "union bnns_graph_argument_t::(anonymous at /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/System/Library/Frameworks/vecLib.framework/Headers/BNNS/bnns_graph.h:532:3)" → unsafe.Pointer (unresolved named type)` | Existing fallback; SDK header line shifted. |
