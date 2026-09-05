package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ahdcode/internal/localdev"
)

// A. The .env reader dev uses accepts the same KEY=value grammar the Env
// module accepts, and executes nothing.
func TestParseDevEnvFileReadsPlainAssignments(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	content := "# comment\n\nAPP_NAME=Ahd Akademi\nAPP_ENV=development\n" +
		"APP_HOST=\"quoted.example\"\nAPP_PROTOCOL='https'\n" +
		"NOT A KEY=ignored\nBARE_LINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	values := parseDevEnvFile(path)
	for key, expected := range map[string]string{
		"APP_NAME": "Ahd Akademi", "APP_ENV": "development",
		"APP_HOST": "quoted.example", "APP_PROTOCOL": "https",
	} {
		if values[key] != expected {
			t.Errorf("%s was %q, expected %q", key, values[key], expected)
		}
	}
	if _, present := values["BARE_LINE"]; present {
		t.Error("a line with no assignment produced a value")
	}
}

// B. A missing .env is not an error: the process environment alone is a
// complete configuration.
func TestParseDevEnvFileToleratesAMissingFile(t *testing.T) {
	if values := parseDevEnvFile(filepath.Join(t.TempDir(), ".env")); len(values) != 0 {
		t.Errorf("expected no values from a missing .env, received %#v", values)
	}
}

// C. A variable already exported by the process wins over the app-root .env,
// which is what makes a container or CI override predictable.
func TestReadWebEnvironmentPrefersTheProcessEnvironment(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "app.ahd")
	content := "APP_NAME=From File\nAPP_ENV=development\nAPP_HOST=file.example\nAPP_PROTOCOL=http\n"
	if err := os.WriteFile(filepath.Join(directory, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_NAME", "From Process")

	environment := readWebEnvironment(entry)
	if environment.name != "From Process" {
		t.Errorf("APP_NAME was %q; the process environment should win", environment.name)
	}
	if environment.host != "file.example" {
		t.Errorf("APP_HOST was %q; .env should fill in what the process does not set", environment.host)
	}
}

// D. Development derives the local identity from APP_HOST by replacing the
// registrable suffix with .test, so the local name reads as the same project
// rather than as the production name with something appended.
func TestDevelopmentURLDerivesTestName(t *testing.T) {
	environment := webEnvironment{host: "ahdakademi.com", protocol: "https"}
	if environment.developmentHost() != "ahdakademi.test" {
		t.Errorf("development host was %q", environment.developmentHost())
	}
	if environment.developmentURL() != "https://ahdakademi.test" {
		t.Errorf("development URL was %q", environment.developmentURL())
	}
}

// E. A production configuration is refused by the development command rather
// than run under development semantics or silently rewritten.
func TestProductionConfigurationIsRefusedByDev(t *testing.T) {
	err := checkWebEnvironment(webEnvironment{environment: "production", host: "example.com", protocol: "http"})
	if err == nil {
		t.Fatal("ahdcode dev accepted a production configuration")
	}
	message := err.Error()
	for _, expected := range []string{"APP_ENV is production", "development command", "was not changed"} {
		if !strings.Contains(message, expected) {
			t.Errorf("the refusal did not mention %q: %s", expected, message)
		}
	}
	for _, environment := range []string{"development", "test", ""} {
		if checkWebEnvironment(webEnvironment{environment: environment, protocol: "http"}) != nil {
			t.Errorf("ahdcode dev refused APP_ENV=%q over http", environment)
		}
	}
}

// F. APP_PROTOCOL=https is refused outright. dev serves plaintext HTTP, so
// starting the child would mean running http while every URL derived from the
// configuration says https.
func TestHTTPSConfigurationIsRefusedByDev(t *testing.T) {
	err := checkWebEnvironment(webEnvironment{
		environment: "development", host: "ahdakademi.com", protocol: "https",
		serverHost: "127.0.0.1", serverPort: "8080",
	})
	if err == nil {
		t.Fatal("ahdcode dev accepted APP_PROTOCOL=https and would have served plaintext http")
	}
	message := err.Error()
	for _, expected := range []string{
		"Local HTTPS is not available",
		"https://ahdakademi.test",
		"APP_PROTOCOL=http",
		"127.0.0.1:8080",
		"Nothing was started and APP_PROTOCOL was not changed",
	} {
		if !strings.Contains(message, expected) {
			t.Errorf("the refusal did not mention %q: %s", expected, message)
		}
	}
	// The command it points at must exist. v0.15 ships no trust tooling.
	if strings.Contains(message, "ahdcode trust") {
		t.Errorf("the refusal pointed at a command that does not exist: %s", message)
	}
}

// G. https is refused in every environment dev can run, not just development:
// the child binds plaintext in all of them.
func TestHTTPSIsRefusedInEveryDevEnvironment(t *testing.T) {
	for _, environment := range []string{"development", "test"} {
		err := checkWebEnvironment(webEnvironment{
			environment: environment, host: "example.com", protocol: "https",
			serverHost: "127.0.0.1", serverPort: "8080",
		})
		if err == nil {
			t.Errorf("APP_ENV=%s with https was accepted", environment)
		}
	}
}

// H. The banner leads with the address that actually works, built from the
// configured socket rather than a hardcoded loopback.
func TestWebBannerLeadsWithTheWorkingAddress(t *testing.T) {
	var out bytes.Buffer
	announceWebApplication(&out, webEnvironment{
		name: "Ahd Akademi", environment: "development",
		host: "ahdakademi.com", protocol: "http",
		serverHost: "127.0.0.1", serverPort: "8137",
	}, devLocalRoute{
		route:      localdev.Route{Hostname: "ahdakademi.test", BindHost: "127.0.0.1", BindPort: 8137},
		routerPort: 80,
	})
	printed := out.String()
	if !strings.Contains(printed, "Open:") || !strings.Contains(printed, "http://127.0.0.1:8137") {
		t.Errorf("the banner did not lead with the working address:\n%s", printed)
	}
	openIndex := strings.Index(printed, "http://127.0.0.1:8137")
	identityIndex := strings.Index(printed, "http://ahdakademi.test/")
	if identityIndex < 0 {
		t.Fatalf("the local identity was dropped entirely:\n%s", printed)
	}
	if openIndex > identityIndex {
		t.Errorf("the logical URL was printed before the working address:\n%s", printed)
	}
	if !strings.Contains(printed, "Bind: 127.0.0.1:8137") {
		t.Errorf("the bind address was not reported alongside the logical URL:\n%s", printed)
	}
}

// H2. A session that could not obtain a route still reports the application
// as running, and says plainly that the name is not routed.
func TestWebBannerReportsAnUnroutedIdentity(t *testing.T) {
	var out bytes.Buffer
	announceWebApplication(&out, webEnvironment{
		name: "Ahd Akademi", environment: "development",
		host: "ahdakademi.com", protocol: "http",
		serverHost: "127.0.0.1", serverPort: "8137",
	}, devLocalRoute{note: "no local router port was available on this machine"})
	printed := out.String()
	if !strings.Contains(printed, "http://127.0.0.1:8137") {
		t.Errorf("the working address was dropped when routing failed:\n%s", printed)
	}
	if !strings.Contains(printed, "not routed") {
		t.Errorf("an unrouted identity was not marked as such:\n%s", printed)
	}
}

// I. The working address follows SERVER_HOST, and a wildcard bind is shown as
// the loopback a browser can actually open.
func TestWorkingAddressFollowsServerHost(t *testing.T) {
	for _, testCase := range []struct{ host, port, expected string }{
		{"127.0.0.1", "8080", "http://127.0.0.1:8080"},
		{"192.168.1.20", "3000", "http://192.168.1.20:3000"},
		{"0.0.0.0", "8080", "http://127.0.0.1:8080"},
		{"::1", "8080", "http://[::1]:8080"},
	} {
		environment := webEnvironment{
			environment: "development", host: "example.com", protocol: "http",
			serverHost: testCase.host, serverPort: testCase.port,
		}
		if got := environment.openURL(); got != testCase.expected {
			t.Errorf("SERVER_HOST=%s SERVER_PORT=%s produced %q, expected %q",
				testCase.host, testCase.port, got, testCase.expected)
		}
	}
}

// J. APP_ENV=test uses APP_HOST unchanged, exactly as AppConfig does, so the
// banner must not advertise a .test development identity for it.
func TestTestEnvironmentPrintsNoDevelopmentIdentity(t *testing.T) {
	var out bytes.Buffer
	announceWebApplication(&out, webEnvironment{
		name: "Ahd Akademi", environment: "test",
		host: "ahdakademi.com", protocol: "http",
		serverHost: "127.0.0.1", serverPort: "8137",
	}, devLocalRoute{})
	printed := out.String()
	if strings.Contains(printed, "ahdakademi.test") {
		t.Errorf("APP_ENV=test advertised a .test development identity:\n%s", printed)
	}
	if strings.Contains(printed, "Local identity") {
		t.Errorf("APP_ENV=test printed a local identity block:\n%s", printed)
	}
	if !strings.Contains(printed, "http://127.0.0.1:8137") {
		t.Errorf("APP_ENV=test did not report the bind address:\n%s", printed)
	}
}
