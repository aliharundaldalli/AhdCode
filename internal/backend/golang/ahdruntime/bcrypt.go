package ahdruntime

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// This file is the bcrypt part of the Security module: Security.bcryptHash
// and Security.bcryptVerify, over the pinned golang.org/x/crypto/bcrypt that
// AhdCode already depends on (see ahdruntime/bcryptvendor). bcrypt is provided
// for compatibility with existing password databases and for migration to
// Argon2id; it is never chosen automatically. Like the MySQL runtime, this
// file joins a generated program only when the program uses bcrypt.
//
// No message raised here contains the password, the hash, or a Go error.

// ahdBcryptCost is the fixed work factor of every new hash. It is not a
// parameter: AhdCode publishes one sound default instead of a tuning knob.
const ahdBcryptCost = 12

// ahdBcryptMaxCost bounds the work factor a hash being verified may name, so
// a hostile encodedHash cannot make one verification run for hours.
const ahdBcryptMaxCost = 16

// ahdBcryptMaxPassword is bcrypt's input limit. A longer password is refused
// rather than silently truncated, in hashing and in verification alike.
const ahdBcryptMaxPassword = 72

func ahdBcryptPassword(password string) *AhdFault {
	if len(password) > ahdBcryptMaxPassword {
		return ahdFault("SecurityError", "bcrypt accepts a password of at most 72 UTF-8 bytes")
	}
	return nil
}

// ahdBcryptEncoding checks the modular crypt form this module verifies:
// $2a$, $2b$, or $2y$, a two-digit cost 04..16, and 53 characters of bcrypt's
// base64 alphabet (22 of salt, 31 of hash): 60 characters in all.
func ahdBcryptEncoding(encoded string) *AhdFault {
	malformed := ahdFault("SecurityError", "bcrypt hash is malformed or uses an unsupported format")
	if len(encoded) != 60 || encoded[0] != '$' || encoded[1] != '2' || encoded[3] != '$' || encoded[6] != '$' {
		return malformed
	}
	switch encoded[2] {
	case 'a', 'b', 'y':
	default:
		return malformed
	}
	if encoded[4] < '0' || encoded[4] > '9' || encoded[5] < '0' || encoded[5] > '9' {
		return malformed
	}
	cost := int(encoded[4]-'0')*10 + int(encoded[5]-'0')
	if cost < bcrypt.MinCost || cost > ahdBcryptMaxCost {
		return ahdFault("SecurityError", "bcrypt hash cost is outside the supported range 04..16")
	}
	for index := 7; index < len(encoded); index++ {
		character := encoded[index]
		if !(character == '.' || character == '/' || character >= 'A' && character <= 'Z' ||
			character >= 'a' && character <= 'z' || character >= '0' && character <= '9') {
			return malformed
		}
	}
	return nil
}

// AhdSecurityBcrypt hashes password at cost 12 and returns the $2a$ form.
func AhdSecurityBcrypt(password string) (string, *AhdFault) {
	if fault := ahdBcryptPassword(password); fault != nil {
		return "", fault
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), ahdBcryptCost)
	if err != nil {
		return "", ahdFault("SecurityError", "bcrypt hashing failed")
	}
	return string(hash), nil
}

// AhdSecurityBcryptCheck reports whether password matches encodedHash. A wrong
// password is false; a malformed or unsupported hash raises SecurityError.
func AhdSecurityBcryptCheck(password, encodedHash string) (bool, *AhdFault) {
	if fault := ahdBcryptPassword(password); fault != nil {
		return false, fault
	}
	if fault := ahdBcryptEncoding(encodedHash); fault != nil {
		return false, fault
	}
	err := bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return false, nil
	default:
		return false, ahdFault("SecurityError", "bcrypt hash is malformed or uses an unsupported format")
	}
}

func AhdSecurityBcryptHash(password string) string {
	hash, fault := AhdSecurityBcrypt(password)
	AhdRaiseFault(fault)
	return hash
}

func AhdSecurityBcryptVerify(password, encodedHash string) bool {
	matched, fault := AhdSecurityBcryptCheck(password, encodedHash)
	AhdRaiseFault(fault)
	return matched
}
