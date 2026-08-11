//go:build darwin

package puregolibs_test

import (
	"testing"

	applearchive "github.com/deploymenttheory/go-bindings-macosplatform/bindings/internal/raw/libraries/applearchive"
)

// TestAppleArchive_ProfileModes verifies the reimplemented AEAProfileGet*
// header-inline functions (pure switches on the profile id, no library call).
// Each of the 6 documented AEA profiles maps to a fixed ciphersuite/signature/
// encryption triple in AEAContext.h; a transcription error in the Go rewrite of
// those switches would produce a wrong mode here. Values from AEAContext.h.
func TestAppleArchive_ProfileModes(t *testing.T) {
	const (
		cipherHMAC     = 0 // AEA_CONTEXT_CIPHERSUITE_HKDF_SHA256_HMAC
		cipherAESCTR   = 1 // AEA_CONTEXT_CIPHERSUITE_HKDF_SHA256_AESCTR_HMAC
		sigNone        = 0
		sigECDSA       = 1
		encNone        = 0
		encSymmetric   = 1
		encECDHE       = 2
		encSCRYPT      = 3
		profHMAC       = 0 // HKDF_SHA256_HMAC__NONE__ECDSA_P256
		profSymNone    = 1 // AESCTR__SYMMETRIC__NONE
		profSymECDSA   = 2 // AESCTR__SYMMETRIC__ECDSA_P256
		profECDHENone  = 3 // AESCTR__ECDHE_P256__NONE
		profECDHEECDSA = 4 // AESCTR__ECDHE_P256__ECDSA_P256
		profSCRYPT     = 5 // AESCTR__SCRYPT__NONE
	)
	cases := []struct {
		profile                   uint32
		cipher, signature, encMod uint32
	}{
		{profHMAC, cipherHMAC, sigECDSA, encNone},
		{profSymNone, cipherAESCTR, sigNone, encSymmetric},
		{profSymECDSA, cipherAESCTR, sigECDSA, encSymmetric},
		{profECDHENone, cipherAESCTR, sigNone, encECDHE},
		{profECDHEECDSA, cipherAESCTR, sigECDSA, encECDHE},
		{profSCRYPT, cipherAESCTR, sigNone, encSCRYPT},
	}
	for _, c := range cases {
		if got := applearchive.AEAProfileGetCiphersuite(c.profile); got != c.cipher {
			t.Errorf("AEAProfileGetCiphersuite(%d) = %d; want %d", c.profile, got, c.cipher)
		}
		if got := applearchive.AEAProfileGetSignatureMode(c.profile); got != c.signature {
			t.Errorf("AEAProfileGetSignatureMode(%d) = %d; want %d", c.profile, got, c.signature)
		}
		if got := applearchive.AEAProfileGetEncryptionMode(c.profile); got != c.encMod {
			t.Errorf("AEAProfileGetEncryptionMode(%d) = %d; want %d", c.profile, got, c.encMod)
		}
	}

	// An unknown profile yields the documented UINT32_MAX sentinel.
	if got := applearchive.AEAProfileGetCiphersuite(999); got != 0xFFFFFFFF {
		t.Errorf("AEAProfileGetCiphersuite(unknown) = %#x; want 0xFFFFFFFF", got)
	}
}
