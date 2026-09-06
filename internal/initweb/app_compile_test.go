package initweb

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/localdev"
)

func TestWriteDogfoodAdminProject(t *testing.T) {
	dest := strings.TrimSpace(os.Getenv("AHDCODE_DOGFOOD_DIR"))
	if dest == "" {
		t.Skip("set AHDCODE_DOGFOOD_DIR")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(localdev.HomeEnvKey, os.Getenv("AHDCODE_LOCAL_HOME"))
	if os.Getenv("AHDCODE_LOCAL_HOME") == "" {
		t.Setenv(localdev.HomeEnvKey, t.TempDir())
	}
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
	_ = os.Unsetenv("AHDCODE_ROOT")
	var out bytes.Buffer
	err := Web(dest, &out, &out, Options{
		Starter:       StarterAdmin,
		AppName:       "Dogfood App",
		Database:      DriverSQLite,
		DatabaseName:  "app",
		AdminName:     "QA Admin",
		AdminEmail:    "qa@example.com",
		AdminPassword: "qa-only-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out.String())
}

func TestWriteMVCHTTPProject(t *testing.T) {
	dest := strings.TrimSpace(os.Getenv("AHDCODE_MVC_HTTP_DIR"))
	if dest == "" {
		t.Skip("set AHDCODE_MVC_HTTP_DIR")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
	_ = os.Unsetenv("AHDCODE_ROOT")
	var out bytes.Buffer
	err := Web(dest, &out, &out, Options{
		Starter:       StarterMVC,
		AppName:       "HTTP Portal",
		Database:      DriverSQLite,
		DatabaseName:  "http_portal",
		AdminName:     "QA Admin",
		AdminEmail:    "qa@example.com",
		AdminPassword: "qa-admin-pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out.String())
}

func TestWriteCRUDHTTPProject(t *testing.T) {
	dest := strings.TrimSpace(os.Getenv("AHDCODE_CRUD_HTTP_DIR"))
	if dest == "" {
		t.Skip("set AHDCODE_CRUD_HTTP_DIR")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(localdev.HomeEnvKey, t.TempDir())
	t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
	_ = os.Unsetenv("AHDCODE_ROOT")
	var out bytes.Buffer
	err := Web(dest, &out, &out, Options{
		Starter:       StarterCRUD,
		AppName:       "HTTP CRUD",
		Database:      DriverSQLite,
		DatabaseName:  "http_crud",
		AdminName:     "QA Admin",
		AdminEmail:    "qa@example.com",
		AdminPassword: "qa-admin-pass",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out.String())
}

func TestMVCAndCRUDApplicationsCompile(t *testing.T) {
	bin := strings.TrimSpace(os.Getenv("AHDCODE_BIN"))
	if bin == "" {
		t.Skip("set AHDCODE_BIN to compile generated starters")
	}
	for _, starter := range []string{StarterMVC, StarterCRUD} {
		t.Run(starter, func(t *testing.T) {
			t.Setenv(localdev.HomeEnvKey, t.TempDir())
			t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
			_ = os.Unsetenv("AHDCODE_ROOT")
			root := t.TempDir()
			var out bytes.Buffer
			err := Web(root, &out, &out, Options{
				Starter:       starter,
				AppName:       "Compile Portal",
				Database:      DriverSQLite,
				DatabaseName:  "compile_portal",
				AdminName:     "Ali Example",
				AdminEmail:    "ali@example.com",
				AdminPassword: "qa-admin-pass",
			})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bin, "build", "app.ahd")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "AHDCODE_ROOT=", "AHDCODE_LOCAL_HOME="+t.TempDir())
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("build %s: %v\n%s", starter, err, output)
			}
			footer := "Built with AhdCode · MVC Starter"
			if starter == StarterCRUD {
				footer = "Built with AhdCode · CRUD Starter"
			}
			body, _ := os.ReadFile(filepath.Join(root, "Components/Footer.ahd"))
			if !strings.Contains(string(body), footer) {
				t.Fatalf("footer missing %s", footer)
			}
		})
	}
}
