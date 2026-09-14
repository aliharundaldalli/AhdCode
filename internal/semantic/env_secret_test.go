package semantic

import "testing"

func TestEnvSecretIsNullableString(t *testing.T) {
	requireSemanticClean(t, analyzeWithStandardModules(t, envPreamble+`password: String? := Env.secret("DB_PASSWORD")
if password != null {
    write(password)
}
`))
	for _, source := range []string{
		`password: String := Env.secret("DB_PASSWORD")`,
		`Env.secret()`,
		`Env.secret(1)`,
		`Env.secret("A", "B")`,
		`Env.secretOr("A", "B")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, envPreamble+source+"\n"))
	}
}
