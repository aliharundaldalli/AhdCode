package ahdruntime

import (
	"crypto/rand"
	"encoding/base64"
)

// Identity.id produces a non-secret public identifier: 16 cryptographically
// random bytes encoded as unpadded URL-safe Base64 (22 characters).
//
// It is an identifier, not an authorization secret. Use Security.token()
// for credentials, CSRF tokens, and other secrets.
const ahdIdentityIDBytes = 16

func AhdIdentityID(errorClass *AhdClass) string {
	var raw [ahdIdentityIDBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		AhdRaiseClass(errorClass, "Identity random generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
