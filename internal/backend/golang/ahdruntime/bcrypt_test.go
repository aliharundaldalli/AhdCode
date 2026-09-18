package ahdruntime

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// phpBcryptVector is the $2y$ example from PHP's password_verify
// documentation, produced by an implementation other than x/crypto.
const phpBcryptVector = "$2y$10$.vGA1O9wmRjrwAVXD98HNOgsNpDczlqm3Jq7KnEd1rVAGv3Fykk1a"

func requireNoSecret(t *testing.T, fault *AhdFault, secrets ...string) {
	t.Helper()
	if fault == nil {
		return
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(fault.Message, secret) {
			t.Fatalf("SecurityError message leaks %q: %s", secret, fault.Message)
		}
	}
	if strings.Contains(fault.Message, "crypto/bcrypt") || strings.Contains(fault.Message, "bcrypt:") {
		t.Fatalf("SecurityError message leaks a Go error: %s", fault.Message)
	}
}

func TestBcryptHashesAtCost12AndVerifies(t *testing.T) {
	hash, fault := AhdSecurityBcrypt("correct horse battery staple")
	requireNoFault(t, fault)
	if len(hash) != 60 || !strings.HasPrefix(hash, "$2a$12$") {
		t.Fatalf("hash = %q, want the 60-character $2a$12$ form", hash)
	}
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != 12 {
		t.Fatalf("cost = %d, %v", cost, err)
	}
	other, _ := AhdSecurityBcrypt("correct horse battery staple")
	if other == hash {
		t.Fatal("two hashes of one password share a salt")
	}
	for password, expected := range map[string]bool{
		"correct horse battery staple":  true,
		"correct horse battery staple ": false,
		"Correct horse battery staple":  false,
		"":                              false,
	} {
		matched, fault := AhdSecurityBcryptCheck(password, hash)
		requireNoFault(t, fault)
		if matched != expected {
			t.Errorf("verify(%q) = %v", password, matched)
		}
	}
}

func TestBcryptPasswordLengthIsExactAndNeverTruncated(t *testing.T) {
	empty, fault := AhdSecurityBcrypt("")
	requireNoFault(t, fault)
	if matched, _ := AhdSecurityBcryptCheck("", empty); !matched {
		t.Fatal("the empty password does not verify")
	}
	exact := strings.Repeat("a", 72)
	hash, fault := AhdSecurityBcrypt(exact)
	requireNoFault(t, fault)
	if matched, _ := AhdSecurityBcryptCheck(exact, hash); !matched {
		t.Fatal("a 72-byte password does not verify")
	}
	// 73 bytes, and 72 characters that are 144 UTF-8 bytes, are refused in
	// both directions, never shortened to a prefix that would still match.
	for _, long := range []string{exact + "a", strings.Repeat("ş", 37), strings.Repeat("ş", 72)} {
		_, fault = AhdSecurityBcrypt(long)
		requireFault(t, fault, "SecurityError", "at most 72 UTF-8 bytes")
		requireNoSecret(t, fault, long)
		_, fault = AhdSecurityBcryptCheck(long, hash)
		requireFault(t, fault, "SecurityError", "at most 72 UTF-8 bytes")
	}
	multibyte := strings.Repeat("ş", 36) // 72 bytes
	hash, fault = AhdSecurityBcrypt(multibyte)
	requireNoFault(t, fault)
	if matched, _ := AhdSecurityBcryptCheck(multibyte, hash); !matched {
		t.Fatal("a 72-byte UTF-8 password does not verify")
	}
}

func TestBcryptVerifiesTheTestedPrefixes(t *testing.T) {
	// $2y$ from PHP.
	if matched, fault := AhdSecurityBcryptCheck("rasmuslerdorf", phpBcryptVector); fault != nil || !matched {
		t.Fatalf("PHP $2y$ vector: %v %v", matched, fault)
	}
	if matched, _ := AhdSecurityBcryptCheck("rasmuslerdorF", phpBcryptVector); matched {
		t.Fatal("a wrong password matched the PHP vector")
	}
	// $2a$ (what bcryptHash writes) and $2b$ describe the same computation for
	// passwords of at most 72 bytes; x/crypto keeps the minor letter.
	hash, _ := AhdSecurityBcrypt("migrate me")
	for _, prefix := range []string{"$2a$", "$2b$", "$2y$"} {
		variant := prefix + hash[4:]
		if matched, fault := AhdSecurityBcryptCheck("migrate me", variant); fault != nil || !matched {
			t.Errorf("%s variant: %v %v", prefix, matched, fault)
		}
	}
}

func TestBcryptRejectsMalformedAndUnsupportedHashes(t *testing.T) {
	valid := phpBcryptVector
	malformed := []string{
		"", "not a hash", valid[:59], valid + "a", " " + valid[1:],
		"$2x$" + valid[4:], "$2z$" + valid[4:], "$2$10$" + valid[7:] + "a", "$3a$" + valid[4:], "$1$" + valid[3:],
		"$2y$1a$" + valid[7:], "$2y$10." + valid[7:], "$2y$10$" + valid[7:59] + "=", "$2y$10$" + valid[7:59] + "é"[:1],
		"$argon2id$v=19$m=65536,t=3,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g",
	}
	for _, encoded := range malformed {
		_, fault := AhdSecurityBcryptCheck("rasmuslerdorf", encoded)
		requireFault(t, fault, "SecurityError", "bcrypt hash is malformed or uses an unsupported format")
		requireNoSecret(t, fault, "rasmuslerdorf", encoded)
	}
	for _, cost := range []string{"00", "03", "17", "31", "99"} {
		_, fault := AhdSecurityBcryptCheck("rasmuslerdorf", "$2y$"+cost+valid[6:])
		requireFault(t, fault, "SecurityError", "bcrypt hash cost is outside the supported range 04..16")
	}
}

// TestBcryptAndArgon2idAreNeverAutoDetected keeps the two password hashes
// explicit: each verifier accepts only its own format, and Argon2id is
// unchanged.
func TestBcryptAndArgon2idAreNeverAutoDetected(t *testing.T) {
	argon := AhdSecurityPasswordHash(AhdClassSecurityError, "shared secret")
	if !strings.HasPrefix(argon, "$argon2id$v=19$m=65536,t=3,p=1$") {
		t.Fatalf("passwordHash changed format: %q", argon)
	}
	if !AhdSecurityPasswordVerify(AhdClassSecurityError, "shared secret", argon) {
		t.Fatal("Argon2id no longer verifies")
	}
	_, fault := AhdSecurityBcryptCheck("shared secret", argon)
	requireFault(t, fault, "SecurityError", "bcrypt hash is malformed")
	bcryptHash, _ := AhdSecurityBcrypt("shared secret")
	raised := captureRaise(t, func() { AhdSecurityPasswordVerify(AhdClassSecurityError, "shared secret", bcryptHash) })
	if raised == "" {
		t.Fatal("passwordVerify accepted a bcrypt hash")
	}
	if strings.Contains(raised, bcryptHash) || strings.Contains(raised, "shared secret") {
		t.Fatalf("passwordVerify error leaks input: %s", raised)
	}
}

// captureRaise runs body and returns the message of the AhdCode error it
// raises, or "" when it returns normally.
func captureRaise(t *testing.T, body func()) (message string) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			signal, ok := recovered.(*AhdSignal)
			if !ok {
				t.Fatalf("expected an AhdSignal; received %v", recovered)
			}
			if signal.Instance.AhdClassOf() != AhdClassSecurityError {
				t.Fatalf("expected SecurityError; received %s", signal.Instance.AhdClassOf().Name)
			}
			message = signal.Message
		}
	}()
	body()
	return ""
}
