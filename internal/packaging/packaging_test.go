package packaging

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// fakeHelpers is a --helpers folder of stand-in helpers for a target.
func fakeHelpers(t *testing.T, windows bool) string {
	t.Helper()
	folder := t.TempDir()
	for _, name := range []string{"ahdgui", "ahdgraphics", "ahdplot", "ahdplotview", "ahdsqlite", "ahdnumeric"} {
		if windows {
			name += ".exe"
		}
		writeFile(t, filepath.Join(folder, name), "helper "+name, 0o755)
	}
	return folder
}

// project writes an entry module beside files that must never be packaged.
func project(t *testing.T, source string) string {
	t.Helper()
	folder := t.TempDir()
	writeFile(t, filepath.Join(folder, "app.ahd"), source, 0o644)
	writeFile(t, filepath.Join(folder, ".env"), "DB_PASSWORD=hunter2\n", 0o600)
	writeFile(t, filepath.Join(folder, "ledger.db"), "sqlite", 0o644)
	writeFile(t, filepath.Join(folder, "notes.txt"), "private", 0o644)
	return filepath.Join(folder, "app.ahd")
}

const guiSQLiteProgram = `bring GUI
bring SQLite
db := SQLite.open(":memory:")
window := GUI.window(title: "Ledger")
window.column().label("hello")
window.close()
write("ok")
`

func listTree(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			relative, _ := filepath.Rel(root, path)
			files = append(files, filepath.ToSlash(relative))
		}
		return err
	})
	sort.Strings(files)
	return files
}

func TestPackageHoldsExactlyWhatTheProgramNeeds(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the macOS layout is checked on macOS; other targets are checked by cross-packaging")
	}
	output := t.TempDir()
	result := Package(Options{Entry: project(t, guiSQLiteProgram), Name: "Kasa Defteri", Output: output, Helpers: fakeHelpers(t, false)})
	if result.Application == "" {
		t.Fatalf("diagnostics %+v", result.Diagnostics)
	}
	want := []string{"Contents/Info.plist", "Contents/MacOS/Kasa Defteri", "Contents/MacOS/ahdgui", "Contents/MacOS/ahdsqlite",
		"Contents/PkgInfo", "Contents/Resources/AppIcon.icns", "Contents/Resources/ahdcode-app.json", "Contents/Resources/app-icon.png"}
	if got := listTree(t, result.Application); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("package contents:\n%s", strings.Join(got, "\n"))
	}
	if strings.Join(result.Helpers, ",") != "ahdgui,ahdsqlite" {
		t.Fatalf("helpers %v", result.Helpers)
	}
	for _, executable := range []string{"Contents/MacOS/Kasa Defteri", "Contents/MacOS/ahdgui"} {
		if info, err := os.Stat(filepath.Join(result.Application, executable)); err != nil || info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("%s is not executable", executable)
		}
	}
	plist, _ := os.ReadFile(filepath.Join(result.Application, "Contents", "Info.plist"))
	for _, want := range []string{"<string>Kasa Defteri</string>", "<string>org.ahdcode.app.kasa-defteri</string>",
		"<key>CFBundleIconFile</key>\n\t<string>AppIcon</string>", "<key>LSUIElement</key>\n\t<true/>"} {
		if !bytes.Contains(plist, []byte(want)) {
			t.Fatalf("Info.plist lacks %q:\n%s", want, plist)
		}
	}
	var identity appIdentity
	data, _ := os.ReadFile(filepath.Join(result.Application, "Contents", "Resources", identityFile))
	if json.Unmarshal(data, &identity) != nil || identity.Name != "Kasa Defteri" || identity.Icon != iconFile {
		t.Fatalf("identity %s", data)
	}
	icns, _ := os.ReadFile(filepath.Join(result.Application, "Contents", "Resources", "AppIcon.icns"))
	if string(icns[:4]) != "icns" || int(binary.BigEndian.Uint32(icns[4:8])) != len(icns) {
		t.Fatal("AppIcon.icns is not an icns file")
	}
	// Packaging again replaces the application it made.
	again := Package(Options{Entry: project(t, guiSQLiteProgram), Name: "Kasa Defteri", Output: output, Helpers: fakeHelpers(t, false)})
	if again.Application != result.Application {
		t.Fatalf("repackaging: %+v", again.Diagnostics)
	}
	if entries, _ := os.ReadDir(output); len(entries) != 1 {
		t.Fatalf("the output folder holds %d entries; staging was left behind", len(entries))
	}
}

func TestCrossPackagesForWindowsAndLinux(t *testing.T) {
	for _, testCase := range []struct {
		target string
		want   []string
	}{
		{"windows-x64", []string{"Ledger/Ledger.exe", "Ledger/runtime/ahdcode-app.json", "Ledger/runtime/ahdgui.exe",
			"Ledger/runtime/ahdsqlite.exe", "Ledger/runtime/app-icon.png"}},
		{"linux-x64", []string{"Ledger/Ledger", "Ledger/runtime/ahdcode-app.json", "Ledger/runtime/ahdgui",
			"Ledger/runtime/ahdsqlite", "Ledger/runtime/app-icon.png"}},
	} {
		t.Run(testCase.target, func(t *testing.T) {
			windows := strings.HasPrefix(testCase.target, "windows")
			output := t.TempDir()
			entry := project(t, guiSQLiteProgram)
			withoutHelpers := Package(Options{Entry: entry, Name: "Ledger", Output: output, Target: testCase.target})
			if withoutHelpers.Application != "" || len(withoutHelpers.Diagnostics) == 0 {
				t.Fatalf("cross-packaging without helpers: %+v", withoutHelpers)
			}
			if testCase.target != HostTarget() && !strings.Contains(withoutHelpers.Diagnostics[0].Message, "needs --helpers") {
				t.Fatalf("cross-packaging without helpers did not require an explicit helper folder: %+v", withoutHelpers)
			}
			result := Package(Options{Entry: entry, Name: "Ledger", Output: output, Target: testCase.target, Helpers: fakeHelpers(t, windows)})
			if result.Application == "" {
				t.Fatalf("diagnostics %+v", result.Diagnostics)
			}
			names, contents := readArchive(t, result.Archive, windows)
			if strings.Join(names, "\n") != strings.Join(testCase.want, "\n") {
				t.Fatalf("archive contents:\n%s", strings.Join(names, "\n"))
			}
			program := contents[testCase.want[0]]
			if windows {
				checkWindowsExecutable(t, program)
			} else {
				file, err := elf.NewFile(bytes.NewReader(program))
				if err != nil || file.Machine != elf.EM_X86_64 {
					t.Fatalf("not a Linux x64 program: %v", err)
				}
			}
			// The archive is the application folder, byte for byte.
			for name, content := range contents {
				onDisk, _ := os.ReadFile(filepath.Join(output, filepath.FromSlash(name)))
				if !bytes.Equal(onDisk, content) {
					t.Fatalf("%s differs from the folder", name)
				}
			}
		})
	}
}

func readArchive(t *testing.T, path string, zipped bool) ([]string, map[string][]byte) {
	t.Helper()
	contents := map[string][]byte{}
	var names []string
	if zipped {
		reader, err := zip.OpenReader(path)
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		for _, file := range reader.File {
			handle, _ := file.Open()
			data, _ := io.ReadAll(handle)
			handle.Close()
			names = append(names, file.Name)
			contents[file.Name] = data
		}
	} else {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		compressed, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		reader := tar.NewReader(compressed)
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(reader)
			names = append(names, header.Name)
			contents[header.Name] = data
		}
	}
	sort.Strings(names)
	return names, contents
}

// checkWindowsExecutable reads the icon resource back out of a packaged
// Windows program and checks it is a GUI-subsystem executable.
func checkWindowsExecutable(t *testing.T, program []byte) {
	t.Helper()
	file, err := pe.NewFile(bytes.NewReader(program))
	if err != nil {
		t.Fatalf("not a Windows program: %v", err)
	}
	header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		t.Fatal("a desktop program must not open a console window")
	}
	section := file.Section(".rsrc")
	if section == nil {
		t.Fatal("the program has no resources")
	}
	data, err := section.Data()
	if err != nil {
		t.Fatal(err)
	}
	directory := header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE]
	base := directory.VirtualAddress
	if base != section.VirtualAddress {
		t.Fatalf("resource directory at %x, section at %x", base, section.VirtualAddress)
	}
	entries := int(binary.LittleEndian.Uint16(data[14:]))
	types := map[uint32]uint32{}
	for index := range entries {
		at := 16 + 8*index
		types[binary.LittleEndian.Uint32(data[at:])] = binary.LittleEndian.Uint32(data[at+4:]) &^ 0x80000000
	}
	iconOffset, hasIcons := types[3]
	groupOffset, hasGroup := types[14]
	if !hasIcons || !hasGroup {
		t.Fatalf("resource types %v", types)
	}
	if count := int(binary.LittleEndian.Uint16(data[iconOffset+14:])); count != len(windowsIconSizes) {
		t.Fatalf("%d icon images", count)
	}
	// Follow the group: id directory, language directory, data entry.
	idDirectory := binary.LittleEndian.Uint32(data[groupOffset+16+4:]) &^ 0x80000000
	dataEntry := binary.LittleEndian.Uint32(data[idDirectory+16+4:])
	address, size := binary.LittleEndian.Uint32(data[dataEntry:]), binary.LittleEndian.Uint32(data[dataEntry+4:])
	group := data[address-base : address-base+size]
	if binary.LittleEndian.Uint16(group[2:]) != 1 || int(binary.LittleEndian.Uint16(group[4:])) != len(windowsIconSizes) {
		t.Fatalf("group icon header %x", group[:6])
	}
	// The 256 pixel image is a PNG.
	iconDirectory := binary.LittleEndian.Uint32(data[iconOffset+16+8*uint32(len(windowsIconSizes)-1)+4:]) &^ 0x80000000
	iconEntry := binary.LittleEndian.Uint32(data[iconDirectory+16+4:])
	iconAddress := binary.LittleEndian.Uint32(data[iconEntry:]) - base
	config, err := png.DecodeConfig(bytes.NewReader(data[iconAddress:]))
	if err != nil || config.Width != 256 {
		t.Fatalf("the 256 pixel icon: %v %v", config, err)
	}
}

func TestPackagingRejectsBadInput(t *testing.T) {
	entry := project(t, "write(\"hi\")\n")
	square := filepath.Join(t.TempDir(), "wide.png")
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, image.NewNRGBA(image.Rect(0, 0, 64, 32)))
	writeFile(t, square, buffer.String(), 0o644)
	notPNG := filepath.Join(t.TempDir(), "icon.png")
	writeFile(t, notPNG, "GIF89a", 0o644)
	for _, testCase := range []struct {
		options Options
		reason  string
	}{
		{Options{Entry: "app.txt"}, "ending in .ahd"},
		{Options{Entry: entry, Name: "../evil"}, "cannot name an application"},
		{Options{Entry: entry, Name: ".hidden"}, "cannot name an application"},
		{Options{Entry: entry, Name: "a/b"}, "cannot name an application"},
		{Options{Entry: entry, Target: "amiga-68k"}, "unknown target"},
		{Options{Entry: entry, Icon: square}, "must be square"},
		{Options{Entry: entry, Icon: notPNG}, "must be a PNG"},
		{Options{Entry: entry, Icon: filepath.Join(t.TempDir(), "missing.png")}, "cannot be read"},
	} {
		testCase.options.Output = t.TempDir()
		result := Package(testCase.options)
		if result.Application != "" || len(result.Diagnostics) == 0 || !strings.Contains(result.Diagnostics[0].Message, testCase.reason) {
			t.Errorf("%+v: %+v", testCase.options, result.Diagnostics)
		}
	}
	// A folder that is not a packaged application is never replaced.
	output := t.TempDir()
	layout := layoutFor("hello", Targets[HostTarget()])
	writeFile(t, filepath.Join(output, layout.root, "precious.txt"), "keep", 0o644)
	result := Package(Options{Entry: entry, Name: "hello", Output: output})
	if result.Application != "" || !strings.Contains(result.Diagnostics[0].Message, "not an application made by ahdcode package") {
		t.Fatalf("overwrote a user folder: %+v", result)
	}
	if data, _ := os.ReadFile(filepath.Join(output, layout.root, "precious.txt")); string(data) != "keep" {
		t.Fatal("a user file was changed")
	}
}

func TestLeakGateRejectsForeignFiles(t *testing.T) {
	target := Targets["linux-x64"]
	layout := layoutFor("App", target)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "App"), "program", 0o755)
	writeFile(t, filepath.Join(root, "runtime", identityFile), `{"name":"App","icon":"app-icon.png"}`, 0o644)
	writeFile(t, filepath.Join(root, "runtime", iconFile), "png", 0o644)
	writeFile(t, filepath.Join(root, "runtime", "ahdgui"), "helper", 0o755)
	sets := []helperSet{{name: "ahdgui"}}
	if problems := leakGate(root, layout, sets, target); len(problems) != 0 {
		t.Fatalf("a clean application: %v", problems)
	}
	writeFile(t, filepath.Join(root, "runtime", ".env"), "KEY=1", 0o600)
	writeFile(t, filepath.Join(root, "runtime", "data.db"), "db", 0o644)
	writeFile(t, filepath.Join(root, "main.ahd"), "source", 0o644)
	writeFile(t, filepath.Join(root, "runtime", "ahdplot"), "unlisted helper", 0o755)
	writeFile(t, filepath.Join(root, "runtime", identityFile), `{"name":"App","password":"x", "api_key": "y"}`, 0o644)
	problems := strings.Join(leakGate(root, layout, sets, target), "\n")
	for _, want := range []string{".env must not be packaged", "data.db must not be packaged", "main.ahd must not be packaged",
		"ahdplot is not part of the application", "looks like a secret"} {
		if !strings.Contains(problems, want) {
			t.Fatalf("leak gate missed %q:\n%s", want, problems)
		}
	}
}

func TestIconsAndNames(t *testing.T) {
	if !bytes.Equal(DefaultIconPNG(), mustRead(t, filepath.Join("..", "..", "editors", "vscode", "images", "ahdcode-icon.png"))) {
		t.Fatal("internal/packaging/ahdcode-icon.png differs from the official icon")
	}
	_, img, err := loadIcon("")
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{16, 256, 1024} {
		if resized := resize(img, size); resized.Bounds().Dx() != size {
			t.Fatalf("resize %d", size)
		}
	}
	shaped := macOSShape(img)
	if shaped.NRGBAAt(10, 512).A != 0 || shaped.NRGBAAt(512, 512).A == 0 || shaped.NRGBAAt(105, 105).A != 0 {
		t.Fatal("the macOS shape")
	}
	small := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	small.Set(3, 3, color.NRGBA{255, 0, 0, 255})
	if icns := makeICNS(small); int(binary.BigEndian.Uint32(icns[4:8])) != len(icns) {
		t.Fatal("icns length")
	}
	for name, valid := range map[string]bool{"Ledger": true, "Kasa Defteri": true, "Müşteri-Takip_2.0": true, "": false,
		"a/b": false, "a\\b": false, ".app": false, " padded": false, "tab\tname": false, strings.Repeat("x", 65): false} {
		if ValidName(name) != valid {
			t.Errorf("ValidName(%q) = %v", name, !valid)
		}
	}
	if bundleIdentifier("Kasa Defteri 2") != "org.ahdcode.app.kasa-defteri-2" || bundleIdentifier("Ğ") != "org.ahdcode.app.application" {
		t.Fatalf("bundle identifiers %q %q", bundleIdentifier("Kasa Defteri 2"), bundleIdentifier("Ğ"))
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestPackagedProgramFindsOnlyItsOwnHelpers runs a packaged GUI program
// with the real ahdgui helper, headless, with every environment override
// pointing somewhere wrong: it must use the helper inside the application.
func TestPackagedProgramFindsOnlyItsOwnHelpers(t *testing.T) {
	name := "ahdgui"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	realGUI := filepath.Join(t.TempDir(), name)
	command := exec.Command("go", "build", "-o", realGUI, ".")
	command.Dir = filepath.Join("..", "..", "cmd", "ahdgui")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("building ahdgui: %v\n%s", err, output)
	}
	output := t.TempDir()
	source := `bring GUI
window := GUI.window(title: "Packaged")
form := window.column()
name := form.textInput()
window.wait()
write("typed {name.text()}")
`
	result := Package(Options{Entry: project(t, source), Name: "Packaged", Output: output, Helpers: filepath.Dir(realGUI)})
	if result.Application == "" {
		t.Fatalf("%+v", result.Diagnostics)
	}
	layout := layoutFor("Packaged", Targets[HostTarget()])
	program := filepath.Join(result.Application, layout.program, layout.executable)
	command = exec.Command(program)
	command.Dir = t.TempDir()
	command.Env = append(os.Environ(), "AHDCODE_GUI_HEADLESS=1", `AHDCODE_GUI_HEADLESS_EVENTS=[{"event":"type","widget":2,"text":"Ali"}]`,
		"AHDCODE_GUI_RUNTIME="+filepath.Join(t.TempDir(), "wrong", "ahdgui"), "PATH=")
	out, err := command.CombinedOutput()
	if err != nil || string(out) != "typed Ali\n" {
		t.Fatalf("packaged program: %v %q", err, out)
	}
	// Without its helper the application reports that, instead of looking
	// elsewhere.
	if err := os.Remove(filepath.Join(result.Application, layout.helpers, filepath.Base(realGUI))); err != nil {
		t.Fatal(err)
	}
	command = exec.Command(program)
	command.Env = append(os.Environ(), "AHDCODE_GUI_HEADLESS=1", "AHDCODE_GUI_RUNTIME="+realGUI)
	out, _ = command.CombinedOutput()
	if !strings.Contains(string(out), "missing from this application") {
		t.Fatalf("a packaged program used a helper from outside: %q", out)
	}
}
