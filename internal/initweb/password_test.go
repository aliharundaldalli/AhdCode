package initweb

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSecretFileSurvivesInputBuffering(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "wizard-input")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.WriteString("\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveOptions(t.TempDir(), Options{
		Starter: StarterEmpty,
		AppName: "Portal",
		Input:   file,
		Output:  io.Discard,
		IsTTY:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.SecretFile != file {
		t.Fatalf("SecretFile = %v; want the original *os.File", resolved.SecretFile)
	}
	if _, ok := resolved.Input.(*bufio.Reader); !ok {
		t.Fatalf("Input type %T; want *bufio.Reader", resolved.Input)
	}
}

func TestReadSecretFromPipeDoesNotHangOrPrint(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	go func() {
		_, _ = writer.Write([]byte("s3cret-value\n"))
		_ = writer.Close()
	}()

	var out bytes.Buffer
	got, err := readSecret(bufio.NewReader(reader), &out, reader)
	if err != nil {
		t.Fatal(err)
	}
	if got != "s3cret-value" {
		t.Fatalf("secret = %q", got)
	}
	if strings.Contains(out.String(), "s3cret-value") {
		t.Fatalf("password written to output: %q", out.String())
	}
}

func TestPromptSecretUsesPreservedTerminalHandle(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "secret-terminal")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.WriteString("hidden-pass\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	options := Options{
		Input:      bufio.NewReader(file),
		Output:     io.Discard,
		SecretFile: file,
	}
	got, err := promptSecret(options, "Admin password:")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hidden-pass" {
		t.Fatalf("secret = %q", got)
	}
}

func TestWizardPasswordsAreNotPrinted(t *testing.T) {
	password := "qa-hidden-pass"
	cases := []struct {
		name    string
		options Options
		input   string
	}{
		{
			name: "admin",
			options: Options{
				IsTTY: true,
			},
			input: "3\nHidden Admin\n1\nhidden_admin\nQA Admin\nqa@example.com\n" + password + "\n" + password + "\n",
		},
		{
			name: "mvc",
			options: Options{
				IsTTY: true,
			},
			input: "4\nHidden MVC\n1\nhidden_mvc\nQA Admin\nqa@example.com\n" + password + "\n" + password + "\nn\n",
		},
		{
			name: "crud",
			options: Options{
				IsTTY: true,
			},
			input: "5\nHidden CRUD\n1\nhidden_crud\nQA Admin\nqa@example.com\n" + password + "\n" + password + "\nn\n",
		},
		{
			name: "smtp",
			options: Options{
				IsTTY: true,
			},
			input: "4\nHidden Mail\n1\nhidden_mail\nQA Admin\nqa@example.com\n" + password + "\n" + password + "\ny\nsmtp.example.com\n587\nmailer\n" + password + "\nmailer@example.com\n\n\n",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("AHDCODE_LOCAL_HOME", t.TempDir())
			t.Setenv("AHDCODE_STUDIO_CACHE", t.TempDir())
			_ = os.Unsetenv("AHDCODE_ROOT")
			var out bytes.Buffer
			test.options.Input = strings.NewReader(test.input)
			test.options.Output = &out
			if err := Web(t.TempDir(), &out, io.Discard, test.options); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), password) {
				t.Fatalf("password appeared in wizard output:\n%s", out.String())
			}
		})
	}
}
