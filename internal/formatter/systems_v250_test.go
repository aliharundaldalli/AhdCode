package formatter

import (
	"strings"
	"testing"
	"unicode"
)

// withoutSpace drops whitespace and commas: the canonical multiline call
// separates its arguments by newlines alone.
func withoutSpace(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == ',' {
			return -1
		}
		return r
	}, text)
}

// TestV250LongNamedSystemsCallsFormatCanonically guards the long-call
// formatting class for the v2.5.0 Process, Archive, File, and SQLite calls:
// every argument survives, the output is idempotent, and one-line and
// multiline spellings of the same call format identically.
func TestV250LongNamedSystemsCallsFormatCanonically(t *testing.T) {
	oneLine := `bring Archive
bring File
bring Process
bring SQLite
result:=Process.run(command:"/usr/bin/systemctl",args:["status","example.service","--no-pager","; touch SHOULD_NOT_EXIST"],timeoutSeconds:15,maxOutputBytes:1048576)
Archive.extract(archive:"/srv/uploads/release-2026-09-28.zip",destination:"/srv/apps/panel/releases/42",maxFiles:5000,maxBytes:1073741824)
entries:=File.walk(path:"/srv/apps/panel/releases/42/public/assets",maxEntries:100000)
File.symlink(target:"releases/42",link:"current.next")
db:=SQLite.open("panel.db")
db.backupTo("/srv/backups/panel/panel-2026-09-28.db")
`
	multiline := `bring Archive
bring File
bring Process
bring SQLite
result := Process.run(
    command: "/usr/bin/systemctl",
    args: ["status", "example.service", "--no-pager", "; touch SHOULD_NOT_EXIST"],
    timeoutSeconds: 15,
    maxOutputBytes: 1048576
)
Archive.extract(
    archive: "/srv/uploads/release-2026-09-28.zip",
    destination: "/srv/apps/panel/releases/42",
    maxFiles: 5000,
    maxBytes: 1073741824
)
entries := File.walk(path: "/srv/apps/panel/releases/42/public/assets", maxEntries: 100000)
File.symlink(target: "releases/42", link: "current.next")
db := SQLite.open("panel.db")
db.backupTo("/srv/backups/panel/panel-2026-09-28.db")
`
	first := formatText(t, oneLine)
	if withoutSpace(first) != withoutSpace(oneLine) {
		t.Fatalf("formatting changed the program's tokens:\n%s", first)
	}
	for _, wanted := range []string{
		`command: "/usr/bin/systemctl"`, `"; touch SHOULD_NOT_EXIST"`, `"--no-pager"`, "timeoutSeconds: 15", "maxOutputBytes: 1048576",
		`destination: "/srv/apps/panel/releases/42"`, "maxFiles: 5000", "maxBytes: 1073741824",
		"maxEntries: 100000", `target: "releases/42"`, `db.backupTo("/srv/backups/panel/panel-2026-09-28.db")`,
	} {
		if !strings.Contains(first, wanted) {
			t.Fatalf("formatted output lacks %q:\n%s", wanted, first)
		}
	}
	if second := formatText(t, first); second != first {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if fromMultiline := formatText(t, multiline); fromMultiline != first {
		t.Fatalf("one-line and multiline spellings format differently:\none-line:\n%s\nmultiline:\n%s", first, fromMultiline)
	}
	for _, line := range strings.Split(first, "\n") {
		if len([]rune(line)) > 100 && !strings.Contains(line, `"`) {
			t.Fatalf("an unbreakable-free line exceeds the width: %q", line)
		}
	}
}
