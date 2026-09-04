package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// D. Development derives the local identity by appending .test; nothing else
// about the host is rewritten.
func TestDevelopmentURLAppendsTest(t *testing.T) {
	environment := webEnvironment{host: "ahdakademi.com", protocol: "https"}
	if environment.developmentHost() != "ahdakademi.com.test" {
		t.Errorf("development host was %q", environment.developmentHost())
	}
	if environment.developmentURL() != "https://ahdakademi.com.test" {
		t.Errorf("development URL was %q", environment.developmentURL())
	}
}

// E. A production configuration is refused by the development command rather
// than run under development semantics or silently rewritten.
func TestProductionConfigurationIsRefusedByDev(t *testing.T) {
	err := checkWebEnvironment(webEnvironment{environment: "production", host: "example.com", protocol: "https"})
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
		if checkWebEnvironment(webEnvironment{environment: environment}) != nil {
			t.Errorf("ahdcode dev refused APP_ENV=%q", environment)
		}
	}
}

// F. The banner leads with the canonical development URL, which is the line
// the user actually needs.
func TestWebBannerShowsTheDevelopmentURL(t *testing.T) {
	var out bytes.Buffer
	announceWebApplication(&out, webEnvironment{
		name: "Ahd Akademi", environment: "development",
		host: "ahdakademi.com", protocol: "http",
	})
	printed := out.String()
	if !strings.Contains(printed, "http://ahdakademi.com.test") {
		t.Errorf("the banner did not show the development URL:\n%s", printed)
	}
	if !strings.Contains(printed, "Ahd Akademi (development)") {
		t.Errorf("the banner did not name the application and environment:\n%s", printed)
	}
}

// G. An https development URL is explained, never downgraded. The banner must
// not offer http as though it were what was configured.
func TestHTTPSNoticeExplainsWithoutDowngrading(t *testing.T) {
	var out bytes.Buffer
	announceWebApplication(&out, webEnvironment{
		name: "Ahd Akademi", environment: "development",
		host: "ahdakademi.com", protocol: "https",
		serverHost: "127.0.0.1", serverPort: "8080",
	})
	printed := out.String()
	if !strings.Contains(printed, "https://ahdakademi.com.test") {
		t.Errorf("the https development URL was not shown:\n%s", printed)
	}
	if !strings.Contains(printed, "APP_PROTOCOL was not changed") {
		t.Errorf("the notice did not state that APP_PROTOCOL is untouched:\n%s", printed)
	}
	if !strings.Contains(printed, "127.0.0.1:8080") {
		t.Errorf("the notice did not name the address a proxy would forward to:\n%s", printed)
	}
	if strings.Contains(printed, "http://ahdakademi.com.test") {
		t.Errorf("the notice offered a silent http downgrade:\n%s", printed)
	}
}

// H. An http development URL needs no notice at all.
func TestHTTPDevelopmentPrintsNoTLSNotice(t *testing.T) {
	if lines := localHTTPSNotice(webEnvironment{protocol: "http", host: "example.com"}); lines != nil {
		t.Errorf("expected no TLS notice for http, received %#v", lines)
	}
}
