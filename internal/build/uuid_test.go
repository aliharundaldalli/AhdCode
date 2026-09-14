package build

import (
	"os"
	"path/filepath"
	"testing"
)

// uuidProgram exercises every UUID operation natively, including the RFC 9562
// test vectors and a documented runtime failure.
const uuidProgram = `bring UUID
from UUID bring (UUIDValue, UUIDError)

vector: UUIDValue := UUID.parse("017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
write(vector.string())
write(str(vector.version()) + " " + str(vector.isZero()))
random4: UUIDValue := UUID.parse("919108f7-52d1-4320-9bac-f847db4148a8")
write(str(random4.version()) + " " + str(random4.compare(vector)) + " " + str(vector.compare(random4)))
write(str(vector.equals(UUID.parse(vector.string()))) + " " + str(vector == UUID.parse(vector.string())))
empty: UUIDValue := UUID.zero()
write(empty.string() + " " + str(empty.isZero()) + " " + str(empty.version()))
write(str(UUID.parse("ffffffff-ffff-ffff-ffff-ffffffffffff").version()))
write(str(vector))

previous: UUIDValue := UUID.v7()
ordered: Bool := true
count: Int := 0
while count < 2000 {
    current: Local UUIDValue := UUID.v7()
    if current.compare(previous) <= 0 {
        ordered = false
    }
    previous = current
    count++
}
write("v7 " + str(ordered) + " " + str(previous.version()))
random: UUIDValue := UUID.v4()
write("v4 " + str(random.version()) + " " + str(len(random.string())) + " " + str(UUID.isValid(random.string())))
for text in ["", "017f22e2-79b0-7cc3-98c4-dc0c0c07398", "017f22e2779b0-7cc3-98c4-dc0c0c07398f", " 017f22e2-79b0-7cc3-98c4-dc0c0c07398f", "017f22e2-79b0-7cc3-98c4-dc0c0c07398g", "017f22e279b07cc398c4dc0c0c07398f", "urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f"] {
    write(str(UUID.isValid(text)))
}
attempt {
    UUID.parse("not-a-uuid")
} except UUIDError as error {
    write(error.message)
}
`

const uuidExpected = `017f22e2-79b0-7cc3-98c4-dc0c0c07398f
7 false
4 1 -1
true false
00000000-0000-0000-0000-000000000000 true 0
15
<UUIDValue>
v7 true 7
v4 4 36 true
false
false
false
false
false
false
false
UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
`

func TestUUIDRunsThroughNativeBackend(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": uuidProgram})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" {
		t.Fatalf("UUID program failed: code=%d stderr=%q", code, stderr)
	}
	if stdout != uuidExpected {
		t.Fatalf("UUID native output mismatch:\n got: %q\nwant: %q", stdout, uuidExpected)
	}
}

func TestUUIDUncaughtErrorsAreAhdCodeLevel(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring UUID\nwrite(UUID.parse(\"x\").string())\n"})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code == 0 {
		t.Fatalf("expected a failing exit code (stdout=%q)", stdout)
	}
	assertNoGoInternals(t, stderr)
	if !containsAll(stderr, "UUIDError", "UUID text must be 36 characters") {
		t.Fatalf("uncaught UUIDError not reported at AhdCode level: %q", stderr)
	}
}

// A sibling UUID.ahd cannot hijack the builtin module.
func TestUUIDBuiltinWinsOverSiblingFile(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"UUID.ahd": "zero: Function := () -> String {\n    return \"shadow\"\n}\n",
		"main.ahd": "bring UUID\nwrite(UUID.zero().string())\n",
	})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != "00000000-0000-0000-0000-000000000000\n" {
		t.Fatalf("sibling file shadowed the builtin: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

// UUID is standard library only: its workspace keeps the bare go.mod and
// vendors nothing.
func TestUUIDProgramWorkspaceIsNotVendored(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": "bring UUID\nwrite(UUID.v7().string())\n"})
	program := Compile(filepath.Join(directory, "main.ahd"))
	if program.HasErrors() {
		t.Fatalf("compilation failed:\n%s", diagnosticText(program.Diagnostics))
	}
	if program.Program.RequiresMySQL || program.Program.RequiresCodes {
		t.Fatal("a UUID program requires a vendored tree")
	}
	workspace, diagnostics := NewWorkspace(program.Program)
	if len(diagnostics) != 0 {
		t.Fatalf("workspace failed: %s", diagnosticText(diagnostics))
	}
	defer workspace.Close()
	goMod, err := os.ReadFile(filepath.Join(workspace.Directory, "go.mod"))
	if err != nil || string(goMod) != workspaceModule {
		t.Fatalf("UUID workspace go.mod = %q (%v)", goMod, err)
	}
	if _, err := os.Stat(filepath.Join(workspace.Directory, "vendor")); !os.IsNotExist(err) {
		t.Fatalf("UUID workspace has a vendor directory: %v", err)
	}
}
