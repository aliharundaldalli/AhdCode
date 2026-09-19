package build

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain removes the helper builds the tests share (ahdgui, ahdgraphics,
// and the Plot helpers, each built once per test run), so repeated runs do
// not fill the temporary directory.
func TestMain(m *testing.M) {
	code := m.Run()
	for _, file := range []string{guiHelperFile, graphicsHelperFile, plotRenderer, plotViewer} {
		if file != "" {
			_ = os.RemoveAll(filepath.Dir(file))
		}
	}
	os.Exit(code)
}
