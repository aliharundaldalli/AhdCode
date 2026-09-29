package formatter

import (
	"strings"
	"testing"
)

// TestV270InspectionCallsFormatCanonically covers the Disk and Service calls:
// tokens survive, output is idempotent, and spellings format identically.
func TestV270InspectionCallsFormatCanonically(t *testing.T) {
	oneLine := `bring Disk
bring Service
disk:=Disk.inspect(path:"/srv/ahdcode.org/releases/current/public/uploads/customer-archive-2026")
unit:=Service.status(name:"ahdcode-org-production-worker-control-channel@primary.service")
short:=Disk.inspect("/")
`
	multiline := `bring Disk
bring Service
disk := Disk.inspect(
    path: "/srv/ahdcode.org/releases/current/public/uploads/customer-archive-2026"
)
unit := Service.status(
    name: "ahdcode-org-production-worker-control-channel@primary.service"
)
short := Disk.inspect("/")
`
	first := formatText(t, oneLine)
	if withoutSpace(first) != withoutSpace(oneLine) {
		t.Fatalf("formatting changed the program's tokens:\n%s", first)
	}
	if !strings.Contains(first, `short := Disk.inspect("/")`) {
		t.Fatalf("short call not kept on one line:\n%s", first)
	}
	if second := formatText(t, first); second != first {
		t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if fromMultiline := formatText(t, multiline); fromMultiline != first {
		t.Fatalf("spellings differ:\none-line:\n%s\nmultiline:\n%s", first, fromMultiline)
	}
}
