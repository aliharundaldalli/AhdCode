package ahdruntime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFileAndPathRuntimeLifecycle(t *testing.T) {
	directory := t.TempDir()
	nested := filepath.Join(directory, "nested")
	AhdFileCreateDir(AhdClassFileError, nested)
	path := AhdPathJoin(AhdNewList(nested, "note.txt"))
	AhdFileWriteText(AhdClassFileError, path, "Merhaba")
	AhdFileAppend(AhdClassFileError, path, " dünya")
	if !AhdFileExists(AhdClassFileError, path) || AhdFileReadText(AhdClassFileError, path) != "Merhaba dünya" {
		t.Fatal("File text lifecycle mismatch")
	}
	if names := AhdFileList(AhdClassFileError, nested).Snapshot(); !reflect.DeepEqual(names, []string{"note.txt"}) {
		t.Fatalf("File.list = %v", names)
	}
	if AhdPathExt(path) != ".txt" || AhdPathBase(path) != "note.txt" || AhdPathDir(path) != nested {
		t.Fatal("Path operations mismatch")
	}
	AhdFileDelete(AhdClassFileError, path)
	if AhdFileExists(AhdClassFileError, path) {
		t.Fatal("deleted file still exists")
	}
}

func TestFileRuntimeRaisesCatchableErrorAndRejectsInvalidUTF8(t *testing.T) {
	expectRaise(t, AhdClassFileError, func() { AhdFileReadText(AhdClassFileError, filepath.Join(t.TempDir(), "missing")) })
	path := filepath.Join(t.TempDir(), "bad.txt")
	if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	expectRaise(t, AhdClassFileError, func() { AhdFileReadText(AhdClassFileError, path) })
	if AhdClassFileError.Parent != AhdClassIOError || AhdClassIOError.Parent != AhdClassError {
		t.Fatal("FileError is not an IOError")
	}
}
