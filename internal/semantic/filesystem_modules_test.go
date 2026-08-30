package semantic

import (
	"reflect"
	"testing"

	"ahdcode/internal/types"
)

func TestFilesystemStandardModulesExposeExactSurface(t *testing.T) {
	modules := StandardModuleInterfaces()
	fileWant := []string{"FileError", "append", "createDir", "delete", "exists", "list", "readText", "writeText"}
	pathWant := []string{"base", "dir", "ext", "join"}
	if !reflect.DeepEqual(modules["File"].ExportNames, fileWant) {
		t.Fatalf("File exports = %v", modules["File"].ExportNames)
	}
	if !reflect.DeepEqual(modules["Path"].ExportNames, pathWant) {
		t.Fatalf("Path exports = %v", modules["Path"].ExportNames)
	}
	if symbol := modules["File"].Exports["list"]; symbol == nil || !types.Equal(symbol.Callable.Signature.Return, types.List{Element: types.String}) {
		t.Fatalf("File.list = %#v", symbol)
	}
	if fileErrorClass.Parent == nil || fileErrorClass.Parent.Name != "IOError" || fileErrorClass.Parent.Parent == nil || fileErrorClass.Parent.Parent.Name != "Error" {
		t.Fatalf("FileError hierarchy = %#v", fileErrorClass)
	}
}

func TestFilesystemCallsUseOrdinaryModuleTyping(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring File
bring Path
from File bring FileError

present: Bool := File.exists("x")
text: String := File.readText("x")
names: List<String> := File.list(".")
joined: String := Path.join(["a", "b"])
failure: FileError := FileError("failed")
`)
	if result.HasErrors() {
		t.Fatalf("filesystem program diagnostics = %#v", result.Diagnostics)
	}
	rejected := analyzeWithStandardModules(t, "bring File\nFile.writeText(1, \"x\")\n")
	if !rejected.HasErrors() {
		t.Fatal("File.writeText accepted a non-String path")
	}
}
