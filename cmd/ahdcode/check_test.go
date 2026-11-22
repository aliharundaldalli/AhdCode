package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"ahdcode/internal/analysis"
	"ahdcode/internal/build"
	"ahdcode/internal/diagnostics"
	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/lsp"
	"ahdcode/internal/module"
	"ahdcode/internal/source"
)

func checkFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "project with spaces")
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func checkedJSON(t *testing.T, entry string, wantExit int) (checkReport, string) {
	t.Helper()
	var out, stderr bytes.Buffer
	if code := runWithIO([]string{"check", entry, "--format", "json"}, nil, &out, &stderr); code != wantExit || stderr.Len() != 0 {
		t.Fatalf("exit=%d, stdout=%q, stderr=%q; want exit=%d and empty stderr", code, &out, &stderr, wantExit)
	}
	var report checkReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.SchemaVersion != 1 || report.Diagnostics == nil {
		t.Fatalf("invalid report %q: %v", &out, err)
	}
	return report, out.String()
}

func TestCheckJSONDeterministicAndSourceAddressed(t *testing.T) {
	const entryText = "bring Zebra\r\nrequire(\"pages/Çalışma.ahd\")\r\nwrite(\"😀<&>\" + missing)\r\n"
	dir := checkFixture(t, map[string]string{
		"Main.ahd":          entryText,
		"Zebra.ahd":         "value: Int := true\r\n",
		"pages/Çalışma.ahd": "bad: Int := false\r\n",
	})
	entry := filepath.Join(dir, "Main.ahd")
	report, raw := checkedJSON(t, entry, 1)
	for range 4 {
		if _, repeated := checkedJSON(t, entry, 1); repeated != raw {
			t.Fatalf("JSON changed across invocations:\n%s\n%s", raw, repeated)
		}
	}
	if strings.Contains(raw, "project with spaces") || strings.Contains(raw, "😀") || strings.Contains(raw, "<&>") {
		t.Fatalf("unexpected source excerpt or absolute path in %s", raw)
	}
	want := []string{"Main.ahd", "Zebra.ahd", "pages/Çalışma.ahd"}
	var paths []string
	for _, item := range report.Diagnostics {
		if item.Severity != "error" || item.Range == nil {
			t.Fatalf("diagnostic has unexpected severity or range: %+v", item)
		}
		paths = append(paths, item.Path)
	}
	if got := slices.Compact(paths); !slices.Equal(got, want) {
		t.Fatalf("diagnostics paths = %q; want %q", got, want)
	}
	index := slices.IndexFunc(report.Diagnostics, func(item checkDiagnostic) bool {
		return item.Path == "Main.ahd" && item.Code == "SEM001"
	})
	if index < 0 {
		t.Fatalf("missing the unknown-name diagnostic: %+v", report.Diagnostics)
	}
	if got := *report.Diagnostics[index].Range; got != (source.UTF16Range{
		Start: source.UTF16Position{Line: 2, Character: 16},
		End:   source.UTF16Position{Line: 2, Character: 23},
	}) {
		t.Fatalf("CRLF and emoji range: %+v", got)
	}

	var out, stderr bytes.Buffer
	if code := runCheck([]string{entry}, &out, &stderr); code != 1 || stderr.Len() != 0 ||
		!strings.Contains(out.String(), "Main.ahd:3:17: error SEM001:") || !strings.Contains(out.String(), "  hint: ") {
		t.Fatalf("text diagnostic: exit=%d stdout=%q stderr=%q", code, &out, &stderr)
	}
}

func TestCheckMissingAndInvalidInputs(t *testing.T) {
	dir := checkFixture(t, map[string]string{
		"missing-import.ahd":  "bring DoesNotExist\n",
		"missing-require.ahd": "require(\"sub/Missing.ahd\")\n",
		"cycle.ahd":           "bring Loop\n",
		"Loop.ahd":            "bring cycle\n",
		"syntax.ahd":          "value: Int :=\n",
		"wrong.txt":           "write(\"valid syntax\")\n",
	})
	for _, test := range []struct {
		path, code string
		localized  bool
	}{
		{"missing-import.ahd", "SEM028", true},
		{"missing-require.ahd", "SEM046", true},
		{"cycle.ahd", "SEM032", true},
		{"syntax.ahd", "PAR", true},
		{"wrong.txt", "SEM028", false},
		{"missing.ahd", "SEM028", false},
	} {
		t.Run(test.path, func(t *testing.T) {
			entry := filepath.Join(dir, test.path)
			report, raw := checkedJSON(t, entry, 1)
			if strings.Contains(raw, "project with spaces") || len(report.Diagnostics) == 0 {
				t.Fatalf("empty diagnostics or leaked local path: %s", raw)
			}
			if !slices.ContainsFunc(report.Diagnostics, func(item checkDiagnostic) bool {
				return strings.HasPrefix(item.Code, test.code) && (item.Range != nil) == test.localized
			}) {
				t.Fatalf("missing expected diagnostic %s (with range: %t): %s", test.code, test.localized, raw)
			}
		})
	}
}

func TestCheckUsageAndJSONEscaping(t *testing.T) {
	for _, args := range [][]string{
		nil, {"--format", "json"}, {"main.ahd", "--format"}, {"main.ahd", "--format", "yaml"},
		{"main.ahd", "--wrong"}, {"a.ahd", "b.ahd"}, {"a.ahd", "--format", "json", "--format=text"},
	} {
		var out, stderr bytes.Buffer
		if code := runCheck(args, &out, &stderr); code != 2 || out.Len() != 0 || !strings.Contains(stderr.String(), "usage:") {
			t.Fatalf("%q: exit=%d stdout=%q stderr=%q", args, code, &out, &stderr)
		}
	}
	dir := checkFixture(t, map[string]string{"-main.ahd": "write(\"valid\")\n"})
	var out, stderr bytes.Buffer
	if code := runCheck([]string{"--format=json", "--", filepath.Join(dir, "-main.ahd")}, &out, &stderr); code != 0 ||
		out.String() != "{\"schemaVersion\":1,\"diagnostics\":[]}\n" || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, &out, &stderr)
	}
	result := module.CompilationResult{Diagnostics: []module.ModuleDiagnostic{{Diagnostic: diagnostics.Diagnostic{
		Code: "test-code", Severity: diagnostics.SeverityWarning, Message: "</script>\"\n", Hint: "\"quoted\"",
	}}}}
	out.Reset()
	if err := writeCheckReport(&out, "json", makeCheckReport("Main.ahd", result)); err != nil {
		t.Fatal(err)
	}
	var report checkReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || result.HasErrors() ||
		strings.Contains(out.String(), "</script>") || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Severity != "warning" || report.Diagnostics[0].Message != "</script>\"\n" ||
		report.Diagnostics[0].Hint != "\"quoted\"" {
		t.Fatalf("warning JSON: %q, %v", &out, err)
	}
}

func TestCheckWithoutGoOnPATHDoesNotRunOrCreateArtifacts(t *testing.T) {
	const program = "bring SQLite\nbring File\nFile.writeText(\"side-effect.txt\", \"executed\")\ndb := SQLite.open(\"side-effect.db\")\n"
	dir := checkFixture(t, map[string]string{"Main with spaces.ahd": program})
	scratch := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"AHDCODE_TEST_CHECK_ENTRY="+filepath.Join(dir, "Main with spaces.ahd"),
		"PATH="+scratch, "GOROOT="+scratch, "TMPDIR="+scratch, "TEMP="+scratch, "TMP="+scratch,
		"AHDCODE_SQLITE_RUNTIME="+filepath.Join(scratch, "missing-helper"),
	)
	out, err := command.CombinedOutput()
	if err != nil || string(out) != "{\"schemaVersion\":1,\"diagnostics\":[]}\n" {
		t.Fatalf("check without Go on PATH: %v, %q", err, out)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 || files[0].Name() != "Main with spaces.ahd" {
		t.Fatalf("unexpected project contents: %v, %v", files, err)
	}
	content, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil || string(content) != program {
		t.Fatalf("entry changed: %v, %q", err, content)
	}
	temporary, err := os.ReadDir(scratch)
	if err != nil || len(temporary) != 0 {
		t.Fatalf("unexpected temporary build contents: %v, %v", temporary, err)
	}
}

func checkerLSP(t *testing.T, entry, text string) map[string][]checkDiagnostic {
	t.Helper()
	root := filepath.Dir(analysis.CanonicalPath(entry))
	var input, output, stderr bytes.Buffer
	message := func(body any) {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&input, "Content-Length: %d\r\n\r\n%s", len(data), data)
	}
	message(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	message(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
		"textDocument": map[string]any{"uri": lsp.PathToURI(entry), "text": text},
	}})
	message(map[string]any{"jsonrpc": "2.0", "method": "exit"})
	if exit := runWithIO([]string{"lsp"}, &input, &output, &stderr); exit != 0 {
		t.Fatalf("lsp exit=%d, stderr=%s", exit, &stderr)
	}
	published := make(map[string][]checkDiagnostic)
	for output.Len() != 0 {
		header, _, ok := bytes.Cut(output.Bytes(), []byte("\r\n\r\n"))
		var size int
		if !ok {
			t.Fatalf("expected LSP header: %s", &output)
		}
		if _, err := fmt.Sscanf(string(header), "Content-Length: %d", &size); err != nil {
			t.Fatal(err)
		}
		output.Next(len(header) + 4)
		// Decode the common fields independently because LSP uses numeric severity.
		var envelope struct {
			Method string
			Params json.RawMessage
		}
		if err := json.Unmarshal(output.Next(size), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Method == "textDocument/publishDiagnostics" {
			var params struct {
				URI         string
				Diagnostics []struct {
					Code     string
					Message  string
					Range    *source.UTF16Range
					Severity int
				}
			}
			if err := json.Unmarshal(envelope.Params, &params); err != nil {
				t.Fatal(err)
			}
			// The LSP emits empty publications for bundled virtual files
			// as well. Their URIs are not local filesystem URIs.
			if strings.HasPrefix(params.URI, "file://%3Cahdcode%3E/") && len(params.Diagnostics) == 0 {
				continue
			}
			path, err := lsp.URIToPath(params.URI)
			if err != nil {
				t.Fatal(err)
			}
			relative := checkSourcePath(root, analysis.CanonicalPath(path))
			items := make([]checkDiagnostic, 0, len(params.Diagnostics))
			for _, item := range params.Diagnostics {
				severity := "error"
				if item.Severity == 2 {
					severity = "warning"
				}
				msg, hint, _ := strings.Cut(item.Message, "\nhint: ")
				items = append(items, checkDiagnostic{
					Path: relative, Code: item.Code, Message: msg,
					Severity: severity, Range: item.Range, Hint: hint,
				})
			}
			published[relative] = items
		}
	}
	return published
}

func TestCheckConformanceInvalidFrontendAndLSPParity(t *testing.T) {
	for _, test := range []struct{ name, program, sibling, code string }{
		{"unicode-cross-file", "bring Library\r\n", "write(\"😀\" + missing)\r\n", "SEM001"},
		{"missing-import", "bring Missing\r\n", "", "SEM028"},
		{"nullable", "x: Int? := null\r\nwrite(x + 1)\r\n", "", "SEM011"},
		{"function-argument", "send: Function := (id: Int) -> Int { return id }\r\nsend(id: \"bad\")\r\n", "", "SEM004"},
		{"function-unknown-parameter", "send: Function := (id: Int) -> Int { return id }\r\nsend(other: 7)\r\n", "", "SEM016"},
		{"method-argument", "values: List<Int> := [1]\r\nvalues.add(\"bad\")\r\n", "", "SEM004"},
		{"web-ui-argument", "bring Web\r\nWeb.UI.h2(7)\r\n", "", "SEM004"},
		{"web-default-argument-type", "bring Web\r\nWeb.text(\"body\", \"bad\")\r\n", "", "SEM004"},
		{"builtin-argument", "len(5)\r\n", "", "SEM016"},
		{"no-matching-overload", "choose: Function := (x: Int) -> Int { return x }\r\nchoose: Overload Function := (x: Real) -> Real { return x }\r\nchoose(\"bad\")\r\n", "", "SEM026"},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := map[string]string{"Main.ahd": test.program}
			if test.sibling != "" {
				files["Library.ahd"] = test.sibling
			}
			entry := filepath.Join(checkFixture(t, files), "Main.ahd")
			report, _ := checkedJSON(t, entry, 1)
			if !slices.ContainsFunc(report.Diagnostics, func(item checkDiagnostic) bool { return item.Code == test.code }) {
				t.Fatalf("missing %s in %+v", test.code, report.Diagnostics)
			}
			native := build.Compile(entry)
			if !native.HasErrors() || native.IR != nil || native.Program != nil {
				t.Fatal("native pipeline must stop before lowering invalid frontend input")
			}
			if count := len(report.Diagnostics); len(native.Diagnostics) != count {
				t.Fatalf("check diagnostic count %d differs from native count %d", count, len(native.Diagnostics))
			}
			if built := makeCheckReport(entry, *native.Compilation); !reflect.DeepEqual(report, built) {
				t.Fatalf("check %+v differs from native frontend %+v", report, built)
			}
			var editor []checkDiagnostic
			for _, diags := range checkerLSP(t, entry, test.program) {
				editor = append(editor, diags...)
			}
			slices.SortFunc(editor, compareCheckDiagnostics)
			if !reflect.DeepEqual(report.Diagnostics, editor) {
				t.Fatalf("check %+v differs from LSP %+v", report.Diagnostics, editor)
			}
		})
	}
}

func TestCheckConformanceValidEvaluatorAndNativeParity(t *testing.T) {
	const main = "from Library bring (choose, label, preferred)\r\nbring Web\r\nbring HTML\r\n" +
		"use: Function := (operation: Function, x: Int) -> Int { return operation(x) }\r\n" +
		"accept: Function := (x: Real) -> Real { return x }\r\n" +
		"emit: Function := (value: Int?) -> Nothing {\r\n" +
		"    if value == null { return }\r\n" +
		"    write(value + 1)\r\n" +
		"}\r\n" +
		"emit(null)\r\nemit(use(choose, 5))\r\nwrite(accept(7) + choose(1.5))\r\n" +
		"write(label())\r\nwrite(preferred(3))\r\nwrite(len(\"ü😀\"))\r\n" +
		"write(HTML.render(Web.UI.h2(\"ü😀\")))\r\n"
	const library = "choose: Function := (x: Int) -> Int { return x * 2 }\r\n" +
		"choose: Overload Function := (x: Real) -> Real { return x * 3 }\r\n" +
		"label: Function := (value: String := \"varsayılan\") -> String { return value }\r\n" +
		"preferred: Function := (x: Int, extra: Int := 2) -> Int { return x + extra }\r\n" +
		"preferred: Overload Function := (x: Int) -> Int { return x }\r\n"
	dir := checkFixture(t, map[string]string{"Main.ahd": main, "Library.ahd": library})
	entry := filepath.Join(dir, "Main.ahd")
	report, _ := checkedJSON(t, entry, 0)
	if len(report.Diagnostics) != 0 {
		t.Fatalf("valid project: %+v", report)
	}
	editor := checkerLSP(t, entry, main)
	if _, ok := editor["Main.ahd"]; !ok {
		t.Fatal("no LSP diagnostics publication for the valid entry")
	}
	for path, items := range editor {
		if len(items) != 0 {
			t.Fatalf("unexpected LSP diagnostics for %s: %+v", path, items)
		}
	}

	frontend := module.NewCompiler(module.FileResolver{}, module.FileLoader{}).Compile(entry)
	lowered := lowering.LowerCompilation(frontend)
	if frontend.HasErrors() || lowered.HasErrors() {
		t.Fatalf("frontend %v or lowering %v failed", frontend.Diagnostics, lowered.Diagnostics)
	}
	var interpreted bytes.Buffer
	session := evaluator.New(nil, &interpreted, dir)
	if result := session.Execute(lowered.Compilation, 0); result.Failure != nil {
		t.Fatalf("evaluator: %+v", result.Failure)
	}
	path, native := build.BuildProgram(entry, filepath.Join(t.TempDir(), "valid project"))
	if native.HasErrors() {
		t.Fatalf("native build: %+v", native.Diagnostics)
	}
	command := exec.Command(path)
	command.Dir = dir
	compiled, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native execution: %v, %s", err, compiled)
	}
	if want := "11\n11.5\nvarsayılan\n3\n2\n<h2>ü😀</h2>\n"; interpreted.String() != want || string(compiled) != want {
		t.Fatalf("evaluator=%q, native=%q, want=%q", &interpreted, compiled, want)
	}
}
