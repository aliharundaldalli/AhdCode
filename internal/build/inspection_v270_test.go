package build

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSystemInspectionMatchesBetweenEvaluatorAndNative runs one Disk/Service
// program natively and in the evaluator. Byte counts move from moment to
// moment, so the program prints their invariants rather than their values.
func TestSystemInspectionMatchesBetweenEvaluatorAndNative(t *testing.T) {
	source := `bring Disk
bring File
bring Service
from Disk bring (DiskInfo, DiskError)
from Service bring (ServiceInfo, ServiceError)

File.createDir("data/nested")
info: DiskInfo := Disk.inspect("data/nested")
write(info.path())
write("consistent: " + str(info.usedBytes() == info.totalBytes() - info.freeBytes() and info.availableBytes() <= info.freeBytes() and info.freeBytes() <= info.totalBytes() and info.totalBytes() > 0))
write("percent in range: " + str(info.usedPercent() >= 0.0 and info.usedPercent() <= 100.0))
root := Disk.inspect(".")
write("same filesystem: " + str(root.totalBytes() == info.totalBytes()))
for bad in ["", "missing/dir"] {
    attempt {
        Disk.inspect(bad)
    }
    except DiskError as failure {
        write(failure.message)
    }
}
for name in ["nginx.service; touch SHOULD_NOT_EXIST", "$(id)", "../x", ""] {
    attempt {
        Service.status(name)
    }
    except ServiceError as failure {
        write(failure.message)
    }
}
attempt {
    status: Local ServiceInfo := Service.status("nginx.service")
    write(status.name() + " " + status.activeState() + " " + status.subState() + " " + str(status.running()) + " " + str(status.enabled()))
}
except Error as failure {
    write("error: " + failure.message)
}
write("shell ran: " + str(File.exists("SHOULD_NOT_EXIST")))
`
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.ahd")
	if err := os.WriteFile(entry, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	native, stderr, code := buildAndRunIn(t, entry, directory)
	if code != 0 || stderr != "" {
		t.Fatalf("native exit %d stderr %q:\n%s", code, stderr, native)
	}
	evaluatorDirectory := t.TempDir()
	var output, errorOutput bytes.Buffer
	runEvaluatorIn(t, evaluatorDirectory, source, &output, &errorOutput)
	if output.String() != native {
		t.Fatalf("evaluator and native differ\nnative:\n%s\nevaluator:\n%s\nstderr: %s", native, output.String(), errorOutput.String())
	}
	wants := []string{
		"data/nested\n", "consistent: true\n", "percent in range: true\n", "same filesystem: true\n",
		`inspect disk of "" failed: the path is empty`,
		`inspect disk of "missing/dir" failed: no such file or directory`,
		`service "nginx.service; touch SHOULD_NOT_EXIST" status failed: the service name contains ';'`,
		`service "$(id)" status failed: the service name contains '$'`,
		`service "../x" status failed: the service name must not start with '-' or '.'`,
		`service "" status failed: the service name is empty`,
		"shell ran: false\n",
	}
	if runtime.GOOS != "linux" {
		wants = append(wants, `error: service "nginx.service" status failed: service inspection is supported only on Linux with systemd; this system is `+runtime.GOOS)
	}
	for _, want := range wants {
		if !strings.Contains(native, want) {
			t.Fatalf("output lacks %q:\n%s", want, native)
		}
	}
}

// TestV270ExamplesCompile keeps examples/v2.7 compiling.
func TestV270ExamplesCompile(t *testing.T) {
	for _, name := range []string{"disk_inspect", "service_status"} {
		entry, err := filepath.Abs(filepath.Join("..", "..", "examples", "v2.7", name, "main.ahd"))
		if err != nil {
			t.Fatal(err)
		}
		if _, result := BuildProgram(entry, filepath.Join(t.TempDir(), name)); result.HasErrors() {
			t.Fatalf("%s does not compile:\n%s", name, diagnosticText(result.Diagnostics))
		}
	}
}
