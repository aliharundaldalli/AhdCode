package evaluator

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
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
	case "sha256":
		sum := sha256.Sum256([]byte(args[0].(string)))
		return hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512([]byte(args[0].(string)))
		return hex.EncodeToString(sum[:])
	case "hmacSHA256":
		return session.securityHMAC(args[0].(string), args[1].(string))
	case "hmacVerify":
		expected := session.securityHMAC(args[0].(string), args[1].(string))
		return subtle.ConstantTimeCompare([]byte(expected), []byte(args[2].(string))) == 1
	case "base64Encode":
		return base64.StdEncoding.EncodeToString([]byte(args[0].(string)))
	case "base64Decode":
		decoded, err := base64.StdEncoding.DecodeString(args[0].(string))
		if err != nil {
			session.raise("SecurityError", "Security base64 input is malformed")
		}
		return string(decoded)
	case "base64UrlEncode":
		return base64.RawURLEncoding.EncodeToString([]byte(args[0].(string)))
	case "base64UrlDecode":
		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(args[0].(string), "="))
		if err != nil {
			session.raise("SecurityError", "Security base64url input is malformed")
		}
		return string(decoded)
	case "hexEncode":
		return hex.EncodeToString([]byte(args[0].(string)))
	case "hexDecode":
		decoded, err := hex.DecodeString(args[0].(string))
		if err != nil {
			session.raise("SecurityError", "Security hex input is malformed")
		}
		return string(decoded)
	case "randomHex":
		return session.securityRandomHex(args[0].(int64))
	case "rsaSignSHA256":
		return session.securityRSASign(args[0].(string), args[1].(string))
	case "rsaVerifySHA256":
		return session.securityRSAVerify(args[0].(string), args[1].(string), args[2].(string))
	case "aesEncrypt":
		return session.securityAESEncrypt(args[0].(string), args[1].(string))
	case "aesDecrypt":
		return session.securityAESDecrypt(args[0].(string), args[1].(string))
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

// The helpers below mirror ahdruntime/security.go exactly: same algorithms,
// same key sizes, same error messages. A program must behave identically in
// the REPL and after compilation.

func (session *Session) securityHMAC(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func (session *Session) securityRandomHex(count int64) string {
	if count < 1 || count > 1024 {
		session.raise("SecurityError", "Security random byte count must be between 1 and 1024")
	}
	buf := make([]byte, count)
	if _, err := rand.Read(buf); err != nil {
		session.raise("SecurityError", "Security random token generation failed")
	}
	return hex.EncodeToString(buf)
}

func (session *Session) securityRSAPrivateKey(pemText string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		session.raise("SecurityError", "Security private key is not valid PEM")
	}
	if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return parsed
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		session.raise("SecurityError", "Security private key could not be parsed")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		session.raise("SecurityError", "Security private key is not an RSA key")
	}
	return key
}

func (session *Session) securityRSASign(privateKeyPEM, message string) string {
	key := session.securityRSAPrivateKey(privateKeyPEM)
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		session.raise("SecurityError", "Security RSA signing failed")
	}
	return base64.RawURLEncoding.EncodeToString(signature)
}

func (session *Session) securityRSAVerify(publicKeyPEM, message, signatureBase64Url string) bool {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		session.raise("SecurityError", "Security public key is not valid PEM")
	}
	var key *rsa.PublicKey
	if parsed, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		key = parsed
	} else {
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			session.raise("SecurityError", "Security public key could not be parsed")
		}
		rsaKey, ok := parsed.(*rsa.PublicKey)
		if !ok {
			session.raise("SecurityError", "Security public key is not an RSA key")
		}
		key = rsaKey
	}
	signature, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(signatureBase64Url, "="))
	if err != nil {
		session.raise("SecurityError", "Security base64url input is malformed")
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) == nil
}

func (session *Session) securityAESKey(keyHex string) []byte {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		session.raise("SecurityError", "Security AES key must be hex encoded")
	}
	if len(key) != 32 {
		session.raise("SecurityError", "Security AES key must be 32 bytes (64 hex characters)")
	}
	return key
}

func (session *Session) securityAESEncrypt(keyHex, plaintext string) string {
	block, err := aes.NewCipher(session.securityAESKey(keyHex))
	if err != nil {
		session.raise("SecurityError", "Security AES cipher could not be created")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		session.raise("SecurityError", "Security AES-GCM could not be created")
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		session.raise("SecurityError", "Security random token generation failed")
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil))
}

func (session *Session) securityAESDecrypt(keyHex, payload string) string {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		session.raise("SecurityError", "Security base64 input is malformed")
	}
	block, err := aes.NewCipher(session.securityAESKey(keyHex))
	if err != nil {
		session.raise("SecurityError", "Security AES cipher could not be created")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		session.raise("SecurityError", "Security AES-GCM could not be created")
	}
	if len(raw) < gcm.NonceSize() {
		session.raise("SecurityError", "Security AES-GCM payload is too short")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		session.raise("SecurityError", "Security AES-GCM authentication failed")
	}
	return string(plaintext)
}
