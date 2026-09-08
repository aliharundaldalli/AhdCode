package golang

import (
	"strings"
	"testing"
)

// Each new Security function must lower to its own runtime helper. A missing
// case would fall through to generator.unsupported and silently change
// meaning, so the mapping is asserted call by call.
func TestSecurityCryptoLoweringEmitsRuntimeCalls(t *testing.T) {
	source := programSource(t, generate(t, `bring Security
write(Security.sha256("abc"))
write(Security.sha512("abc"))
write(Security.hmacSHA256("key", "message"))
write(str(Security.hmacVerify("key", "message", "mac")))
write(Security.base64Encode("text"))
write(Security.base64Decode("dGV4dA=="))
write(Security.base64UrlEncode("text"))
write(Security.base64UrlDecode("dGV4dA"))
write(Security.hexEncode("text"))
write(Security.hexDecode("74657874"))
write(Security.randomHex(32))
write(Security.rsaSignSHA256("pem", "message"))
write(str(Security.rsaVerifySHA256("pem", "message", "sig")))
write(Security.aesEncrypt("key", "text"))
write(Security.aesDecrypt("key", "payload"))
`))
	expected := []string{
		"AhdSecuritySHA256(",
		"AhdSecuritySHA512(",
		"AhdSecurityHMACSHA256(",
		"AhdSecurityHMACVerify(",
		"AhdSecurityBase64Encode(",
		"AhdSecurityBase64Decode(",
		"AhdSecurityBase64UrlEncode(",
		"AhdSecurityBase64UrlDecode(",
		"AhdSecurityHexEncode(",
		"AhdSecurityHexDecode(",
		"AhdSecurityRandomHex(",
		"AhdSecurityRSASignSHA256(",
		"AhdSecurityRSAVerifySHA256(",
		"AhdSecurityAESGCMEncrypt(",
		"AhdSecurityAESGCMDecrypt(",
	}
	for _, call := range expected {
		if !strings.Contains(source, call) {
			t.Fatalf("generated program does not call %s", call)
		}
	}
	// Functions that can fail on malformed input receive the error class;
	// pure encoders do not.
	if !strings.Contains(source, "AhdSecurityHexDecode(cd_SecurityError") {
		t.Fatal("hexDecode lowering does not pass the Security error class")
	}
	if strings.Contains(source, "AhdSecuritySHA256(cd_SecurityError") {
		t.Fatal("sha256 lowering should not take an error class")
	}
	// randomHex takes an Int, so the count must be lowered as int64 rather
	// than as a string. The descriptor name carries a hash suffix, so the
	// call is matched in two parts.
	if !strings.Contains(source, "AhdSecurityRandomHex(cd_SecurityError") {
		t.Fatal("randomHex lowering does not pass the Security error class")
	}
	if !strings.Contains(source, "int64(32))") {
		t.Fatal("randomHex lowering does not pass an int64 count")
	}
}

// The original password primitives must keep lowering exactly as before.
func TestSecurityPasswordLoweringUnchanged(t *testing.T) {
	source := programSource(t, generate(t, `bring Security
write(Security.passwordHash("pw"))
write(str(Security.passwordVerify("pw", "hash")))
write(Security.token())
write(str(Security.secureEqual("a", "b")))
`))
	for _, call := range []string{
		"AhdSecurityPasswordHash(cd_SecurityError",
		"AhdSecurityPasswordVerify(cd_SecurityError",
		"AhdSecurityToken(cd_SecurityError",
		"AhdSecuritySecureEqual(",
	} {
		if !strings.Contains(source, call) {
			t.Fatalf("generated program does not call %s", call)
		}
	}
}
