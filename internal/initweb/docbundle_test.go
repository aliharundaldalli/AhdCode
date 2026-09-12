package initweb

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"ahdcode/internal/ahdversion"
)

// The documentation a Web project receives is the canonical docs/ tree,
// reconciled into docbundle/ by tooling/distribution/sync_docs.py. These tests
// fail when a release changes the canonical documents without regenerating the
// bundle, so a new project can never describe an older release.

var bundleLink = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)
var bundleImage = regexp.MustCompile(`<img[^>]+>`)

// bundleComparable removes what sync_docs.py legitimately rewrites: link
// targets and images. Everything else must be identical.
func bundleComparable(text string) string {
	return bundleLink.ReplaceAllString(bundleImage.ReplaceAllString(text, ""), "$1")
}

func canonicalDocuments(t *testing.T) map[string]string {
	t.Helper()
	root := filepath.Join("..", "..")
	paths, err := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths, filepath.Join(root, "README.md"), filepath.Join(root, "AHDCODE_LANGUAGE_SPEC_v0.1.md"))
	documents := map[string]string{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_TR.md") {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		documents[filepath.Base(path)] = string(content)
	}
	return documents
}

func TestDocumentationBundleMatchesCanonicalDocs(t *testing.T) {
	canonical := canonicalDocuments(t)
	var want, manifest, bundled []string
	for name := range canonical {
		want = append(want, name)
	}
	manifest = append(manifest, documentationManifest...)
	entries, err := fs.ReadDir(projectDocs, "docbundle")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		bundled = append(bundled, entry.Name())
	}
	sort.Strings(want)
	sort.Strings(manifest)
	sort.Strings(bundled)
	if strings.Join(want, ",") != strings.Join(manifest, ",") {
		t.Fatalf("documentationManifest is stale; run tooling/distribution/sync_docs.py\ncanonical: %v\nmanifest:  %v", want, manifest)
	}
	if strings.Join(want, ",") != strings.Join(bundled, ",") {
		t.Fatalf("docbundle/ is stale; run tooling/distribution/sync_docs.py\ncanonical: %v\nbundled:   %v", want, bundled)
	}
	for name, text := range canonical {
		copy, err := fs.ReadFile(projectDocs, "docbundle/"+name)
		if err != nil {
			t.Fatal(err)
		}
		if bundleComparable(string(copy)) != bundleComparable(text) {
			t.Fatalf("docbundle/%s differs from its canonical document; run tooling/distribution/sync_docs.py", name)
		}
	}
}

func TestDocumentationBundleDescribesThisRelease(t *testing.T) {
	readme, err := fs.ReadFile(projectDocs, "docbundle/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "This is **v"+ahdversion.Number+"**") {
		t.Fatalf("the bundled README does not identify v%s as the current release", ahdversion.Number)
	}
	required := map[string][]string{
		"CRON.md":       {"Cron.scheduler()", "AhdCode Cron  !=  the operating system's crontab"},
		"CHARACTERS.md": {"no `Char` type", "grapheme"},
		"LATEX.md":      {"Latex.tikz", "pgfornament", "landscape"},
		"WEB.md":        {"CRON.md", "Cron: scheduled application work"},
		"MODULES.md":    {"`Cron`", "`Characters`"},
	}
	for name, phrases := range required {
		text, err := fs.ReadFile(projectDocs, "docbundle/"+name)
		if err != nil {
			t.Fatalf("docbundle/%s is missing: %v", name, err)
		}
		for _, phrase := range phrases {
			if !strings.Contains(string(text), phrase) {
				t.Fatalf("docbundle/%s does not mention %q", name, phrase)
			}
		}
	}
}

func TestProjectGuideNamesVersionTwelveDocuments(t *testing.T) {
	text := renderAHDCODE(Options{AppName: "demo"})
	for _, name := range []string{"Documents/AhdCode/CRON.md", "CHARACTERS.md", "LATEX.md"} {
		if !strings.Contains(text, name) {
			t.Fatalf("AHDCODE.md does not point to %s:\n%s", name, text)
		}
	}
}
