package semantic

import "testing"

func TestCSVModuleHasExactSignatures(t *testing.T) {
	preamble := "bring CSV\nfrom CSV bring CSVError\n\n"
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"parse rows", `rows: List<List<String>> := CSV.parse("a,b")`, true},
		{"parse custom delimiter", `rows: List<List<String>> := CSV.parse("a;b", ";")`, true},
		{"stringify rows", `text: String := CSV.stringify([["a", "b"]])`, true},
		{"read rows", `rows: List<List<String>> := CSV.read("data.csv")`, true},
		{"write rows", `CSV.write("data.csv", [["a"]])`, true},
		{"parse records", `rows: List<Pair<String, String>> := CSV.parseRecords("name\nAli")`, true},
		{"read records", `rows: List<Pair<String, String>> := CSV.readRecords("data.csv")`, true},
		{"stringify records", `text: String := CSV.stringifyRecords([{"name": "Ali"}])`, true},
		{"write records", `CSV.writeRecords("data.csv", [{"name": "Ali"}])`, true},
		{"CSVError is Error", `failure: Error := CSVError(message: "bad")`, true},
		{"parse rejects non-string input", `rows: List<List<String>> := CSV.parse(1)`, false},
		{"rows stay strings", `rows: List<List<Int>> := CSV.parse("1")`, false},
		{"delimiter is string", `rows: List<List<String>> := CSV.parse("a", 1)`, false},
		{"records have string values", `rows: List<Pair<String, Int>> := CSV.parseRecords("a\n1")`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, preamble+test.text)
			if test.ok {
				requireSemanticClean(t, result)
			} else if !result.HasErrors() {
				t.Fatal("expected a CSV diagnostic")
			}
		})
	}
}
