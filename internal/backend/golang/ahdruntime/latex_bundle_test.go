package ahdruntime

import "testing"

// Tectonic reads --bundle as a URL before it reads it as a path, so a Windows
// absolute path is parsed as the scheme "c" and rejected with "doesn't specify
// a valid bundle". These cases pin the shape that survives that parser on every
// platform; the Windows shapes are exercised from any development machine
// because the separator conversion is done by the caller.
func TestLatexBundleURL(t *testing.T) {
	cases := []struct{ slashed, want string }{
		// POSIX installations.
		{"/Users/ahd/go/libexec/ahdcode/latex/ahdcode-latex.ttb", "file:///Users/ahd/go/libexec/ahdcode/latex/ahdcode-latex.ttb"},
		{"/opt/ahdcode/libexec/ahdcode/latex/ahdcode-latex.ttb", "file:///opt/ahdcode/libexec/ahdcode/latex/ahdcode-latex.ttb"},
		// A Windows path, already converted to forward slashes by the caller.
		// The leading slash is what stops "C:" from parsing as a scheme.
		{"C:/Users/aliha/AppData/Local/AhdCode/versions/1.0.0-rc.1/libexec/ahdcode/latex/ahdcode-latex.ttb",
			"file:///C:/Users/aliha/AppData/Local/AhdCode/versions/1.0.0-rc.1/libexec/ahdcode/latex/ahdcode-latex.ttb"},
		// Characters that would otherwise change how the URL parses.
		{"C:/Users/Ali Harun/AhdCode/latex/ahdcode-latex.ttb",
			"file:///C:/Users/Ali%20Harun/AhdCode/latex/ahdcode-latex.ttb"},
		{"C:/Users/a#b/latex/ahdcode-latex.ttb", "file:///C:/Users/a%23b/latex/ahdcode-latex.ttb"},
		{"C:/Users/a?b/latex/ahdcode-latex.ttb", "file:///C:/Users/a%3Fb/latex/ahdcode-latex.ttb"},
		{"/tmp/ahd latex space test/zrd hbv/ahdcode-latex.ttb",
			"file:///tmp/ahd%20latex%20space%20test/zrd%20hbv/ahdcode-latex.ttb"},
	}
	for _, c := range cases {
		if got := ahdLatexBundleURL(c.slashed); got != c.want {
			t.Errorf("ahdLatexBundleURL(%q)\n got %q\nwant %q", c.slashed, got, c.want)
		}
	}
}

// Whatever the input, the argument handed to Tectonic must never begin with a
// bare drive letter, which is the form the engine rejects.
func TestLatexBundleURLNeverStartsWithADriveLetter(t *testing.T) {
	for _, slashed := range []string{
		"C:/x/ahdcode-latex.ttb",
		"Z:/deep/nested/path/ahdcode-latex.ttb",
		"/already/absolute/ahdcode-latex.ttb",
	} {
		got := ahdLatexBundleURL(slashed)
		if len(got) < 8 || got[:8] != "file:///" {
			t.Errorf("ahdLatexBundleURL(%q) = %q, want a file:/// URL", slashed, got)
		}
	}
	// ahdLatexBundleSpec is the runtime entry point and must agree on this host.
	if got := ahdLatexBundleSpec("/tmp/ahdcode-latex.ttb"); got != "file:///tmp/ahdcode-latex.ttb" {
		t.Errorf("ahdLatexBundleSpec = %q", got)
	}
}
