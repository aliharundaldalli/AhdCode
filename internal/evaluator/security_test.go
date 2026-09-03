package evaluator

import (
	"strings"
	"testing"

	"ahdcode/internal/semantic"
)

const securityTestPassword = "AHD_SECURITY_QA_PASSWORD_9f31_secret"

func newSecurityTestSession() *Session {
	return newLatexTestSession()
}

// A. Argon2id hash + verify (correct password → true)
func TestSecurityPasswordHashAndVerify(t *testing.T) {
	session := newSecurityTestSession()
	hash := session.securityBuiltin("passwordHash", []any{securityTestPassword}).(string)
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=1$") {
		t.Fatalf("unexpected PHC prefix: %s", hash[:min64(len(hash), 50)])
	}
	ok := session.securityBuiltin("passwordVerify", []any{securityTestPassword, hash}).(bool)
	if !ok {
		t.Fatal("passwordVerify returned false for correct password")
	}
	wrong := session.securityBuiltin("passwordVerify", []any{"definitely-wrong-password", hash}).(bool)
	if wrong {
		t.Fatal("passwordVerify returned true for wrong password")
	}
}

// B. Random salt (same password twice → different hashes)
func TestSecurityRandomSalt(t *testing.T) {
	session := newSecurityTestSession()
	h1 := session.securityBuiltin("passwordHash", []any{securityTestPassword}).(string)
	h2 := session.securityBuiltin("passwordHash", []any{securityTestPassword}).(string)
	if h1 == h2 {
		t.Fatal("two hashes of the same password are identical (salt is not random)")
	}
}

// C. Malformed PHC → SecurityError
func TestSecurityMalformedPHC(t *testing.T) {
	session := newSecurityTestSession()
	cases := []string{
		"not-a-phc",
		"$argon2id$",
		"$argon2id$v=19$m=65536,t=3,p=1$badbas64!!!$badhash",
		"$argon2id$v=19$m=65536,t=3$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	for _, bad := range cases {
		bad := bad
		expectEvaluatorRaise(t, "SecurityError", func() {
			session.securityBuiltin("passwordVerify", []any{securityTestPassword, bad})
		})
	}
}

// D. Unsafe PHC parameters → SecurityError BEFORE expensive work
func TestSecurityUnsafeParameters(t *testing.T) {
	session := newSecurityTestSession()
	// memory too low (< 8192)
	lowMem := "$argon2id$v=19$m=4096,t=3,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("passwordVerify", []any{securityTestPassword, lowMem})
	})
	// time too high (> 10)
	highTime := "$argon2id$v=19$m=65536,t=11,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	expectEvaluatorRaise(t, "SecurityError", func() {
		session.securityBuiltin("passwordVerify", []any{securityTestPassword, highTime})
	})
}

// E. Token generation (correct length, URL-safe chars, uniqueness)
func TestSecurityToken(t *testing.T) {
	session := newSecurityTestSession()
	tok := session.securityBuiltin("token", []any{}).(string)
	if len(tok) != 43 {
		t.Fatalf("token length = %d, want 43", len(tok))
	}
	for _, ch := range tok {
		if !isURLSafe(ch) {
			t.Fatalf("token contains non-URL-safe character: %q", ch)
		}
	}
	tok2 := session.securityBuiltin("token", []any{}).(string)
	if tok == tok2 {
		t.Fatal("two successive tokens are identical")
	}
}

func isURLSafe(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9') || r == '-' || r == '_'
}

// F. secureEqual true/false semantics
func TestSecuritySecureEqual(t *testing.T) {
	session := newSecurityTestSession()
	tok := session.securityBuiltin("token", []any{}).(string)
	if !session.securityBuiltin("secureEqual", []any{tok, tok}).(bool) {
		t.Fatal("secureEqual(tok, tok) returned false")
	}
	tok2 := session.securityBuiltin("token", []any{}).(string)
	if session.securityBuiltin("secureEqual", []any{tok, tok2}).(bool) {
		t.Fatal("secureEqual(tok, tok2) returned true for different tokens")
	}
	if session.securityBuiltin("secureEqual", []any{"short", "longer-than-short"}).(bool) {
		t.Fatal("secureEqual returned true for strings of different lengths")
	}
}

// G. Module integration — Security appears in StandardModuleInterfaces
func TestSecurityModuleRegistered(t *testing.T) {
	modules := semantic.StandardModuleInterfaces()
	sec, ok := modules["Security"]
	if !ok {
		t.Fatal("Security module not found in StandardModuleInterfaces")
	}
	for _, name := range []string{"passwordHash", "passwordVerify", "token", "secureEqual", "SecurityError"} {
		if sec.Exports[name] == nil {
			t.Fatalf("Security module missing export %q", name)
		}
	}
}

func min64(a, b int) int {
	if a < b {
		return a
	}
	return b
}
