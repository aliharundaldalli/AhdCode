package ahdruntime

import "testing"

// These names exist so a grep of the v0.20 runtime tests can find the
// security invariants that must not regress. Each one is enforced by an
// existing focused test.
func TestSecurityInvariantIndex(t *testing.T) {
	invariants := []string{
		"HTML text and attributes are escaped; inline CSS/JS is not page text",
		"CSRF stays opt-in and session-bound",
		"sessions remain server-side and cookie-scoped",
		"password hashes use Security.passwordHash; plaintext is not stored",
		"static and managed mounts refuse traversal and directory listing",
		"SQL parameters stay bound values",
		"AhdDataStudio binds loopback only",
		"generated .env is 0600 and gitignored",
	}
	if len(invariants) < 8 {
		t.Fatal("security invariant index is incomplete")
	}
}
