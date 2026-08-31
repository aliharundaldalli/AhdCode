package semantic

import "testing"

const envPreamble = "bring Env\nfrom Env bring EnvError\n\n"

func TestEnvModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, envPreamble+`found: String? := Env.get("PATH")
value: String := Env.getOr("PATH", "default")
present: Bool := Env.exists("PATH")
Env.set("NAME", "value")
Env.unset("NAME")
record: Pair<String, String> := Env.read(".env")
Env.load(".env")
Env.load(".env", true)
`)
	requireSemanticClean(t, result)
}

func TestEnvGetResultIsNullableWithoutGuard(t *testing.T) {
	result := analyzeWithStandardModules(t, envPreamble+`found: String := Env.get("PATH")
`)
	requireSemanticFailure(t, result)
}

func TestEnvFunctionsRejectWrongArityAndTypes(t *testing.T) {
	tests := []string{
		`Env.get()`,
		`Env.get(1)`,
		`Env.getOr("A")`,
		`Env.exists()`,
		`Env.set("A")`,
		`Env.unset()`,
		`Env.read()`,
		`Env.load()`,
		`Env.load(".env", "not-a-bool")`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			result := analyzeWithStandardModules(t, envPreamble+source+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestEnvErrorCatchable(t *testing.T) {
	result := analyzeWithStandardModules(t, envPreamble+`attempt {
    Env.set("", "x")
} except EnvError as error {
    write(error.message)
}
`)
	requireSemanticClean(t, result)
}
