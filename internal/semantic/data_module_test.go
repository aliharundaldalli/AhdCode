package semantic

import "testing"

// requireSemanticFailure asserts that analysis rejected the program, without
// pinning which diagnostic code did it.
func requireSemanticFailure(t *testing.T, result Result) {
	t.Helper()
	if !result.HasErrors() {
		t.Fatal("expected a semantic diagnostic, but the program was accepted")
	}
}

const dataPreamble = "bring Data\nfrom Data bring Table\nfrom Data bring DataError\n\n"

func TestDataModuleValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromRows(["name", "score"], [["Ali", "91"]])
records: Table := Data.fromRecords([{"name": "Ali"}])
parsed: Table := Data.fromCSV("name\nAli\n")
custom: Table := Data.fromCSV("name;x\nAli;1\n", ";")

count: Int := table.rowCount()
width: Int := table.columnCount()
names: List<String> := table.columns()
every: List<Pair<String, String>> := table.rows()
first: Pair<String, String> := table.row(0)
scores: List<String> := table.column("score")

top: Table := table.head()
bottom: Table := table.tail(2)
picked: Table := table.select(["name"])
without: Table := table.drop(["score"])
renamed: Table := table.rename("score", "points")
flipped: Table := table.reverse()

kept: Table := table.filter(lambda (row: Pair<String, String>) -> row["name"] != "")
byColumn: Table := table.sort("name")
byKey: Table := table.sort(lambda (row: Pair<String, String>) -> -int(row["score"]))
cleaned: Table := table.transform("name", lambda (value: String) -> value.trim())
labelled: Table := table.derive("flag", lambda (row: Pair<String, String>) -> str(row["name"] == "Ali"))

distinct: List<String> := table.unique("name")
counts: Pair<String, Int> := table.valueCounts("name")
groups: Pair<String, Table> := table.groupBy("name")

text: String := table.toCSV()
delimited: String := table.toCSV(";")
`)
	requireSemanticClean(t, result)
}

// TestDataCallbackContractsAreCheckedStatically keeps a wrong callback a
// compile-time diagnostic rather than a runtime surprise.
func TestDataCallbackContractsAreCheckedStatically(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"filter must return Bool", `table.filter(lambda (row: Pair<String, String>) -> row["name"])`},
		{"filter parameter must be a row", `table.filter(lambda (value: String) -> true)`},
		{"transform must return String", `table.transform("name", lambda (value: String) -> len(value))`},
		{"transform parameter must be String", `table.transform("name", lambda (row: Pair<String, String>) -> "x")`},
		{"derive must return String", `table.derive("flag", lambda (row: Pair<String, String>) -> true)`},
		{"derive parameter must be a row", `table.derive("flag", lambda (value: String) -> value)`},
		{"sort key must be Int Real or String", `table.sort(lambda (row: Pair<String, String>) -> row["name"] == "")`},
		{"sort rejects an unrelated value", `table.sort(5)`},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, dataPreamble+
				"table: Table := Data.fromCSV(\"name,score\\nAli,91\\n\")\n"+testCase.text+"\n")
			requireSemanticFailure(t, result)
		})
	}
}

func TestDataOperationWrongArgumentCount(t *testing.T) {
	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromCSV("name\nAli\n")
table.rowCount(1)
`)
	requireSemanticCode(t, result, codeCallArguments)
}

func TestDataOperationWrongArgumentType(t *testing.T) {
	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromCSV("name\nAli\n")
table.column(5)
`)
	requireSemanticFailure(t, result)
}

func TestDataUnknownMemberIsRejected(t *testing.T) {
	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromCSV("name\nAli\n")
table.describe()
`)
	requireSemanticFailure(t, result)
}

// TestDataTableIsNotConstructedDirectly documents that a Table comes only from
// the Data factories, and that the diagnostic says so.
func TestDataTableIsNotConstructedDirectly(t *testing.T) {
	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Table()
`)
	requireSemanticCode(t, result, codeCallArguments)
}

// TestDataStorageIsNotAPublishedAttribute keeps the Table's hidden storage
// unreadable, so immutability cannot be bypassed by reaching the backing List.
func TestDataStorageIsNotAPublishedAttribute(t *testing.T) {
	for _, member := range []string{"columns", "cells"} {
		result := analyzeWithStandardModules(t, dataPreamble+
			"table: Table := Data.fromCSV(\"name\\nAli\\n\")\nwrite(table."+member+")\n")
		requireSemanticFailure(t, result)
	}
}

// TestDataCellsStayStrings pins the no-inference rule: a numeric-looking cell
// is still a String and needs an explicit conversion.
func TestDataCellsStayStrings(t *testing.T) {
	clean := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromCSV("score\n91\n")
value: String := table.row(0)["score"]
number: Int := int(table.row(0)["score"])
`)
	requireSemanticClean(t, clean)

	result := analyzeWithStandardModules(t, dataPreamble+`table: Table := Data.fromCSV("score\n91\n")
number: Int := table.row(0)["score"]
`)
	requireSemanticFailure(t, result)
}
