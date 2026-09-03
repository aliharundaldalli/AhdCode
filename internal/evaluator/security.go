package evaluator

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// The Security standard module's REPL implementation. It mirrors the native
// backend's ahdruntime security functions exactly: same PHC format, same
// Argon2id parameters, same bounds checks, same error messages.

const (
	securityArgon2Memory      = 65536 // KiB
	securityArgon2Time        = 3
	securityArgon2Parallelism = 1
	securityArgon2SaltLen     = 16
	securityArgon2KeyLen      = 32
	securityArgon2Version     = 19

	securityMaxPasswordBytes = 1 << 20 // 1 MiB
)

func (session *Session) securityBuiltin(name string, args []any) any {
	switch name {
	case "passwordHash":
		password := args[0].(string)
		return session.securityPasswordHash(password)
	case "passwordVerify":
		password := args[0].(string)
		encodedHash := args[1].(string)
		return session.securityPasswordVerify(password, encodedHash)
	case "token":
		return session.securityToken()
	case "secureEqual":
		expected := args[0].(string)
		received := args[1].(string)
		return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
	}
	session.raise("Error", "unsupported Security function "+name)
	return nil
}

func (session *Session) securityPasswordHash(password string) string {
	if len(password) > securityMaxPasswordBytes {
		session.raise("SecurityError", "Security password input is too large")
	}
	salt := make([]byte, securityArgon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		session.raise("SecurityError", "Security random token generation failed")
	}
	hash := argon2.IDKey([]byte(password), salt,
		securityArgon2Time, securityArgon2Memory, uint8(securityArgon2Parallelism), securityArgon2KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		securityArgon2Version,
		securityArgon2Memory, securityArgon2Time, securityArgon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func (session *Session) securityPasswordVerify(password, encodedHash string) bool {
	memory, time, parallelism, salt, storedHash, errMsg := securityPHCDecode(encodedHash)
	if errMsg != "" {
		session.raise("SecurityError", errMsg)
	}
	candidate := argon2.IDKey([]byte(password), salt, time, memory, uint8(parallelism), uint32(len(storedHash)))
	return subtle.ConstantTimeCompare(candidate, storedHash) == 1
}

func (session *Session) securityToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		session.raise("SecurityError", "Security random token generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// securityPHCDecode parses a PHC string and validates its parameters.
func securityPHCDecode(encoded string) (memory, time, parallelism uint32, salt, hash []byte, errMsg string) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	if parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, "Security password hash uses an unsupported algorithm"
	}
	if !strings.HasPrefix(parts[2], "v=") {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	version, err := strconv.ParseUint(parts[2][2:], 10, 32)
	if err != nil || version != securityArgon2Version {
		return 0, 0, 0, nil, nil, "Security password hash uses an unsupported algorithm"
	}
	paramMap := make(map[string]uint64)
	for _, kv := range strings.Split(parts[3], ",") {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			return 0, 0, 0, nil, nil, "Security password hash is malformed"
		}
		val, perr := strconv.ParseUint(kv[eq+1:], 10, 32)
		if perr != nil {
			return 0, 0, 0, nil, nil, "Security password hash is malformed"
		}
		paramMap[kv[:eq]] = val
	}
	mVal, mOk := paramMap["m"]
	tVal, tOk := paramMap["t"]
	pVal, pOk := paramMap["p"]
	if !mOk || !tOk || !pOk {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	memory = uint32(mVal)
	time = uint32(tVal)
	parallelism = uint32(pVal)
	if memory < 8192 || memory > 262144 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if time < 1 || time > 10 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if parallelism < 1 || parallelism > 16 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	salt, serr := base64.RawStdEncoding.DecodeString(parts[4])
	if serr != nil {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	hash, herr := base64.RawStdEncoding.DecodeString(parts[5])
	if herr != nil {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	if len(salt) < 8 || len(salt) > 64 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if len(hash) < 16 || len(hash) > 64 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	return memory, time, parallelism, salt, hash, ""
}
