package semantic

import (
	"strings"
	"testing"
)

func TestTemplatesModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, `bring Templates
from Templates bring (TemplateSet, TemplateData, TemplateError)

pages: TemplateSet := Templates.parse("pages/home.html", r"<h1>{{.title}}</h1>")
data: TemplateData := Templates.object({
    "title": Templates.fromString("Merhaba")
    "n": Templates.fromInt(2)
})
page: String := pages.layout("pages/home.html", "pages/home.html", data)
page = pages.render("pages/home.html", data)
`)
	requireSemanticClean(t, result)
}

func TestTemplatesRejectsDirectConstruction(t *testing.T) {
	result := analyzeWithStandardModules(t, "bring Templates\nfrom Templates bring TemplateSet\nset: TemplateSet := TemplateSet()\n")
	requireSemanticFailure(t, result)
	if !strings.Contains(result.Diagnostics[0].Message, "Templates.parse") && !strings.Contains(result.Diagnostics[0].Hint, "Templates.parse") {
		t.Fatalf("diagnostic = %#v", result.Diagnostics[0])
	}
}

func TestTemplatesModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["Templates"]
	want := []string{"TemplateData", "TemplateError", "TemplateSet", "files", "fromBool", "fromInt", "fromReal", "fromString", "list", "nullValue", "object", "parse", "sources", "trusted"}
	if strings.Join(module.ExportNames, ",") != strings.Join(want, ",") {
		t.Fatalf("exports %v", module.ExportNames)
	}
	if module.Exports["TemplateError"].Class.Parent.Name != "Error" {
		t.Fatal("TemplateError parent")
	}
}
