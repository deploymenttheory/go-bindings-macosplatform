package rawlib

// appleArchiveManualBody is the hand-written purego body for AppleArchive's 25
// APPLE_ARCHIVE_INLINE AEAContext/AEAProfile functions, which have no exported
// dylib symbol. Each is a thin reimplementation of the SDK header inline:
//
//   - profile mode queries (AEAProfileGet*) are pure switches on the profile id;
//   - AEAContextGet<Field> delegate to the exported AEAContextGetFieldUInt/Blob;
//   - AEAContextSet<Field> delegate to the exported AEAContextSetFieldUInt/Blob.
//
// The four AEAContextGet/SetField{UInt,Blob} primitives are real
// APPLE_ARCHIVE_API exports but are not among the generated bindings (only the
// inline wrappers were scanned), so this file binds them itself. All field,
// representation, profile, and mode constants are the package's own generated
// enum consts, referenced directly. The AEAContext handle's Ptr() is nil-safe.
const appleArchiveManualBody = `import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	_aeaPrimOnce       sync.Once
	_pgAEAGetFieldUInt func(unsafe.Pointer, uint32) uint64
	_pgAEASetFieldUInt func(unsafe.Pointer, uint32, uint64) int32
	_pgAEAGetFieldBlob func(unsafe.Pointer, uint32, uint32, uint64, *uint8, *uint64) int32
	_pgAEASetFieldBlob func(unsafe.Pointer, uint32, uint32, *uint8, uint64) int32
)

// _aeaPrimitives binds the exported AEAContext field accessors the reimplemented
// header-inline functions delegate to (bound lazily on first use).
func _aeaPrimitives() {
	_loadOnce.Do(_loadLibrary)
	_aeaPrimOnce.Do(func() {
		if _applearchiveLib == 0 {
			return
		}
		_register("AEAContextGetFieldUInt", func() { purego.RegisterLibFunc(&_pgAEAGetFieldUInt, _applearchiveLib, "AEAContextGetFieldUInt") })
		_register("AEAContextSetFieldUInt", func() { purego.RegisterLibFunc(&_pgAEASetFieldUInt, _applearchiveLib, "AEAContextSetFieldUInt") })
		_register("AEAContextGetFieldBlob", func() { purego.RegisterLibFunc(&_pgAEAGetFieldBlob, _applearchiveLib, "AEAContextGetFieldBlob") })
		_register("AEAContextSetFieldBlob", func() { purego.RegisterLibFunc(&_pgAEASetFieldBlob, _applearchiveLib, "AEAContextSetFieldBlob") })
	})
}

func _aeaGetUInt(context_ *AEAContext, field AEAContextFields) uint64 {
	_aeaPrimitives()
	return _pgAEAGetFieldUInt(context_.Ptr(), uint32(field))
}

func _aeaSetUInt(context_ *AEAContext, field AEAContextFields, value uint64) int32 {
	_aeaPrimitives()
	return _pgAEASetFieldUInt(context_.Ptr(), uint32(field), value)
}

func _aeaGetBlob(context_ *AEAContext, field AEAContextFields, capacity uint64, buf *uint8, size *uint64) int32 {
	_aeaPrimitives()
	return _pgAEAGetFieldBlob(context_.Ptr(), uint32(field), uint32(AEA_CONTEXT_FIELD_REPRESENTATION_RAW), capacity, buf, size)
}

func _aeaSetBlob(context_ *AEAContext, field AEAContextFields, rep AEAContextFieldRepresentations, buf *uint8, size uint64) int32 {
	_aeaPrimitives()
	return _pgAEASetFieldBlob(context_.Ptr(), uint32(field), uint32(rep), buf, size)
}

// AEAContextGetProfile returns the context's AEA profile.
func AEAContextGetProfile(context_ *AEAContext) uint32 {
	return uint32(_aeaGetUInt(context_, AEA_CONTEXT_FIELD_PROFILE))
}

// AEAProfileGetCiphersuite returns the ciphersuite for an AEA profile.
func AEAProfileGetCiphersuite(profile uint32) uint32 {
	switch profile {
	case uint32(AEA_PROFILE__HKDF_SHA256_HMAC__NONE__ECDSA_P256):
		return uint32(AEA_CONTEXT_CIPHERSUITE_HKDF_SHA256_HMAC)
	case uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__ECDSA_P256),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__ECDSA_P256),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SCRYPT__NONE):
		return uint32(AEA_CONTEXT_CIPHERSUITE_HKDF_SHA256_AESCTR_HMAC)
	}
	return 0xFFFFFFFF // invalid
}

// AEAProfileGetSignatureMode returns the signature mode for an AEA profile.
func AEAProfileGetSignatureMode(profile uint32) uint32 {
	switch profile {
	case uint32(AEA_PROFILE__HKDF_SHA256_HMAC__NONE__ECDSA_P256),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__ECDSA_P256),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__ECDSA_P256):
		return uint32(AEA_CONTEXT_SIGNATURE_ECDSA_P256)
	case uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SCRYPT__NONE):
		return uint32(AEA_CONTEXT_SIGNATURE_NONE)
	}
	return 0xFFFFFFFF // invalid
}

// AEAProfileGetEncryptionMode returns the encryption mode for an AEA profile.
func AEAProfileGetEncryptionMode(profile uint32) uint32 {
	switch profile {
	case uint32(AEA_PROFILE__HKDF_SHA256_HMAC__NONE__ECDSA_P256):
		return uint32(AEA_CONTEXT_ENCRYPTION_NONE)
	case uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SYMMETRIC__ECDSA_P256):
		return uint32(AEA_CONTEXT_ENCRYPTION_SYMMETRIC)
	case uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__NONE),
		uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__ECDHE_P256__ECDSA_P256):
		return uint32(AEA_CONTEXT_ENCRYPTION_ECDHE_P256)
	case uint32(AEA_PROFILE__HKDF_SHA256_AESCTR_HMAC__SCRYPT__NONE):
		return uint32(AEA_CONTEXT_ENCRYPTION_SCRYPT)
	}
	return 0xFFFFFFFF // invalid
}

// AEAContextGetPaddingSize returns the context padding size.
func AEAContextGetPaddingSize(context_ *AEAContext) uint64 {
	return _aeaGetUInt(context_, AEA_CONTEXT_FIELD_PADDING_SIZE)
}

// AEAContextGetChecksumMode returns the context checksum mode.
func AEAContextGetChecksumMode(context_ *AEAContext) uint32 {
	return uint32(_aeaGetUInt(context_, AEA_CONTEXT_FIELD_CHECKSUM_MODE))
}

// AEAContextGetCompressionBlockSize returns the context compression block size.
func AEAContextGetCompressionBlockSize(context_ *AEAContext) uint64 {
	return _aeaGetUInt(context_, AEA_CONTEXT_FIELD_COMPRESSION_BLOCK_SIZE)
}

// AEAContextGetRawSize returns the context raw (uncompressed) size.
func AEAContextGetRawSize(context_ *AEAContext) uint64 {
	return _aeaGetUInt(context_, AEA_CONTEXT_FIELD_RAW_SIZE)
}

// AEAContextGetContainerSize returns the context container size.
func AEAContextGetContainerSize(context_ *AEAContext) uint64 {
	return _aeaGetUInt(context_, AEA_CONTEXT_FIELD_CONTAINER_SIZE)
}

// AEAContextGetAuthData copies the context auth data into auth_data.
func AEAContextGetAuthData(context_ *AEAContext, auth_data_capacity uint64, auth_data *uint8, auth_data_size *uint64) int32 {
	return _aeaGetBlob(context_, AEA_CONTEXT_FIELD_AUTH_DATA, auth_data_capacity, auth_data, auth_data_size)
}

// AEAContextGetSignatureEncryptionKey copies the signature encryption key into key.
func AEAContextGetSignatureEncryptionKey(context_ *AEAContext, key_capacity uint64, key *uint8, key_size *uint64) int32 {
	return _aeaGetBlob(context_, AEA_CONTEXT_FIELD_SIGNATURE_ENCRYPTION_KEY, key_capacity, key, key_size)
}

// AEAContextGetArchiveIdentifier copies the archive identifier into identifier.
func AEAContextGetArchiveIdentifier(context_ *AEAContext, identifier_capacity uint64, identifier *uint8, identifier_size *uint64) int32 {
	return _aeaGetBlob(context_, AEA_CONTEXT_FIELD_ARCHIVE_IDENTIFIER, identifier_capacity, identifier, identifier_size)
}

// AEAContextGetMainKey copies the main key into key.
func AEAContextGetMainKey(context_ *AEAContext, key_capacity uint64, key *uint8, key_size *uint64) int32 {
	return _aeaGetBlob(context_, AEA_CONTEXT_FIELD_MAIN_KEY, key_capacity, key, key_size)
}

// AEAContextSetCompressionBlockSize sets the context compression block size (capped at UINT32_MAX).
func AEAContextSetCompressionBlockSize(context_ *AEAContext, compression_block_size uint64) int32 {
	if compression_block_size > 0xFFFFFFFF {
		compression_block_size = 0xFFFFFFFF
	}
	return _aeaSetUInt(context_, AEA_CONTEXT_FIELD_COMPRESSION_BLOCK_SIZE, compression_block_size)
}

// AEAContextSetChecksumMode sets the context checksum mode.
func AEAContextSetChecksumMode(context_ *AEAContext, checksum_mode uint32) int32 {
	return _aeaSetUInt(context_, AEA_CONTEXT_FIELD_CHECKSUM_MODE, uint64(checksum_mode))
}

// AEAContextSetPaddingSize sets the context padding size.
func AEAContextSetPaddingSize(context_ *AEAContext, padding_size uint64) int32 {
	return _aeaSetUInt(context_, AEA_CONTEXT_FIELD_PADDING_SIZE, padding_size)
}

// AEAContextSetAuthData sets the context auth data.
func AEAContextSetAuthData(context_ *AEAContext, auth_data *uint8, auth_data_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_AUTH_DATA, AEA_CONTEXT_FIELD_REPRESENTATION_RAW, auth_data, auth_data_size)
}

// AEAContextSetSignatureEncryptionKey sets the signature encryption key.
func AEAContextSetSignatureEncryptionKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_SIGNATURE_ENCRYPTION_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_RAW, key, key_size)
}

// AEAContextSetSymmetricKey sets the symmetric key.
func AEAContextSetSymmetricKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_SYMMETRIC_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_RAW, key, key_size)
}

// AEAContextSetPassword sets the password.
func AEAContextSetPassword(context_ *AEAContext, password *uint8, password_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_PASSWORD, AEA_CONTEXT_FIELD_REPRESENTATION_RAW, password, password_size)
}

// AEAContextSetSigningPublicKey sets the signing public key (X9.63 representation).
func AEAContextSetSigningPublicKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_SIGNING_PUBLIC_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_X963, key, key_size)
}

// AEAContextSetSigningPrivateKey sets the signing private key (X9.63 representation).
func AEAContextSetSigningPrivateKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_SIGNING_PRIVATE_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_X963, key, key_size)
}

// AEAContextSetRecipientPublicKey sets the recipient public key (X9.63 representation).
func AEAContextSetRecipientPublicKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_RECIPIENT_PUBLIC_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_X963, key, key_size)
}

// AEAContextSetRecipientPrivateKey sets the recipient private key (X9.63 representation).
func AEAContextSetRecipientPrivateKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_RECIPIENT_PRIVATE_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_X963, key, key_size)
}

// AEAContextSetMainKey sets the main key.
func AEAContextSetMainKey(context_ *AEAContext, key *uint8, key_size uint64) int32 {
	return _aeaSetBlob(context_, AEA_CONTEXT_FIELD_MAIN_KEY, AEA_CONTEXT_FIELD_REPRESENTATION_RAW, key, key_size)
}
`

// appleArchiveManualFuncs is the set of AppleArchive C function names whose
// binding the manual body above replaces (header-inline, no exported symbol).
var appleArchiveManualFuncs = []string{
	"AEAContextGetProfile", "AEAProfileGetCiphersuite", "AEAProfileGetSignatureMode",
	"AEAProfileGetEncryptionMode", "AEAContextGetPaddingSize", "AEAContextGetChecksumMode",
	"AEAContextGetCompressionBlockSize", "AEAContextGetRawSize", "AEAContextGetContainerSize",
	"AEAContextGetAuthData", "AEAContextGetSignatureEncryptionKey", "AEAContextGetArchiveIdentifier",
	"AEAContextGetMainKey", "AEAContextSetCompressionBlockSize", "AEAContextSetChecksumMode",
	"AEAContextSetPaddingSize", "AEAContextSetAuthData", "AEAContextSetSignatureEncryptionKey",
	"AEAContextSetSymmetricKey", "AEAContextSetPassword", "AEAContextSetSigningPublicKey",
	"AEAContextSetSigningPrivateKey", "AEAContextSetRecipientPublicKey",
	"AEAContextSetRecipientPrivateKey", "AEAContextSetMainKey",
}
