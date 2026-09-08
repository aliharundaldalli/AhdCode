package semantic

import "testing"

const securityPreamble = "bring Security\nfrom Security bring SecurityError\n\n"

// The crypto surface added alongside the original password primitives must be
// visible to the semantic layer with exact types.
func TestSecurityModuleExportsCryptoSurface(t *testing.T) {
	modules := StandardModuleInterfaces()
	module, ok := modules["Security"]
	if !ok {
		t.Fatal("Security module not found in StandardModuleInterfaces")
	}
	exports := []string{
		// original surface
		"passwordHash", "passwordVerify", "token", "secureEqual", "SecurityError",
		// digests and MACs
		"sha256", "sha512", "hmacSHA256", "hmacVerify",
		// encodings
		"base64Encode", "base64Decode", "base64UrlEncode", "base64UrlDecode",
		"hexEncode", "hexDecode",
		// random material
		"randomHex",
		// signatures and symmetric encryption
		"rsaSignSHA256", "rsaVerifySHA256", "aesEncrypt", "aesDecrypt",
	}
	for _, name := range exports {
		if module.Exports[name] == nil {
			t.Fatalf("Security module missing export %q", name)
		}
	}
}

func TestSecurityCryptoValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, securityPreamble+`digest: String := Security.sha256("abc")
long: String := Security.sha512("abc")
mac: String := Security.hmacSHA256("key", "message")
macOk: Bool := Security.hmacVerify("key", "message", mac)
b64: String := Security.base64Encode("text")
plain: String := Security.base64Decode(b64)
url: String := Security.base64UrlEncode("text")
urlPlain: String := Security.base64UrlDecode(url)
hexed: String := Security.hexEncode("text")
unhexed: String := Security.hexDecode(hexed)
key: String := Security.randomHex(32)
signature: String := Security.rsaSignSHA256("pem", "message")
valid: Bool := Security.rsaVerifySHA256("pem", "message", signature)
sealed: String := Security.aesEncrypt(key, "secret")
opened: String := Security.aesDecrypt(key, sealed)
`)
	requireSemanticClean(t, result)
}

// hmacVerify and rsaVerifySHA256 answer Bool; the rest answer String.
func TestSecurityCryptoReturnTypes(t *testing.T) {
	failures := []string{
		`wrong: Bool := Security.sha256("abc")`,
		`wrong: String := Security.hmacVerify("k", "m", "mac")`,
		`wrong: String := Security.rsaVerifySHA256("pem", "m", "sig")`,
		`wrong: Bool := Security.aesEncrypt("key", "text")`,
		`wrong: Bool := Security.randomHex(16)`,
	}
	for _, source := range failures {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, securityPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSecurityCryptoRejectsWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`Security.sha256()`,
		`Security.sha256("a", "b")`,
		`Security.sha256(1)`,
		`Security.sha512()`,
		`Security.hmacSHA256("key")`,
		`Security.hmacVerify("key", "message")`,
		`Security.base64Encode()`,
		`Security.base64Decode(1)`,
		`Security.base64UrlEncode()`,
		`Security.hexEncode()`,
		`Security.hexDecode()`,
		// randomHex takes an Int count, not a String
		`Security.randomHex("32")`,
		`Security.randomHex()`,
		`Security.rsaSignSHA256("pem")`,
		`Security.rsaVerifySHA256("pem", "message")`,
		`Security.aesEncrypt("key")`,
		`Security.aesDecrypt("key")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, securityPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

// Malformed encodings and unusable keys raise SecurityError at run time, so it
// must remain catchable across the new surface too.
func TestSecurityCryptoErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, securityPreamble+`attempt {
    write(Security.hexDecode("not-hex"))
} except SecurityError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
