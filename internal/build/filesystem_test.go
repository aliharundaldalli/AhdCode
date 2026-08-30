package build

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFileAndPathModulesBuildAndRunNatively(t *testing.T) {
	data := filepath.Join(t.TempDir(), "data")
	path := filepath.Join(data, "note.txt")
	missing := filepath.Join(data, "missing.txt")
	quote := strconv.Quote
	source := `bring File
bring Path
from File bring FileError

File.createDir(` + quote(data) + `)
File.writeText(` + quote(path) + `, "hello")
File.append(` + quote(path) + `, " world")
write(File.readText(` + quote(path) + `))
write(File.exists(` + quote(path) + `))
write(Path.base(` + quote(path) + `))
write(File.list(` + quote(data) + `))
attempt {
    File.readText(` + quote(missing) + `)
}
except FileError as error {
    write("caught")
}
attempt {
    File.readText(` + quote(missing) + `)
}
except IOError as error {
    write("io caught")
}
File.delete(` + quote(path) + `)
write(File.exists(` + quote(path) + `))
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || errorOutput != "" {
		t.Fatalf("filesystem program failed: code=%d stderr=%s", code, errorOutput)
	}
	want := "hello world\ntrue\nnote.txt\n[\"note.txt\"]\ncaught\nio caught\nfalse\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q\nsource:\n%s", out, want, strings.TrimSpace(source))
	}
}
