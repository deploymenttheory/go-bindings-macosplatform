package typemap

// structCTypes is the set of ObjC/C struct type names that are passed by value
// in method signatures. These require special bridge handling: parameters must be
// dereferenced from void*, and returns must be heap-allocated and returned as void*.
var structCTypes = map[string]bool{
	"NSRect":                   true,
	"NSPoint":                  true,
	"NSSize":                   true,
	"NSRange":                  true,
	"NSEdgeInsets":             true,
	"NSAffineTransformStruct":  true,
	"NSDecimal":                true,
	"NSOperatingSystemVersion": true,
	"NSSwappedFloat":           true,
	"NSSwappedDouble":          true,
	"CGRect":            true,
	"CGPoint":           true,
	"CGSize":            true,
	"CGVector":          true,
	"CGAffineTransform": true,
	// CoreFoundation value-type structs
	"CFRange":          true,
	"CFSwappedFloat32": true,
	"CFSwappedFloat64": true,
	"CFGregorianDate":  true,
	"CFGregorianUnits": true,
	"CFStreamError":    true,
	"CFUUIDBytes":      true,
	// CoreGraphics structs
	"CGAffineTransformComponents": true,
	"CGColorBufferFormat":         true,
	"CGColorDataFormat":           true,
	"CGContentToneMappingInfo":    true,
	// CoreMedia value-type structs
	"CMTime":        true,
	"CMTimeRange":   true,
	"CMTimeMapping": true,
	// ColorSync structs
	"ColorSyncMD5": true,
	// Carbon / CarbonCore structs
	"UnsignedWide": true,
	"wide":         true,
	"Point":        true,
	"NumVersion":   true,
	// Metadata framework
	"MDQueryBatchingParams": true,
	// Security / CSSM structs
	"CSSM_DL_DB_HANDLE": true,
	"audit_token_t":     true,
	// POSIX structs
	"timeval":      true,
	"timespec":     true,
	"in_addr":      true,
	"ether_addr_t": true,
	// Foundation legacy hash/map table structs (NSHashTable, NSMapTable C API)
	"NSHashEnumerator":         true,
	"NSHashTableCallBacks":     true,
	"NSMapEnumerator":          true,
	"NSMapTableKeyCallBacks":   true,
	"NSMapTableValueCallBacks": true,
	// AppKit / UIKit value-type structs
	"NSDirectionalEdgeInsets": true,
	// QuartzCore value-type structs
	"CAFrameRateRange": true,
	"CATransform3D":    true,
	// Metal value-type structs
	"MTL4BufferRange":                        true,
	"MTLSize":                                true,
	"MTLOrigin":                              true,
	"MTLRegion":                              true,
	"MTLSamplePosition":                      true,
	"MTLCoordinate2D":                        true,
	"MTLClearColor":                          true,
	"MTLTextureSwizzleChannels":              true,
	"MTLPackedFloat3":                        true,
	"MTLPackedFloatQuaternion":               true,
	"MTLIndirectCommandBufferExecutionRange": true,
	"MTLScissorRect":                         true,
	"MTLViewport":                            true,
	"MTLVertexAmplificationViewMapping":      true,
	// SIMD matrix value types (AVFoundation/ARKit/etc)
	"matrix_float3x3":  true,
	"matrix_float4x3":  true,
	"matrix_float4x4":  true,
	"matrix_double4x4": true,
	// AVFoundation value-type structs
	"AVCaptionPoint":                    true,
	"AVCaptionSize":                     true,
	"AVCaptureTimecode":                 true,
	"AVSampleCursorSyncInfo":            true,
	"AVSampleCursorDependencyInfo":      true,
	"AVSampleCursorAudioDependencyInfo": true,
	"AVSampleCursorStorageRange":        true,
	"AVSampleCursorChunkInfo":           true,
	"AVPixelAspectRatio":                true,
	"AVEdgeWidths":                      true,
	// CoreMedia value-type structs
	"CMVideoDimensions": true,
	// CoreVideo value-type structs
	"CVTimeStamp": true,
	// CoreMIDI value-type structs
	"MIDI2DeviceManufacturer":  true,
	"MIDI2DeviceRevisionLevel": true,
	"MIDICIProfileID":          true,
	// CoreMotion value-type structs
	"CMAcceleration":            true,
	"CMRotationMatrix":          true,
	"CMQuaternion":              true,
	"CMRotationRate":            true,
	"CMMagneticField":           true,
	"CMCalibratedMagneticField": true,
	// CoreLocation value-type structs
	"CLLocationCoordinate2D": true,
	// CompositorServices value-type structs
	"cp_time_t": true,
	"cp_time":   true,
	// SIMD/simd vector and quaternion value types (ModelIO, MPS, ARKit, etc.)
	"simd_float2":  true,
	"simd_float3":  true,
	"simd_float4":  true,
	"simd_double2": true,
	"simd_double3": true,
	"simd_double4": true,
	"simd_quatf":   true,
	"simd_quatd":   true,
	"simd_uchar16": true,
	// vector_* aliases for simd types (including integer variants used by ModelIO)
	"vector_float2":  true,
	"vector_float3":  true,
	"vector_float4":  true,
	"vector_double2": true,
	"vector_double3": true,
	"vector_double4": true,
	"vector_uchar16": true,
	"vector_int2":    true,
	"vector_int3":    true,
	"vector_int4":    true,
	"vector_uint2":   true,
	"vector_uint3":   true,
	"vector_uint4":   true,
	// simd integer variants
	"simd_int2":  true,
	"simd_int3":  true,
	"simd_int4":  true,
	"simd_uint2": true,
	"simd_uint3": true,
	"simd_uint4": true,
	// ModelIO value-type structs
	"MDLAxisAlignedBoundingBox": true,
	// MetalPerformanceShaders value-type structs
	"MPSDimensionSlice":       true,
	"MPSStateTextureInfo":     true,
	"MPSImageCoordinate":      true,
	"MPSImageReadWriteParams": true,
	// Hypervisor SIMD vector type
	"hv_simd_fp_uchar16_t": true,
	// IOUSBHost value-type structs
	"IOUSBDeviceRequest": true,
	// ParavirtualizedGraphics value-type structs
	"PGDisplayCoord_t": true,
	// vecLib / Accelerate sparse matrix descriptor structs
	"SparseAttributesComplex_t": true,
	"SparseAttributes_t":        true,
	// BNNS graph types (Accelerate/vecLib)
	"bnns_graph_compile_options_t": true,
	"bnns_graph_t":                 true,
}

// IsStructCType reports whether the ObjC/C type name (after normalisation)
// is a known value-type struct that requires special bridge handling.
func IsStructCType(qt string) bool { return structCTypes[Normalise(qt)] }

// bsdStructTypes maps POSIX/BSD C struct names to their exported Go type names
// in the bsd package. These types originate from Darwin system headers rather
// than Apple SDK framework headers, so they are absent from .gometa.json
// metadata. The mapper checks this table after StructIndex to resolve them as
// bsd.TypeName references rather than unsafe.Pointer.
//
// Only value-type structs are listed here; polymorphic pointer bases (e.g.
// struct sockaddr) are intentionally excluded and remain unsafe.Pointer.
var bsdStructTypes = map[string]string{
	"ether_addr":   "EtherAddr", // struct ether_addr    { u_char octet[6]; }
	"ether_addr_t": "EtherAddr", // typedef ether_addr_t → struct ether_addr
	"timespec":     "Timespec",  // struct timespec      { time_t tv_sec; long tv_nsec; }
	"timeval":      "Timeval",   // struct timeval       { time_t tv_sec; suseconds_t tv_usec; }
	"in_addr":      "InAddr",    // struct in_addr       { uint32_t s_addr; }
	"in6_addr":     "In6Addr",   // struct in6_addr      { uint8_t s6_addr[16]; }
	"iovec":        "Iovec",     // struct iovec         { void *iov_base; size_t iov_len; }
}

// BSDStructCNames returns the set of C struct names that map to types in the
// bsd package. Used by the loader to pre-register these names in StructIndex.
func BSDStructCNames() []string {
	names := make([]string, 0, len(bsdStructTypes))
	for n := range bsdStructTypes {
		names = append(names, n)
	}
	return names
}
