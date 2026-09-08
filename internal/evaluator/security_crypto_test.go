package evaluator

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
)

// Expected digests and MACs below are the published test vectors for the
// algorithms, so these tests fail if the implementation drifts to a different
// algorithm or encoding rather than merely comparing the code to itself.

func TestSecuritySHA256KnownVectors(t *testing.T) {
	session := newSecurityTestSession()
	cases := map[string]string{
		"":        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"abc":     "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		"AhdCode": "4addc0355935376654c6c1fc127dc452d1b12418805637c95aa1f1f3aa850d67",
	}
	for input, want := range cases {
		got := session.securityBuiltin("sha256", []any{input}).(string)
		if got != want {
			t.Fatalf("sha256(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestSecuritySHA512KnownVectors(t *testing.T) {
	session := newSecurityTestSession()
	cases := map[string]string{
		"": "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce" +
			"47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		"abc": "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a" +
			"2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f",
	}
	for input, want := range cases {
		got := session.securityBuiltin("sha512", []any{input}).(string)
		if got != want {
			t.Fatalf("sha512(%q) = %s, want %s", input, got, want)
		}
	}
}

// RFC 4231 test case 2 for HMAC-SHA256.
func TestSecurityHMACKnownVector(t *testing.T) {
	session := newSecurityTestSession()
	got := session.securityBuiltin("hmacSHA256", []any{"Jefe", "what do ya want for nothing?"}).(string)
	want := "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"
	if got != want {
		t.Fatalf("hmacSHA256 = %s, want %s", got, want)
	}
	if !session.securityBuiltin("hmacVerify", []any{"Jefe", "what do ya want for nothing?", want}).(bool) {
		t.Fatal("hmacVerify rejected a correct MAC")
	}
	if session.securityBuiltin("hmacVerify", []any{"Jefe", "what do ya want for nothing?", strings.Repeat("0", 64)}).(bool) {
		t.Fatal("hmacVerify accepted a wrong MAC")
	}
	// A different key must produce a different MAC.
	if session.securityBuiltin("hmacVerify", []any{"other", "what do ya want for nothing?", want}).(bool) {
		t.Fatal("hmacVerify accepted a MAC computed under a different key")
	}
}

// RFC 4648 vectors, plus the two characters that distinguish base64url from
// standard base64.
func TestSecurityEncodingKnownVectors(t *testing.T) {
	session := newSecurityTestSession()
	standard := map[string]string{
		"":       "",
		"f":      "Zg==",
		"fo":     "Zm8=",
		"foobar": "Zm9vYmFy",
	}
	for input, want := range standard {
		got := session.securityBuiltin("base64Encode", []any{input}).(string)
		if got != want {
			t.Fatalf("base64Encode(%q) = %q, want %q", input, got, want)
		}
		back := session.securityBuiltin("base64Decode", []any{got}).(string)
		if back != input {
			t.Fatalf("base64Decode(%q) = %q, want %q", got, back, input)
		}
	}

	// 0xFB 0xFF encodes to "+/" in standard base64 and "-_" in base64url,
	// and base64url is emitted without padding.
	raw := string([]byte{0xfb, 0xff})
	if got := session.securityBuiltin("base64Encode", []any{raw}).(string); got != "+/8=" {
		t.Fatalf("base64Encode(0xFBFF) = %q, want %q", got, "+/8=")
	}
	if got := session.securityBuiltin("base64UrlEncode", []any{raw}).(string); got != "-_8" {
		t.Fatalf("base64UrlEncode(0xFBFF) = %q, want %q", got, "-_8")
	}
	if got := session.securityBuiltin("base64UrlDecode", []any{"-_8"}).(string); got != raw {
		t.Fatal("base64UrlDecode did not round-trip")
	}
	// Padded base64url input is accepted, because other systems emit it.
	if got := session.securityBuiltin("base64UrlDecode", []any{"-_8="}).(string); got != raw {
		t.Fatal("base64UrlDecode rejected padded input")
	}

	if got := session.securityBuiltin("hexEncode", []any{"abc"}).(string); got != "616263" {
		t.Fatalf("hexEncode(\"abc\") = %q, want %q", got, "616263")
	}
	if got := session.securityBuiltin("hexDecode", []any{"616263"}).(string); got != "abc" {
		t.Fatalf("hexDecode round-trip failed: %q", got)
	}
}

func TestSecurityEncodingRejectsMalformedInput(t *testing.T) {
	session := newSecurityTestSession()
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("base64Decode", []any{"not valid base64!!"})
	})
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("hexDecode", []any{"zz"})
	})
}

func TestSecurityRandomHex(t *testing.T) {
	session := newSecurityTestSession()
	// Hex doubles the byte count.
	for _, count := range []int64{1, 16, 32} {
		got := session.securityBuiltin("randomHex", []any{count}).(string)
		if int64(len(got)) != count*2 {
			t.Fatalf("randomHex(%d) length = %d, want %d", count, len(got), count*2)
		}
		if strings.ToLower(got) != got {
			t.Fatalf("randomHex(%d) is not lowercase: %q", count, got)
		}
	}
	first := session.securityBuiltin("randomHex", []any{int64(32)}).(string)
	second := session.securityBuiltin("randomHex", []any{int64(32)}).(string)
	if first == second {
		t.Fatal("randomHex returned the same value twice")
	}
	for _, bad := range []int64{0, -1, 1025} {
		bad := bad
		expectEvaluatorRaise(t, "SecurityError", func() {
			session.securityBuiltin("randomHex", []any{bad})
		})
	}
}

func TestSecurityAESGCMRoundTrip(t *testing.T) {
	session := newSecurityTestSession()
	key := session.securityBuiltin("randomHex", []any{int64(32)}).(string)
	sealed := session.securityBuiltin("aesEncrypt", []any{key, "gizli mesaj"}).(string)
	if sealed == "gizli mesaj" {
		t.Fatal("aesEncrypt returned the plaintext")
	}
	opened := session.securityBuiltin("aesDecrypt", []any{key, sealed}).(string)
	if opened != "gizli mesaj" {
		t.Fatalf("aesDecrypt = %q, want %q", opened, "gizli mesaj")
	}

	// The nonce is random per call, so the same plaintext seals differently.
	again := session.securityBuiltin("aesEncrypt", []any{key, "gizli mesaj"}).(string)
	if again == sealed {
		t.Fatal("aesEncrypt produced an identical ciphertext twice; nonce is not random")
	}

	// A different key must not open it, and a tampered payload must fail
	// authentication rather than returning garbage plaintext.
	otherKey := session.securityBuiltin("randomHex", []any{int64(32)}).(string)
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("aesDecrypt", []any{otherKey, sealed})
	})
	tampered := "A" + sealed[1:]
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("aesDecrypt", []any{key, tampered})
	})
}

func TestSecurityAESRejectsWrongKeySize(t *testing.T) {
	session := newSecurityTestSession()
	// 16 bytes is a valid AES key size in general, but this API is AES-256 only.
	shortKey := strings.Repeat("ab", 16)
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("aesEncrypt", []any{shortKey, "text"})
	})
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("aesEncrypt", []any{"not-hex", "text"})
	})
}

// securityTestKeyPEM generates a real RSA key pair for the test. The signature
// produced by the evaluator is then checked with Go's own verifier rather than
// with the evaluator itself, so the test proves the output really is RS256 and
// not merely self-consistent.
func securityTestKeyPEM(t *testing.T) (privatePEM string, publicPEM string, key *rsa.PrivateKey) {
	t.Helper()
	generated, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("could not generate test key: %v", err)
	}
	privateBytes, err := x509.MarshalPKCS8PrivateKey(generated)
	if err != nil {
		t.Fatalf("could not marshal private key: %v", err)
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(&generated.PublicKey)
	if err != nil {
		t.Fatalf("could not marshal public key: %v", err)
	}
	privatePEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}))
	publicPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
	return privatePEM, publicPEM, generated
}

func TestSecurityRSASignAndVerify(t *testing.T) {
	session := newSecurityTestSession()
	privatePEM, publicPEM, key := securityTestKeyPEM(t)

	signature := session.securityBuiltin("rsaSignSHA256", []any{privatePEM, "imzalanacak"}).(string)
	if signature == "" {
		t.Fatal("rsaSignSHA256 returned an empty signature")
	}
	// The published contract is unpadded base64url: never +, / or =.
	if strings.ContainsAny(signature, "+/=") {
		t.Fatalf("signature is not unpadded base64url: %q", signature)
	}

	// Independent check: Go's own PKCS#1 v1.5 verifier must accept it.
	raw, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		t.Fatalf("signature is not valid base64url: %v", err)
	}
	digest := sha256.Sum256([]byte("imzalanacak"))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], raw); err != nil {
		t.Fatalf("signature rejected by Go's RS256 verifier: %v", err)
	}

	// The module's own verifier agrees, and rejects a different message.
	if !session.securityBuiltin("rsaVerifySHA256", []any{publicPEM, "imzalanacak", signature}).(bool) {
		t.Fatal("rsaVerifySHA256 rejected a valid signature")
	}
	if session.securityBuiltin("rsaVerifySHA256", []any{publicPEM, "baska metin", signature}).(bool) {
		t.Fatal("rsaVerifySHA256 accepted a signature over a different message")
	}

	// RSASSA-PKCS1-v1_5 is deterministic: the same input signs identically.
	again := session.securityBuiltin("rsaSignSHA256", []any{privatePEM, "imzalanacak"}).(string)
	if again != signature {
		t.Fatal("rsaSignSHA256 is not deterministic")
	}
}

func TestSecurityRSARejectsUnusableKeys(t *testing.T) {
	session := newSecurityTestSession()
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("rsaSignSHA256", []any{"not a pem", "message"})
	})
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("rsaVerifySHA256", []any{"not a pem", "message", "sig"})
	})
}
