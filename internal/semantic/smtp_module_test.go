package semantic

import (
	"strings"
	"testing"
)

const smtpPreamble = "bring SMTP\nfrom SMTP bring SMTPClient\nfrom SMTP bring SMTPMessage\nfrom SMTP bring SMTPError\n\n"

func TestSMTPModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, smtpPreamble+`client: SMTPClient := SMTP.client("127.0.0.1", 2525)
client = SMTP.client("127.0.0.1", 2525, "none")
client = SMTP.client("127.0.0.1", 2525, "tls", 1)
authenticated: SMTPClient := client.withPlainAuth("qa-user", "SMTP_QA_PASSWORD_7f91_secret")
message: SMTPMessage := SMTP.message("sender@example.com", ["student@example.com"], "Hello")
message = message.withCc(["cc@example.com"])
message = message.withBcc(["bcc@example.com"])
message = message.withReplyTo("reply@example.com")
message = message.withText("Merhaba")
message = message.withHtml("<strong>Merhaba</strong>")
client.send(message)
`)
	requireSemanticClean(t, result)
}

func TestSMTPOperationsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`SMTP.client()`,
		`SMTP.client(2525, "127.0.0.1")`,
		`SMTP.message("sender@example.com")`,
		`SMTP.message("sender@example.com", "student@example.com", "Hello")`,
		`client.send()`,
		`client.withPlainAuth("qa-user")`,
		`message.withCc("cc@example.com")`,
		`message.withText()`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, smtpPreamble+
				"client: SMTPClient := SMTP.client(\"127.0.0.1\", 2525, \"none\")\n"+
				"message: SMTPMessage := SMTP.message(\"sender@example.com\", [\"student@example.com\"], \"Hello\")\n"+
				source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestSMTPTypesAreNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, smtpPreamble+"client: SMTPClient := SMTPClient()\n")
	requireSemanticFailure(t, result)
	result = analyzeWithStandardModules(t, smtpPreamble+"message: SMTPMessage := SMTPMessage()\n")
	requireSemanticFailure(t, result)
}

func TestSMTPModuleInterfaceExportsExactSurface(t *testing.T) {
	module := StandardModuleInterfaces()["SMTP"]
	if module == nil || module.ModuleID != "builtin:SMTP" {
		t.Fatalf("SMTP is not a registered builtin module: %#v", module)
	}
	wantExports := []string{"SMTPClient", "SMTPError", "SMTPMessage", "client", "message"}
	if strings.Join(module.ExportNames, ",") != strings.Join(wantExports, ",") {
		t.Fatalf("SMTP exports %v; want %v", module.ExportNames, wantExports)
	}
	signatures := map[string]string{
		"client":  "(host: String, port: Int, security: String := default, timeoutSeconds: Int := default) -> SMTPClient",
		"message": "(from: String, to: List<String>, subject: String) -> SMTPMessage",
	}
	for name, want := range signatures {
		symbol := module.Exports[name]
		if symbol == nil || symbol.Callable == nil {
			t.Fatalf("SMTP.%s is not an exported function", name)
		}
		if have := FormatSignature(symbol.Callable.Signature); have != want {
			t.Fatalf("SMTP.%s signature %q; want %q", name, have, want)
		}
	}
	errorSymbol := module.Exports["SMTPError"]
	if errorSymbol.Class == nil || errorSymbol.Class.Parent == nil || errorSymbol.Class.Parent.Name != "Error" {
		t.Fatalf("SMTPError does not derive from Error: %#v", errorSymbol.Class)
	}
}
