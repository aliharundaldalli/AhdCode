package build

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/evaluator"
	"ahdcode/internal/lowering"
	"ahdcode/internal/module"
)

func templateObjectOnceSource() string {
	return `bring Templates
from Templates bring (TemplateSet, TemplateData)

calls: Int := 0

buildMap: Function := () -> Pair<String, TemplateData> {
    calls: Global Int
    calls = calls + 1
    return {
        "title": Templates.fromString("bir")
    }
}

pages: TemplateSet := Templates.parse("page.html", r"<p>{{.title}}</p>")
data: TemplateData := Templates.object(buildMap())
write(str(calls))
write(pages.render("page.html", data))
`
}

func TestTemplatesObjectEvaluatesArgumentOnceNative(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "app.ahd")
	if err := os.WriteFile(entry, []byte(templateObjectOnceSource()), 0o644); err != nil {
		t.Fatal(err)
	}
	executable, result := BuildProgram(entry, filepath.Join(directory, "program"))
	if result.HasErrors() {
		t.Fatalf("%s", templateDiagnostics(result))
	}
	command := exec.Command(executable)
	command.Dir = directory
	body, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, body)
	}
	got := strings.ReplaceAll(string(body), "\r\n", "\n")
	if got != "1\n<p>bir</p>\n" {
		t.Fatalf("native output = %q", got)
	}
}

func TestTemplatesObjectEvaluatesArgumentOnceEvaluator(t *testing.T) {
	workspace := module.NewInMemoryWorkspace(map[string]string{"/Main.ahd": templateObjectOnceSource()})
	frontend := module.NewCompiler(workspace, workspace).Compile("/Main.ahd")
	if frontend.HasErrors() {
		t.Fatalf("frontend diagnostics: %+v", frontend.Diagnostics)
	}
	lowered := lowering.LowerCompilation(frontend)
	if lowered.HasErrors() {
		t.Fatalf("lowering diagnostics: %+v", lowered.Diagnostics)
	}
	var stdout bytes.Buffer
	session := evaluator.New(bufio.NewReader(strings.NewReader("")), &stdout, t.TempDir())
	result := session.Execute(lowered.Compilation, 0)
	if result.Failure != nil {
		t.Fatalf("evaluator failed: %s", result.Failure.Message)
	}
	got := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	if got != "1\n<p>bir</p>\n" {
		t.Fatalf("evaluator output = %q", got)
	}
}
