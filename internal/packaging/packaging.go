// Package packaging implements `ahdcode package`: it turns one AhdCode entry
// module into a self-contained desktop application that runs without a
// separate AhdCode installation.
//
// An application holds only what its program needs: the compiled program,
// the bundled helpers its compile graph requires (ahdgui for GUI, ahdgraphics
// for Graphics, ahdplot, ahdplotview, and ahdgui for Plot, ahdsqlite for
// SQLite, ahdnumeric for Numeric, and the LaTeX engine for Latex), its name,
// and its icon. It never holds the compiler, the Go toolchain, the language
// server, documentation, source files, or anything else from the project
// folder: nothing is copied from beside the entry module.
//
//	macOS    Name.app (Contents/MacOS: program and helpers; Resources: icon)
//	Windows  Name/ (Name.exe and runtime/) and Name-windows-x64.zip
//	Linux    Name/ (Name and runtime/) and Name-linux-x64.tar.gz
//
// The program finds its helpers only inside the application (see
// AhdPackagedApplication in the runtime). macOS applications are not code
// signed with a developer certificate; see docs/PACKAGING.md.
package packaging

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ahdcode/internal/backend/golang"
	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/source"
)

// Options are `ahdcode package`'s settings.
type Options struct {
	Entry  string
	Name   string // defaults to the entry module's file name
	Output string // defaults to "dist"
	Icon   string // a PNG; defaults to the AhdCode icon
	// Target is os-arch, such as macos-arm64; it defaults to this computer.
	Target string
	// Helpers is a folder holding the target's AhdCode helpers; it defaults
	// to this installation's own, which fit only this computer's target.
	Helpers string
	// Console keeps a Windows console window for a program that uses no
	// window of its own anyway; a desktop program has none by default.
	Console bool
}

// Result describes a finished application.
type Result struct {
	Application string   // the .app bundle or the application folder
	Archive     string   // the .zip or .tar.gz, if any
	Helpers     []string // the helpers it holds
	Target      string
	Diagnostics []diagnostics.Diagnostic
	// Files render the diagnostics' source locations.
	Files map[source.FileID]source.File
}

// Targets are the supported package targets.
var Targets = map[string]build.Target{
	"macos-arm64":   {OS: "darwin", Arch: "arm64"},
	"macos-x64":     {OS: "darwin", Arch: "amd64"},
	"windows-x64":   {OS: "windows", Arch: "amd64"},
	"windows-arm64": {OS: "windows", Arch: "arm64"},
	"linux-x64":     {OS: "linux", Arch: "amd64"},
	"linux-arm64":   {OS: "linux", Arch: "arm64"},
}

// HostTarget is this computer's target name.
func HostTarget() string {
	for name, target := range Targets {
		if target.OS == runtime.GOOS && target.Arch == runtime.GOARCH {
			return name
		}
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}

// Name bounds.
const maxNameRunes = 64

// ValidName reports whether a text can name an application: letters,
// digits, spaces, and . _ - only, not starting with a dot or a space.
func ValidName(name string) bool {
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes || strings.HasPrefix(name, ".") ||
		strings.TrimSpace(name) != name {
		return false
	}
	for _, character := range name {
		if !(unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune(" ._-", character)) {
			return false
		}
	}
	return true
}

// helperSet is one bundled helper or resource folder.
type helperSet struct {
	name     string
	override string // the environment variable that locates it in development
	folder   bool   // a folder (the LaTeX engine), not one executable
}

// helpersFor lists what a program needs, from its compile graph.
func helpersFor(program *golang.GeneratedProgram) []helperSet {
	var helpers []helperSet
	add := func(set helperSet) {
		for _, known := range helpers {
			if known.name == set.name {
				return
			}
		}
		helpers = append(helpers, set)
	}
	if program.RequiresGUI {
		add(helperSet{name: "ahdgui", override: "AHDCODE_GUI_RUNTIME"})
	}
	if program.RequiresGraphics {
		add(helperSet{name: "ahdgraphics", override: "AHDCODE_GRAPHICS_RUNTIME"})
	}
	if program.RequiresPlot {
		add(helperSet{name: "ahdplot", override: "AHDCODE_PLOT_RUNTIME"})
		add(helperSet{name: "ahdplotview", override: "AHDCODE_PLOTVIEW_RUNTIME"})
		// The Plot viewer's Save button uses the GUI helper's save dialog.
		add(helperSet{name: "ahdgui", override: "AHDCODE_GUI_RUNTIME"})
	}
	if program.RequiresSQLite {
		add(helperSet{name: "ahdsqlite", override: "AHDCODE_SQLITE_RUNTIME"})
	}
	if program.RequiresNumeric {
		add(helperSet{name: "ahdnumeric", override: "AHDCODE_NUMERIC_RUNTIME"})
	}
	if program.RequiresLatex {
		add(helperSet{name: "latex", override: "AHDCODE_LATEX_RUNTIME", folder: true})
	}
	sort.Slice(helpers, func(a, b int) bool { return helpers[a].name < helpers[b].name })
	return helpers
}

// findHelper locates one helper: in the --helpers folder, or in this
// installation.
func findHelper(set helperSet, helpers string, target build.Target) (string, error) {
	file := set.name
	if target.OS == "windows" && !set.folder {
		file += ".exe"
	}
	var candidates []string
	if helpers != "" {
		candidates = append(candidates, filepath.Join(helpers, file), filepath.Join(helpers, "libexec", "ahdcode", file))
	} else {
		if custom := os.Getenv(set.override); custom != "" {
			candidates = append(candidates, custom, filepath.Join(custom, file))
		}
		if executable, err := os.Executable(); err == nil {
			bin := filepath.Dir(executable)
			if resolved, err := filepath.EvalSymlinks(executable); err == nil {
				bin = filepath.Dir(resolved)
			}
			candidates = append(candidates, filepath.Join(bin, file), filepath.Join(bin, "..", "libexec", "ahdcode", file))
		}
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if set.folder && info.IsDir() && filepath.Base(candidate) == "latex" {
			return filepath.Abs(candidate)
		}
		if !set.folder && info.Mode().IsRegular() && filepath.Base(candidate) == file {
			return filepath.Abs(candidate)
		}
	}
	where := "this AhdCode installation"
	if helpers != "" {
		where = helpers
	}
	return "", fmt.Errorf("the %s helper (%s) was not found in %s", set.name, file, where)
}

// appIdentity is ahdcode-app.json, which tells the helpers whose
// application they belong to.
type appIdentity struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

const (
	identityFile = "ahdcode-app.json"
	iconFile     = "app-icon.png"
	// archiveTime is every archive entry's time, so an archive depends only
	// on its content.
	archiveYear = 2000
)

// Package makes the application.
func Package(options Options) Result {
	result := Result{}
	fail := func(format string, arguments ...any) Result {
		result.Diagnostics = append(result.Diagnostics, diagnostics.Diagnostic{
			Code: "PKG001", Severity: diagnostics.SeverityError, Message: fmt.Sprintf(format, arguments...),
		})
		return result
	}
	if options.Entry == "" || !strings.EqualFold(filepath.Ext(options.Entry), ".ahd") {
		return fail("ahdcode package needs one entry module ending in .ahd")
	}
	if options.Name == "" {
		options.Name = strings.TrimSuffix(filepath.Base(options.Entry), filepath.Ext(options.Entry))
	}
	if !ValidName(options.Name) {
		return fail("%q cannot name an application; use up to %d letters, digits, spaces, dots, underscores, or hyphens", options.Name, maxNameRunes)
	}
	if options.Output == "" {
		options.Output = "dist"
	}
	if options.Target == "" {
		options.Target = HostTarget()
	}
	target, known := Targets[options.Target]
	if !known {
		return fail("unknown target %q; use one of %s", options.Target, targetList())
	}
	result.Target = options.Target
	if options.Target != HostTarget() && options.Helpers == "" {
		return fail("packaging for %s needs --helpers with the %s helpers, such as the libexec/ahdcode folder of an AhdCode %s download; this installation's helpers run only on %s",
			options.Target, options.Target, options.Target, HostTarget())
	}
	iconBytes, iconImage, err := loadIcon(options.Icon)
	if err != nil {
		return fail("%s", err.Error())
	}

	compiled := build.Compile(options.Entry)
	result.Diagnostics = append(result.Diagnostics, compiled.Diagnostics...)
	result.Files = compiled.Files
	if compiled.HasErrors() || compiled.Program == nil {
		return result
	}
	program := compiled.Program
	sets := helpersFor(program)
	sources := map[string]string{}
	for _, set := range sets {
		source, err := findHelper(set, options.Helpers, target)
		if err != nil {
			return fail("%s", err.Error())
		}
		sources[set.name] = source
		result.Helpers = append(result.Helpers, set.name)
	}

	if err := os.MkdirAll(options.Output, 0o755); err != nil {
		return fail("the output folder %s cannot be created", options.Output)
	}
	staging, err := os.MkdirTemp(options.Output, ".ahdcode-package-")
	if err != nil {
		return fail("the output folder %s is not writable", options.Output)
	}
	defer os.RemoveAll(staging)

	layout := layoutFor(options.Name, target)
	root := filepath.Join(staging, layout.root)
	for _, folder := range []string{layout.program, layout.helpers, layout.resources} {
		if err := os.MkdirAll(filepath.Join(root, folder), 0o755); err != nil {
			return fail("the application cannot be staged: %v", err)
		}
	}

	// The program itself, built for the target with no installation hint.
	packaged := build.PackagedBuild{Target: target}
	if target.OS == "windows" {
		machine := uint16(0x8664)
		if target.Arch == "arm64" {
			machine = 0xAA64
		}
		packaged.Files = append(packaged.Files, golang.GeneratedFile{
			Name:    "ahdcode_application_icon_windows_" + target.Arch + ".syso",
			Content: string(makeWindowsResource(iconImage, machine)),
		})
		if !options.Console && (program.RequiresGUI || program.RequiresGraphics || program.RequiresPlot) {
			packaged.LinkerFlags = append(packaged.LinkerFlags, "-H=windowsgui")
		}
	}
	executable := filepath.Join(root, layout.program, layout.executable)
	if failures := build.BuildPackagedProgram(program, executable, packaged); len(failures) != 0 {
		result.Diagnostics = append(result.Diagnostics, failures...)
		return result
	}

	for _, set := range sets {
		destination := filepath.Join(root, layout.helpers, filepath.Base(sources[set.name]))
		if set.folder {
			err = copyTree(sources[set.name], destination)
		} else {
			err = copyFile(sources[set.name], destination, 0o755)
		}
		if err != nil {
			return fail("the %s helper cannot be copied: %v", set.name, err)
		}
	}

	identity, _ := json.MarshalIndent(appIdentity{Name: options.Name, Icon: iconFile}, "", "  ")
	writes := map[string][]byte{
		filepath.Join(layout.identity, identityFile): append(identity, '\n'),
		filepath.Join(layout.identity, iconFile):     iconBytes,
	}
	if target.OS == "darwin" {
		writes[filepath.Join("Contents", "Info.plist")] = infoPlist(options.Name, layout.executable)
		writes[filepath.Join("Contents", "PkgInfo")] = []byte("APPL????")
		shaped := iconImage
		if options.Icon == "" {
			shaped = macOSShape(iconImage)
		}
		writes[filepath.Join("Contents", "Resources", "AppIcon.icns")] = makeICNS(shaped)
	}
	for path, content := range writes {
		if err := os.WriteFile(filepath.Join(root, path), content, 0o644); err != nil {
			return fail("the application cannot be staged: %v", err)
		}
	}

	if problems := leakGate(root, layout, sets, target); len(problems) != 0 {
		return fail("the application was not written, because it would hold files it must not: %s", strings.Join(problems, "; "))
	}

	final := filepath.Join(options.Output, layout.root)
	if err := replace(final, filepath.Join(staging, layout.root)); err != nil {
		return fail("%s", err.Error())
	}
	result.Application, _ = filepath.Abs(final)
	switch target.OS {
	case "windows":
		result.Archive, err = writeArchive(filepath.Join(options.Output, options.Name+"-"+options.Target+".zip"), final, layout.root, true)
	case "linux":
		result.Archive, err = writeArchive(filepath.Join(options.Output, options.Name+"-"+options.Target+".tar.gz"), final, layout.root, false)
	}
	if err != nil {
		return fail("the archive cannot be written: %v", err)
	}
	return result
}

func targetList() string {
	var names []string
	for name := range Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// layout is where everything goes inside an application.
type layout struct {
	root       string // the .app or the folder
	program    string // the folder of the main executable
	executable string
	helpers    string
	resources  string
	identity   string // the folder of ahdcode-app.json
}

func layoutFor(name string, target build.Target) layout {
	switch target.OS {
	case "darwin":
		return layout{root: name + ".app", program: "Contents/MacOS", executable: name,
			helpers: "Contents/MacOS", resources: "Contents/Resources", identity: "Contents/Resources"}
	case "windows":
		return layout{root: name, program: ".", executable: name + ".exe", helpers: "runtime", resources: "runtime", identity: "runtime"}
	}
	return layout{root: name, program: ".", executable: name, helpers: "runtime", resources: "runtime", identity: "runtime"}
}

// bundleIdentifier is a reverse-DNS identifier made from the name.
func bundleIdentifier(name string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(name) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9', character == '.', character == '-':
			builder.WriteRune(character)
		default:
			builder.WriteRune('-')
		}
	}
	part := strings.Trim(builder.String(), ".-")
	if part == "" {
		part = "application"
	}
	return "org.ahdcode.app." + part
}

func xmlText(text string) string {
	var buffer bytes.Buffer
	for _, character := range text {
		switch character {
		case '&':
			buffer.WriteString("&amp;")
		case '<':
			buffer.WriteString("&lt;")
		case '>':
			buffer.WriteString("&gt;")
		default:
			buffer.WriteRune(character)
		}
	}
	return buffer.String()
}

// infoPlist is the bundle's Info.plist. The bundle's main executable opens
// no window itself -- its helpers do -- so it is an agent (LSUIElement); each
// helper window shows the application's name and icon.
func infoPlist(name, executable string) []byte {
	entries := [][2]string{
		{"CFBundleDevelopmentRegion", "en"},
		{"CFBundleDisplayName", name},
		{"CFBundleExecutable", executable},
		{"CFBundleIconFile", "AppIcon"},
		{"CFBundleIdentifier", bundleIdentifier(name)},
		{"CFBundleInfoDictionaryVersion", "6.0"},
		{"CFBundleName", name},
		{"CFBundlePackageType", "APPL"},
		{"CFBundleShortVersionString", "1.0"},
		{"CFBundleVersion", "1"},
		{"LSMinimumSystemVersion", "11.0"},
	}
	var buffer bytes.Buffer
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	for _, entry := range entries {
		fmt.Fprintf(&buffer, "\t<key>%s</key>\n\t<string>%s</string>\n", entry[0], xmlText(entry[1]))
	}
	buffer.WriteString("\t<key>AhdCodePackagedApplication</key>\n\t<true/>\n")
	buffer.WriteString("\t<key>LSUIElement</key>\n\t<true/>\n")
	buffer.WriteString("\t<key>NSHighResolutionCapable</key>\n\t<true/>\n")
	buffer.WriteString("</dict>\n</plist>\n")
	return buffer.Bytes()
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

// copyTree copies a helper folder: regular files and folders only.
func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(source, path)
		target := filepath.Join(destination, relative)
		switch {
		case entry.IsDir():
			return os.MkdirAll(target, 0o755)
		case entry.Type().IsRegular():
			info, err := entry.Info()
			if err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if info.Mode()&0o111 != 0 {
				mode = 0o755
			}
			return copyFile(path, target, mode)
		}
		return fmt.Errorf("%s is neither a file nor a folder", path)
	})
}

// replace moves the staged application into place. An existing application
// is replaced only when it is one `ahdcode package` made: it holds an
// ahdcode-app.json. Anything else is never removed.
func replace(final, staged string) error {
	if _, err := os.Lstat(final); err == nil {
		marker := false
		for _, candidate := range []string{filepath.Join(final, "runtime", identityFile), filepath.Join(final, "Contents", "Resources", identityFile)} {
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				marker = true
			}
		}
		if !marker {
			return fmt.Errorf("%s already exists and is not an application made by ahdcode package; choose another --output or --name", final)
		}
		if err := os.RemoveAll(final); err != nil {
			return fmt.Errorf("the earlier %s cannot be replaced: %v", final, err)
		}
	}
	return os.Rename(staged, final)
}

// writeArchive writes a zip or tar.gz of the application folder, with fixed
// times and sorted entries, so it depends only on the content.
func writeArchive(path, folder, root string, zipped bool) (string, error) {
	var files []string
	err := filepath.WalkDir(folder, func(file string, entry fs.DirEntry, err error) error {
		if err == nil && entry.Type().IsRegular() {
			files = append(files, file)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	output, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer output.Close()
	stamp := time.Date(archiveYear, 1, 1, 0, 0, 0, 0, time.UTC)
	if zipped {
		writer := zip.NewWriter(output)
		for _, file := range files {
			relative, _ := filepath.Rel(folder, file)
			info, err := os.Stat(file)
			if err != nil {
				return "", err
			}
			header := &zip.FileHeader{Name: filepath.ToSlash(filepath.Join(root, relative)), Method: zip.Deflate, Modified: stamp}
			header.SetMode(info.Mode().Perm())
			entry, err := writer.CreateHeader(header)
			if err != nil {
				return "", err
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}
			if _, err := entry.Write(content); err != nil {
				return "", err
			}
		}
		if err := writer.Close(); err != nil {
			return "", err
		}
	} else {
		compressed, _ := gzip.NewWriterLevel(output, gzip.BestCompression)
		compressed.ModTime = stamp
		writer := tar.NewWriter(compressed)
		for _, file := range files {
			relative, _ := filepath.Rel(folder, file)
			info, err := os.Stat(file)
			if err != nil {
				return "", err
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return "", err
			}
			header := &tar.Header{Name: filepath.ToSlash(filepath.Join(root, relative)), Mode: int64(info.Mode().Perm()),
				Size: int64(len(content)), ModTime: stamp, Typeflag: tar.TypeReg, Format: tar.FormatPAX}
			if err := writer.WriteHeader(header); err != nil {
				return "", err
			}
			if _, err := writer.Write(content); err != nil {
				return "", err
			}
		}
		if err := writer.Close(); err != nil {
			return "", err
		}
		if err := compressed.Close(); err != nil {
			return "", err
		}
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

// ---- leak gate ---------------------------------------------------------------

// forbiddenNames are files an application must never hold.
var forbiddenNames = regexp.MustCompile(`(?i)(^\.env|^\.git|\.ahd$|\.go$|\.db$|\.sqlite3?$|\.key$|\.pem$|\.p12$|id_rsa|id_ed25519|\.DS_Store|\.log$|^go\.mod$|^go\.sum$|\.xlsx$|\.csv$)`)

// secretPatterns are texts that must not appear in the application's own
// metadata files.
var secretPatterns = regexp.MustCompile(`(?i)(-----BEGIN [A-Z ]*PRIVATE KEY-----|password"?\s*[=:]|api[_-]?key"?\s*[=:]|secret"?\s*[=:]|token"?\s*[=:])`)

// leakGate checks the staged application: every file is one packaging put
// there -- the program, the listed helpers, and the generated metadata --
// no file has a forbidden name, and the generated metadata holds no secret.
// It cannot see into the program's own compiled text: a secret written in
// the program's source is the program's.
func leakGate(root string, layout layout, sets []helperSet, target build.Target) []string {
	allowed := map[string]bool{
		filepath.Join(layout.program, layout.executable):       true,
		filepath.Join(layout.identity, identityFile):           true,
		filepath.Join(layout.identity, iconFile):               true,
		filepath.Join("Contents", "Info.plist"):                target.OS == "darwin",
		filepath.Join("Contents", "PkgInfo"):                   target.OS == "darwin",
		filepath.Join("Contents", "Resources", "AppIcon.icns"): target.OS == "darwin",
	}
	folders := map[string]bool{}
	for _, set := range sets {
		name := set.name
		if target.OS == "windows" && !set.folder {
			name += ".exe"
		}
		if set.folder {
			folders[filepath.Join(layout.helpers, name)] = true
		} else {
			allowed[filepath.Join(layout.helpers, name)] = true
		}
	}
	var problems []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			problems = append(problems, err.Error())
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		if relative == "." {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			problems = append(problems, relative+" is a symbolic link")
			return nil
		}
		if forbiddenNames.MatchString(entry.Name()) {
			problems = append(problems, relative+" must not be packaged")
		}
		if entry.IsDir() {
			return nil
		}
		inFolder := false
		for folder := range folders {
			if strings.HasPrefix(relative, folder+string(filepath.Separator)) {
				inFolder = true
			}
		}
		if !allowed[relative] && !inFolder {
			problems = append(problems, relative+" is not part of the application")
		}
		if strings.HasSuffix(relative, ".json") || strings.HasSuffix(relative, ".plist") {
			if content, err := os.ReadFile(path); err == nil && secretPatterns.Match(content) {
				problems = append(problems, relative+" holds something that looks like a secret")
			}
		}
		return nil
	})
	sort.Strings(problems)
	return problems
}
