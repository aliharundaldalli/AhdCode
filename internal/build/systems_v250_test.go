package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// systemsFixtures writes the release archives the v2.5.0 parity program reads.
func systemsFixtures(t *testing.T, directory string) {
	t.Helper()
	files := []struct {
		name, body string
		mode       int64
	}{
		{"app/bin/server", "\x7fELF-server-\x00\xff", 0o755},
		{"app/public/index.html", "<h1>Merhaba</h1>", 0o644},
		{"app/config.json", `{"port": 80}`, 0o644},
	}
	var zipped bytes.Buffer
	zipWriter := zip.NewWriter(&zipped)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.name, Method: zip.Deflate}
		header.SetMode(os.FileMode(file.mode))
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(file.body))
	}
	_ = zipWriter.Close()
	if err := os.WriteFile(filepath.Join(directory, "release.zip"), zipped.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var tarred bytes.Buffer
	compressor := gzip.NewWriter(&tarred)
	tarWriter := tar.NewWriter(compressor)
	for _, file := range files {
		_ = tarWriter.WriteHeader(&tar.Header{Name: "./" + file.name, Mode: file.mode, Size: int64(len(file.body)), Typeflag: tar.TypeReg})
		_, _ = tarWriter.Write([]byte(file.body))
	}
	_ = tarWriter.Close()
	_ = compressor.Close()
	if err := os.WriteFile(filepath.Join(directory, "release.tar.gz"), tarred.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	var evil bytes.Buffer
	evilWriter := zip.NewWriter(&evil)
	writer, _ := evilWriter.Create("../escaped.txt")
	_, _ = writer.Write([]byte("escaped"))
	_ = evilWriter.Close()
	if err := os.WriteFile(filepath.Join(directory, "evil.zip"), evil.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func systemsParitySource(echo, falseProgram string) string {
	return `bring Archive
bring File
bring Process
bring SQLite
from Archive bring (ArchiveEntry, ArchiveError)
from File bring (FileEntry, FileError)
from Process bring (ProcessResult, ProcessError)
from SQLite bring (Database, SQLiteError)

entries: List<ArchiveEntry> := Archive.list("release.zip")
for entry in entries {
    write(entry.path() + " " + entry.kind() + " " + str(entry.size()))
}
File.createDir("releases")
Archive.extract(archive: "release.zip", destination: "releases/42", maxFiles: 100, maxBytes: 1048576)
Archive.extract("release.tar.gz", "releases/43")
attempt {
    Archive.extract("evil.zip", "releases/evil")
} except ArchiveError as error {
    write("rejected: " + error.message)
}
write("evil exists: " + str(File.exists("releases/evil")) + " " + str(File.exists("escaped.txt")))
attempt {
    Archive.extract("release.zip", "releases/42")
} except ArchiveError as error {
    write(error.message)
}

walked: List<FileEntry> := File.walk("releases/42")
for item in walked {
    write(item.relativePath() + " " + item.kind() + " " + str(item.size()) + " " + str(item.isSymlink()))
}
write(walked[0].path())
File.copy("releases/42/app/bin/server", "server.backup")
attempt {
    File.copy("releases/42/app/bin/server", "server.backup")
} except FileError as error {
    write(error.message)
}
File.atomicWrite("releases/42/app/config.json", r'{"port": 8080}')
write(File.readText("releases/42/app/config.json"))
File.setPermissions("releases/42/app/bin/server", "0750")
write(File.permissions("releases/42/app/bin/server"))
attempt {
    File.setPermissions("releases/42/app/bin/server", "0o755")
} except FileError as error {
    write(error.message)
}
File.symlink(target: "releases/42", link: "current.next")
File.atomicMove("current.next", "current")
write(File.readLink("current") + " " + str(File.isSymlink("current")))
File.symlink("releases/43", "current.next")
File.atomicMove(source: "current.next", destination: "current")
write(File.readLink("current"))

result: ProcessResult := Process.run(
    command: ` + strconv.Quote(echo) + `,
    args: ["; touch SHOULD_NOT_EXIST", "$(touch SHOULD_NOT_EXIST)"],
    timeoutSeconds: 10,
    maxOutputBytes: 4096
)
write(str(result.exitCode()) + "|" + result.stdout() + "|" + result.stderr())
write("shell ran: " + str(File.exists("SHOULD_NOT_EXIST")))
failed: ProcessResult := Process.run(` + strconv.Quote(falseProgram) + `)
write("false exit: " + str(failed.exitCode()))
attempt {
    Process.run("ahd-no-such-program-v250", [])
} except ProcessError as error {
    write(error.message)
}

db: Database := SQLite.open("panel.db")
db.execute("CREATE TABLE apps (name TEXT NOT NULL)")
db.execute("INSERT INTO apps (name) VALUES ('ata')")
db.backupTo("panel-backup.db")
db.execute("INSERT INTO apps (name) VALUES ('after')")
attempt {
    db.backupTo("panel-backup.db")
} except SQLiteError as error {
    write(error.message)
}
snapshot: Database := SQLite.open("panel-backup.db")
rows := snapshot.query("SELECT count(*) AS n FROM apps")
write("backup rows: " + str(rows[0]["n"].int()))
snapshot.close()
db.close()
`
}

// TestSystemsPrimitivesMatchBetweenEvaluatorAndNative runs one program that
// uses every v2.5.0 primitive natively and in the evaluator, each in its own
// fresh directory, and requires byte-identical output.
func TestSystemsPrimitivesMatchBetweenEvaluatorAndNative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the parity program uses Unix symbolic links, permissions, echo, and false")
	}
	echo, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo not available")
	}
	falseProgram, err := exec.LookPath("false")
	if err != nil {
		t.Skip("false not available")
	}
	sqliteHelperForTest(t)
	source := systemsParitySource(echo, falseProgram)

	nativeDirectory := t.TempDir()
	systemsFixtures(t, nativeDirectory)
	entry := filepath.Join(t.TempDir(), "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	native, stderr, code := buildAndRunIn(t, entry, nativeDirectory)
	if code != 0 || stderr != "" {
		t.Fatalf("native exit %d, stderr %q, stdout:\n%s", code, stderr, native)
	}

	evaluatorDirectory := t.TempDir()
	systemsFixtures(t, evaluatorDirectory)
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, evaluatorDirectory, source, &output, &errorOutput)
	if output.String() != native {
		t.Fatalf("evaluator and native output differ\nnative:\n%s\nevaluator:\n%s\nstderr: %s", native, output.String(), errorOutput.String())
	}

	for _, want := range []string{
		"app/bin/server file 14\n",
		`rejected: extract "evil.zip" failed: unsafe entry path "../escaped.txt": the path escapes the destination`,
		"evil exists: false false\n",
		`destination "releases/42" already exists`,
		"app/bin/server file 14 false\n",
		"releases/42/app\n",
		"destination already exists; File.copy never overwrites",
		"{\"port\": 8080}\n0750\n",
		`mode "0o755" is not three octal digits`,
		"releases/42 true\nreleases/43\n",
		"0|; touch SHOULD_NOT_EXIST $(touch SHOULD_NOT_EXIST)\n|\n",
		"shell ran: false\n",
		"false exit: 1\n",
		`run "ahd-no-such-program-v250" failed: executable not found`,
		"the backup destination already exists; backupTo never overwrites",
		"backup rows: 1\n",
	} {
		if !strings.Contains(native, want) {
			t.Fatalf("output lacks %q:\n%s", want, native)
		}
	}
	for _, directory := range []string{nativeDirectory, evaluatorDirectory} {
		if _, err := os.Stat(filepath.Join(directory, "SHOULD_NOT_EXIST")); err == nil {
			t.Fatal("a shell interpreted a Process argument")
		}
		if content, err := os.ReadFile(filepath.Join(directory, "server.backup")); err != nil || string(content) != "\x7fELF-server-\x00\xff" {
			t.Fatalf("binary copy is not byte-exact: %q %v", content, err)
		}
	}
}

// TestProcessBoundsAreCatchableErrorsInBothBackends proves a timeout and an
// output overflow surface as ProcessError (an Error) in the evaluator and in
// a native program alike, and that neither leaves the program hanging.
func TestProcessBoundsAreCatchableErrorsInBothBackends(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses Unix sleep and yes")
	}
	sleep, sleepErr := exec.LookPath("sleep")
	yes, yesErr := exec.LookPath("yes")
	if sleepErr != nil || yesErr != nil {
		t.Skip("sleep or yes not available")
	}
	source := `bring Process
from Process bring ProcessError

attempt {
    Process.run(command: ` + strconv.Quote(sleep) + `, args: ["30"], timeoutSeconds: 1)
} except Error as failure {
    write(failure.message)
}
attempt {
    Process.run(command: ` + strconv.Quote(yes) + `, args: ["y"], maxOutputBytes: 100)
} except ProcessError as failure {
    write(failure.message)
}
attempt {
    Process.run(command: ` + strconv.Quote(yes) + `, timeoutSeconds: 0)
} except ProcessError as failure {
    write(failure.message)
}
`
	want := "run " + strconv.Quote(sleep) + " failed: timed out after 1 seconds (timeoutSeconds); the process was terminated\n" +
		"run " + strconv.Quote(yes) + " failed: output limit exceeded: stdout and stderr together produced more than 100 bytes (maxOutputBytes); the process was terminated\n" +
		"run " + strconv.Quote(yes) + " failed: timeoutSeconds must be between 1 and 3600; received 0\n"
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	native, stderr, code := buildAndRunIn(t, entry, directory)
	if code != 0 || native != want {
		t.Fatalf("native exit %d stderr %q:\n%s", code, stderr, native)
	}
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, directory, source, &output, &errorOutput)
	if output.String() != want {
		t.Fatalf("evaluator:\n%s\nstderr %s", output.String(), errorOutput.String())
	}
}

// TestV250ExamplesCompile keeps examples/v2.5 compiling with the release.
func TestV250ExamplesCompile(t *testing.T) {
	for _, name := range []string{"archive_extract", "filesystem_deploy", "process_safe", "sqlite_backup"} {
		entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v2.5", name, "main.ahd"))
		if err != nil {
			t.Fatal(err)
		}
		if _, result := BuildProgram(entry, filepath.Join(t.TempDir(), name)); result.HasErrors() {
			t.Fatalf("%s does not compile:\n%s", name, diagnosticText(result.Diagnostics))
		}
	}
}
