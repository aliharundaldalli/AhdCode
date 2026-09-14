package build

import (
	"path/filepath"
	"testing"
)

// Compile only: the v1.4.0 acceptance programs and the realtime attendance
// dogfood application. Their server-backed behaviour is covered by the
// PostgreSQL and WebSocket integration tests and recorded in
// docs/qa/v1.4/dogfood.md. The flags check that each program pulls in exactly
// the vendored trees it uses.
func TestV140ExamplesCompile(t *testing.T) {
	for _, testCase := range []struct {
		entry                 string
		postgresql, websocket bool
	}{
		{"../../examples/v0.1/69_uuid.ahd", false, false},
		{"../../examples/v0.1/70_env_secret.ahd", false, false},
		{"../../examples/v0.1/71_postgresql.ahd", true, false},
		{"../../examples/v0.1/72_websocket_echo.ahd", false, true},
		{"../../examples/v1.4/realtime_attendance/app.ahd", true, true},
		{"../../examples/v1.4/realtime_attendance/jobs.ahd", true, false},
	} {
		path, err := filepath.Abs(testCase.entry)
		if err != nil {
			t.Fatal(err)
		}
		result := Compile(path)
		if result.HasErrors() {
			t.Fatalf("%s:\n%s", testCase.entry, diagnosticText(result.Diagnostics))
		}
		if result.Program.RequiresPostgreSQL != testCase.postgresql || result.Program.RequiresWebSocket != testCase.websocket {
			t.Fatalf("%s: postgresql=%v websocket=%v, want %v %v", testCase.entry,
				result.Program.RequiresPostgreSQL, result.Program.RequiresWebSocket, testCase.postgresql, testCase.websocket)
		}
	}
}
