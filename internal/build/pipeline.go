package build

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	backend "ahdcode/internal/backend/golang"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/ir"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
	"ahdcode/internal/source"
)

// Result carries every stage's outcome for one compilation.
type Result struct {
	Compilation *module.CompilationResult
	IR          *ir.Compilation
	Program     *backend.GeneratedProgram
	Diagnostics []diagnostics.Diagnostic
	Files       map[source.FileID]source.File
}

// HasErrors reports whether compilation failed.
func (result Result) HasErrors() bool {
	for _, item := range result.Diagnostics {
		if item.Severity == diagnostics.SeverityError {
			return true
		}
	}
	return false
}

// Compile runs the full frontend, lowering, IR validation, and Go generation
// chain for one entry module. Each stage consumes the previous stage's
// decisions instead of repeating them.
func Compile(entryPath string) Result {
	compiler := module.NewCompiler(module.FileResolver{}, module.FileLoader{})
	compilation := compiler.Compile(entryPath)
	result := Result{Compilation: &compilation, Files: make(map[source.FileID]source.File)}
	for _, current := range compilation.Modules {
		if current != nil && current.File.ID != 0 {
			result.Files[current.File.ID] = current.File
		}
	}
	for _, item := range compilation.Diagnostics {
		result.Diagnostics = append(result.Diagnostics, item.Diagnostic)
	}
	if compilation.HasErrors() {
		return result
	}
	lowered := lowering.LowerCompilation(compilation)
	result.Diagnostics = append(result.Diagnostics, lowered.Diagnostics...)
	if lowered.HasErrors() || lowered.Compilation == nil {
		return result
	}
	result.IR = lowered.Compilation
	program, generated := backend.Generate(lowered.Compilation)
	result.Diagnostics = append(result.Diagnostics, generated...)
	if result.HasErrors() {
		return result
	}
	result.Program = program
	return result
}

// Workspace is a controlled temporary directory holding one generated Go
// program. Generated sources never touch the user's source tree.
type Workspace struct {
	Directory string
	toolchain string
}

const workspaceModule = "module ahdcodeprogram\n\ngo 1.25\n"

// NewWorkspace materializes a generated program in a private directory.
func NewWorkspace(program *backend.GeneratedProgram) (*Workspace, []diagnostics.Diagnostic) {
	toolchain, err := FindGoToolchain()
	if err != nil {
		return nil, []diagnostics.Diagnostic{missingToolchain(err)}
	}
	directory, err := os.MkdirTemp("", "ahdcode-build-")
	if err != nil {
		return nil, []diagnostics.Diagnostic{workspaceFailure("could not create a temporary build workspace: " + err.Error())}
	}
	workspace := &Workspace{Directory: directory, toolchain: toolchain}
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(workspaceModule), 0o600); err != nil {
		workspace.Close()
		return nil, []diagnostics.Diagnostic{workspaceFailure("could not write the generated go.mod: " + err.Error())}
	}
	for _, file := range program.Files {
		if err := os.WriteFile(filepath.Join(directory, file.Name), []byte(file.Content), 0o600); err != nil {
			workspace.Close()
			return nil, []diagnostics.Diagnostic{workspaceFailure("could not write generated source " + file.Name + ": " + err.Error())}
		}
	}
	return workspace, nil
}

func missingToolchain(err error) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Code: backend.CodeMissingToolchain, Severity: diagnostics.SeverityError,
		Message: "the Go toolchain is required to build AhdCode programs: " + err.Error(),
		Hint:    "install Go and make the go command reachable from PATH",
	}
}

func workspaceFailure(message string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Code: backend.CodeWorkspaceFailure, Severity: diagnostics.SeverityError, Message: message,
		Hint: "check the temporary directory permissions and available disk space",
	}
}

// Close removes the temporary workspace.
func (workspace *Workspace) Close() {
	if workspace == nil || workspace.Directory == "" {
		return
	}
	_ = os.RemoveAll(workspace.Directory)
}

// BuildExecutable compiles the generated program to a native executable.
// Process arguments are passed directly; no shell interpolation occurs.
func (workspace *Workspace) BuildExecutable(outputPath string) []diagnostics.Diagnostic {
	absolute, err := filepath.Abs(outputPath)
	if err != nil {
		return []diagnostics.Diagnostic{workspaceFailure("could not resolve the output path: " + err.Error())}
	}
	command := exec.Command(workspace.toolchain, "build", "-trimpath", "-o", absolute, ".")
	command.Dir = workspace.Directory
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return []diagnostics.Diagnostic{{
		Code: backend.CodeBuildFailure, Severity: diagnostics.SeverityError,
		Message: "go build failed for the generated program:\n" + message,
		Hint:    "this is a code generation defect; report the failing AhdCode program",
	}}
}

// DefaultOutputPath is the conventional executable name for an entry module.
func DefaultOutputPath(entryPath string) string {
	base := filepath.Base(entryPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		name = "program"
	}
	directory, err := os.Getwd()
	if err != nil {
		return name
	}
	return filepath.Join(directory, name)
}

// BuildProgram compiles one entry module into a native executable at
// outputPath. An empty outputPath uses the conventional default.
func BuildProgram(entryPath, outputPath string) (string, Result) {
	result := Compile(entryPath)
	if result.HasErrors() || result.Program == nil {
		return "", result
	}
	if outputPath == "" {
		outputPath = DefaultOutputPath(entryPath)
	}
	result.Program = configureLatexRuntime(result.Program)
	result.Program = configurePlotRuntime(result.Program)
	result.Program = configureNumericRuntime(result.Program)
	result.Program = configureSQLiteRuntime(result.Program)
	workspace, failures := NewWorkspace(result.Program)
	if len(failures) != 0 {
		result.Diagnostics = append(result.Diagnostics, failures...)
		return "", result
	}
	defer workspace.Close()
	result.Diagnostics = append(result.Diagnostics, workspace.BuildExecutable(outputPath)...)
	if result.HasErrors() {
		return "", result
	}
	return outputPath, result
}

// runExecutable produces the native executable for one generated program and
// the cleanup its workspace needs.
//
// An identical program is built only once. The cache key covers the complete
// generated source, so a hit is only ever possible when the program that would
// be built is byte-for-byte the one already built: any change to any AhdCode
// source or imported module changes the generated text and therefore the key.
// Compilation and diagnostics have already run in full by this point, so the
// cache reuses the native build and nothing else.
func runExecutable(program *backend.GeneratedProgram) (string, func(), []diagnostics.Diagnostic) {
	nothing := func() {}
	toolchain, err := FindGoToolchain()
	if err != nil {
		return "", nothing, []diagnostics.Diagnostic{missingToolchain(err)}
	}
	cache, key, reserved := openRunCache(), "", ""
	if cache != nil {
		key = cache.key(program, toolchain)
		if executable, found := cache.lookup(key); found {
			return executable, nothing, nil
		}
		if name, ok := cache.reserve(); ok {
			reserved = name
		}
	}
	workspace, failures := NewWorkspace(program)
	if len(failures) != 0 {
		return "", nothing, failures
	}
	// A miss builds straight into the cache directory when one is available,
	// so publishing the result is a rename rather than a copy.
	executable := reserved
	if executable == "" {
		executable = filepath.Join(workspace.Directory, "ahdcode-program")
	}
	if built := workspace.BuildExecutable(executable); len(built) != 0 {
		workspace.Close()
		if reserved != "" {
			_ = os.Remove(reserved)
		}
		return "", nothing, built
	}
	if reserved == "" {
		// Without a cache the executable lives in the workspace, so the
		// workspace has to outlive the run.
		return executable, workspace.Close, nil
	}
	workspace.Close()
	if published, ok := cache.publish(key, reserved); ok {
		return published, nothing, nil
	}
	// The build succeeded but could not be published; run it where it is and
	// remove it afterwards rather than leaving a stray file behind.
	return reserved, func() { _ = os.Remove(reserved) }, nil
}

// RunProgram builds one entry module into a temporary executable and runs it,
// propagating its standard streams and exit code.
func RunProgram(entryPath string, arguments []string, stdin *os.File, stdout, stderr *os.File) (int, Result) {
	return RunProgramIO(entryPath, arguments, stdin, stdout, stderr)
}

// RunProgramIO is the stream-generic form used by the REPL and tests. It keeps
// file compilation and interactive execution on the exact same compiler,
// lowering, backend, native-build, and runtime path.
func RunProgramIO(entryPath string, arguments []string, stdin io.Reader, stdout, stderr io.Writer) (int, Result) {
	return RunProgramObserved(entryPath, arguments, stdin, stdout, stderr, nil)
}

// RunProgramObserved is RunProgramIO with one hook invoked after the compiled
// program has started, carrying the running process id. The CLI uses it to
// write an AhdCode run descriptor for `ahdcode kill`; every other caller
// passes nil and behaves exactly as before.
func RunProgramObserved(entryPath string, arguments []string, stdin io.Reader, stdout, stderr io.Writer, started func(pid int)) (int, Result) {
	result := Compile(entryPath)
	if result.HasErrors() || result.Program == nil {
		return 1, result
	}
	result.Program = configureLatexRuntime(result.Program)
	result.Program = configurePlotRuntime(result.Program)
	result.Program = configureNumericRuntime(result.Program)
	result.Program = configureSQLiteRuntime(result.Program)
	executable, cleanup, failures := runExecutable(result.Program)
	defer cleanup()
	result.Diagnostics = append(result.Diagnostics, failures...)
	if result.HasErrors() {
		return 1, result
	}
	command := exec.Command(executable, arguments...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		result.Diagnostics = append(result.Diagnostics, workspaceFailure("could not run the generated executable: "+err.Error()))
		return 1, result
	}
	if started != nil && command.Process != nil {
		started(command.Process.Pid)
	}
	if err := command.Wait(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), result
		}
		result.Diagnostics = append(result.Diagnostics, workspaceFailure("could not run the generated executable: "+err.Error()))
		return 1, result
	}
	return 0, result
}

// configureLatexRuntime records the shared installation resource directory in
// programs that actually use Latex. Ordinary generated programs are unchanged.
// The runtime still validates the files and raises LatexError if an installed
// resource has been removed after build time.
func configureLatexRuntime(program *backend.GeneratedProgram) *backend.GeneratedProgram {
	if program == nil || !program.RequiresLatex {
		return program
	}
	root := findLatexRuntimeRoot()
	if root == "" {
		return program
	}
	// RequiresPlot is preserved from the original program (not just
	// RequiresLatex: true) so a program using both Latex and Plot keeps its
	// Plot hint regardless of which configure*Runtime call ran first.
	copyProgram := *program
	copyProgram.RequiresLatex = true
	copyProgram.Files = append([]backend.GeneratedFile(nil), program.Files...)
	copyProgram.Files = append(copyProgram.Files, backend.GeneratedFile{
		Name:    "ahdcode_latex_runtime.go",
		Content: "package main\n\nfunc init() { AhdLatexRuntimeHint = " + strconv.Quote(root) + " }\n",
	})
	return &copyProgram
}

// configurePlotRuntime records the shared installation resource directory in
// programs that actually use Plot, mirroring configureLatexRuntime. The
// runtime still validates the helper exists and raises PlotError if it is
// missing at run time.
func configurePlotRuntime(program *backend.GeneratedProgram) *backend.GeneratedProgram {
	if program == nil || !program.RequiresPlot {
		return program
	}
	root := findPlotRuntimeRoot()
	if root == "" {
		return program
	}
	copyProgram := *program
	copyProgram.RequiresPlot = true
	copyProgram.Files = append([]backend.GeneratedFile(nil), program.Files...)
	copyProgram.Files = append(copyProgram.Files, backend.GeneratedFile{
		Name:    "ahdcode_plot_runtime.go",
		Content: "package main\n\nfunc init() { AhdPlotRuntimeHint = " + strconv.Quote(root) + " }\n",
	})
	return &copyProgram
}

// configureNumericRuntime preserves every unrelated runtime requirement while
// adding the installed ahdnumeric discovery hint.
func configureNumericRuntime(program *backend.GeneratedProgram) *backend.GeneratedProgram {
	if program == nil || !program.RequiresNumeric {
		return program
	}
	root := findNumericRuntimeRoot()
	if root == "" {
		return program
	}
	copyProgram := *program
	copyProgram.RequiresNumeric = true
	copyProgram.Files = append([]backend.GeneratedFile(nil), program.Files...)
	copyProgram.Files = append(copyProgram.Files, backend.GeneratedFile{Name: "ahdcode_numeric_runtime.go", Content: "package main\n\nfunc init() { AhdNumericRuntimeHint = " + strconv.Quote(root) + " }\n"})
	return &copyProgram
}

// configureSQLiteRuntime records the installed ahdsqlite helper's directory
// in programs that actually use SQLite, mirroring configureNumericRuntime.
// The runtime still validates the helper exists and raises SQLiteError if it
// is missing at run time.
func configureSQLiteRuntime(program *backend.GeneratedProgram) *backend.GeneratedProgram {
	if program == nil || !program.RequiresSQLite {
		return program
	}
	root := findHelperRuntimeRoot("ahdsqlite", "AHDCODE_SQLITE_RUNTIME")
	if root == "" {
		return program
	}
	copyProgram := *program
	copyProgram.RequiresSQLite = true
	copyProgram.Files = append([]backend.GeneratedFile(nil), program.Files...)
	copyProgram.Files = append(copyProgram.Files, backend.GeneratedFile{Name: "ahdcode_sqlite_runtime_hint.go", Content: "package main\n\nfunc init() { AhdSQLiteRuntimeHint = " + strconv.Quote(root) + " }\n"})
	return &copyProgram
}

// findHelperRuntimeRoot locates the directory holding one bundled helper
// executable: an explicit override (a file or its directory), then the
// compiler's own bin/ directory, then a sibling libexec/ahdcode/ directory.
func findHelperRuntimeRoot(name, override string) string {
	if filepath.Ext(os.Args[0]) == ".exe" {
		name += ".exe"
	}
	var candidates []string
	if custom := os.Getenv(override); custom != "" {
		if info, err := os.Stat(custom); err == nil && info.Mode().IsRegular() {
			candidates = append(candidates, filepath.Dir(custom))
		} else {
			candidates = append(candidates, custom)
		}
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, bin, filepath.Join(bin, "..", "libexec", "ahdcode"))
	}
	for _, candidate := range candidates {
		helper := filepath.Join(filepath.Clean(candidate), name)
		if info, err := os.Stat(helper); err == nil && info.Mode().IsRegular() {
			if absolute, err := filepath.Abs(candidate); err == nil {
				return absolute
			}
		}
	}
	return ""
}

func findNumericRuntimeRoot() string {
	name := "ahdnumeric"
	if filepath.Ext(os.Args[0]) == ".exe" {
		name += ".exe"
	}
	var candidates []string
	if custom := os.Getenv("AHDCODE_NUMERIC_RUNTIME"); custom != "" {
		if info, err := os.Stat(custom); err == nil && info.Mode().IsRegular() {
			candidates = append(candidates, filepath.Dir(custom))
		} else {
			candidates = append(candidates, custom)
		}
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, bin, filepath.Join(bin, "..", "libexec", "ahdcode"))
	}
	for _, candidate := range candidates {
		helper := filepath.Join(filepath.Clean(candidate), name)
		if info, err := os.Stat(helper); err == nil && info.Mode().IsRegular() {
			if absolute, err := filepath.Abs(candidate); err == nil {
				return absolute
			}
		}
	}
	return ""
}

// findPlotRuntimeRoot locates the directory holding the bundled ahdplot
// renderer helper, the same way findLatexRuntimeRoot locates the Tectonic
// bundle: an explicit override, then a path relative to the compiler's own
// executable (bin/ or a sibling libexec/ahdcode/ directory).
func findPlotRuntimeRoot() string {
	name := "ahdplot"
	if filepath.Ext(os.Args[0]) == ".exe" {
		name = "ahdplot.exe"
	}
	var candidates []string
	if custom := os.Getenv("AHDCODE_PLOT_RUNTIME"); custom != "" {
		candidates = append(candidates, filepath.Dir(custom))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, bin, filepath.Join(bin, "..", "libexec", "ahdcode"))
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		helper := filepath.Join(candidate, name)
		if info, err := os.Stat(helper); err == nil && info.Mode().IsRegular() {
			absolute, err := filepath.Abs(candidate)
			if err == nil {
				return absolute
			}
		}
	}
	return ""
}

func findLatexRuntimeRoot() string {
	candidates := []string{os.Getenv("AHDCODE_LATEX_RUNTIME")}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(bin, "latex"),
			filepath.Join(bin, "..", "libexec", "ahdcode", "latex"),
		)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(candidate)
		engine := filepath.Join(candidate, "tectonic")
		if filepath.Ext(os.Args[0]) == ".exe" {
			engine += ".exe"
		}
		bundle := filepath.Join(candidate, "ahdcode-latex.ttb")
		engineInfo, engineError := os.Stat(engine)
		bundleInfo, bundleError := os.Stat(bundle)
		if engineError == nil && bundleError == nil && engineInfo.Mode().IsRegular() && bundleInfo.Mode().IsRegular() {
			absolute, err := filepath.Abs(candidate)
			if err == nil {
				return absolute
			}
		}
	}
	return ""
}
