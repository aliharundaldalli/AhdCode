package build

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDataStandardLibraryRunsAsNativeExecutables exercises the Data table
// layer end to end through the Go backend. Every cell stays a String, every
// operation is pure, and the CSV module supplies the transport.
func TestDataStandardLibraryRunsAsNativeExecutables(t *testing.T) {
	preamble := "bring Data\nfrom Data bring Table\nfrom Data bring DataError\n\n"
	sample := `"name,department,score\nAli,Math,91\nAyse,Physics,78\nMehmet,Math,84\nZeynep,Physics,95\n"`
	cases := []program{
		{
			name: "fromRows builds a table and reports its shape",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromRows(["name", "score"], [["Ali", "91"], ["Ayse", "78"]])
write(table.rowCount())
write(table.columnCount())
write(table.columns())
write(table.column("score"))
`},
			expected: "2\n2\n[\"name\", \"score\"]\n[\"91\", \"78\"]\n",
		},
		{
			name: "fromRows accepts zero rows and keeps the schema",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromRows(["name", "score"], [])
write(table.rowCount())
write(table.columns())
`},
			expected: "0\n[\"name\", \"score\"]\n",
		},
		{
			name: "fromRecords normalizes later records into first-record order",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromRecords([{"b": "2", "a": "1"}, {"a": "3", "b": "4"}])
write(table.columns())
write(table.rows())
`},
			expected: "[\"b\", \"a\"]\n[{\"b\": \"2\", \"a\": \"1\"}, {\"b\": \"4\", \"a\": \"3\"}]\n",
		},
		{
			name: "empty records produce an empty zero-column table",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromRecords([])
write(table.columnCount())
write(table.rowCount())
`},
			expected: "0\n0\n",
		},
		{
			name: "header-only CSV preserves the schema",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name,score\n")
write(table.columns())
write(table.rowCount())
write(table.toCSV())
`},
			expected: "[\"name\", \"score\"]\n0\nname,score\n\n",
		},
		{
			name: "empty CSV produces an empty table that serializes to empty text",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("")
write(table.columnCount())
write(table.rowCount())
write("[{table.toCSV()}]")
`},
			expected: "0\n0\n[]\n",
		},
		{
			name: "quoted fields embedded newlines and custom delimiters survive",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name;note\nAli;\"a;b\"\nAyse;\"line1\nline2\"\n", ";")
write(table.column("note")[0])
write(table.column("note")[1] == "line1\nline2")
write(table.toCSV(";"))
`},
			expected: "a;b\ntrue\nname;note\nAli;\"a;b\"\nAyse;\"line1\nline2\"\n\n",
		},
		{
			name: "head tail select drop rename and reverse are pure",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
write(table.head(2).column("name"))
write(table.tail(1).column("name"))
write(table.head(0).columns())
write(table.select(["score", "name"]).columns())
write(table.drop(["department"]).columns())
write(table.rename("score", "points").columns())
write(table.reverse().column("name"))
write(table.rowCount())
write(table.columns())
`},
			expected: "[\"Ali\", \"Ayse\"]\n[\"Zeynep\"]\n[\"name\", \"department\", \"score\"]\n" +
				"[\"score\", \"name\"]\n[\"name\", \"score\"]\n[\"name\", \"department\", \"points\"]\n" +
				"[\"Zeynep\", \"Mehmet\", \"Ayse\", \"Ali\"]\n4\n[\"name\", \"department\", \"score\"]\n",
		},
		{
			name: "filter accepts a named Function and a Lambda",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
strong: Function := (
    row: Pair<String, String>
) -> Bool {
    return int(row["score"]) >= 84
}
write(table.filter(strong).column("name"))
write(table.filter(lambda (row: Pair<String, String>) -> row["department"] == "Math").column("name"))
write(table.rowCount())
`},
			expected: "[\"Ali\", \"Mehmet\", \"Zeynep\"]\n[\"Ali\", \"Mehmet\"]\n4\n",
		},
		{
			name: "sort orders by column text and by Int Real and String keys",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
write(table.sort("name").column("name"))
write(table.sort(lambda (row: Pair<String, String>) -> -int(row["score"])).column("name"))
write(table.sort(lambda (row: Pair<String, String>) -> real(row["score"])).column("name"))
write(table.sort(lambda (row: Pair<String, String>) -> row["department"]).column("name"))
write(table.column("name"))
`},
			expected: "[\"Ali\", \"Ayse\", \"Mehmet\", \"Zeynep\"]\n" +
				"[\"Zeynep\", \"Ali\", \"Mehmet\", \"Ayse\"]\n" +
				"[\"Ayse\", \"Mehmet\", \"Ali\", \"Zeynep\"]\n" +
				"[\"Ali\", \"Mehmet\", \"Ayse\", \"Zeynep\"]\n" +
				"[\"Ali\", \"Ayse\", \"Mehmet\", \"Zeynep\"]\n",
		},
		{
			name: "sort is stable for duplicate keys and runs the key once per row",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
calls: Int := 0
countingKey: Function := (
    row: Pair<String, String>
) -> String {
    calls: Global Int
    calls = calls + 1
    return row["department"]
}
write(table.sort(countingKey).column("name"))
write(calls)
`},
			expected: "[\"Ali\", \"Mehmet\", \"Ayse\", \"Zeynep\"]\n4\n",
		},
		{
			name: "transform and derive rewrite and append columns",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
upper: Table := table.transform("name", lambda (value: String) -> value.upper())
write(upper.column("name"))
write(upper.columns())
labelled: Table := table.derive("passed", lambda (row: Pair<String, String>) -> str(int(row["score"]) >= 85))
write(labelled.columns())
write(labelled.column("passed"))
write(table.columns())
write(table.column("name"))
`},
			expected: "[\"ALI\", \"AYSE\", \"MEHMET\", \"ZEYNEP\"]\n[\"name\", \"department\", \"score\"]\n" +
				"[\"name\", \"department\", \"score\", \"passed\"]\n" +
				"[\"true\", \"false\", \"false\", \"true\"]\n" +
				"[\"name\", \"department\", \"score\"]\n[\"Ali\", \"Ayse\", \"Mehmet\", \"Zeynep\"]\n",
		},
		{
			name: "unique valueCounts and groupBy keep first-occurrence order",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
write(table.unique("department"))
write(table.valueCounts("department"))
groups: Pair<String, Table> := table.groupBy("department")
write(len(groups))
write(groups["Math"].column("name"))
write(groups["Physics"].column("name"))
write(groups["Math"].columns())
`},
			expected: "[\"Math\", \"Physics\"]\n{\"Math\": 2, \"Physics\": 2}\n2\n" +
				"[\"Ali\", \"Mehmet\"]\n[\"Ayse\", \"Zeynep\"]\n[\"name\", \"department\", \"score\"]\n",
		},
		{
			name: "grouping an empty table produces empty results",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name,score\n")
write(len(table.groupBy("name")))
write(table.unique("name"))
write(len(table.valueCounts("name")))
`},
			expected: "0\n[]\n0\n",
		},
		{
			name: "row uses List index rules and rows snapshots every record",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
write(table.row(0)["name"])
write(table.row(-1)["name"])
write(len(table.rows()))
write(table.rows()[1]["department"])
`},
			expected: "Ali\nZeynep\n4\nPhysics\n",
		},
		{
			name: "returned collections are snapshots that cannot reach the table",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
columns: List<String> := table.columns()
columns.add("injected")
row: Pair<String, String> := table.row(0)
row["name"] = "CHANGED"
rows: List<Pair<String, String>> := table.rows()
rows[0]["name"] = "MUTATED"
values: List<String> := table.column("score")
values.add("999")
write(table.columns())
write(table.row(0)["name"])
write(table.column("score"))
`},
			expected: "[\"name\", \"department\", \"score\"]\nAli\n[\"91\", \"78\", \"84\", \"95\"]\n",
		},
		{
			name: "cells stay String and convert only when asked",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
write(type(table))
write(type(table.row(0)["score"]))
write(int(table.row(0)["score"]) + 1)
scores: List<Real> := table.column("score").map(lambda (value: String) -> real(value))
write(scores[0])
`},
			expected: "Table\nString\n92\n91.0\n",
		},
		{
			name: "toCSV round trips through fromCSV",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV(` + sample + `)
again: Table := Data.fromCSV(table.toCSV())
write(again.columns())
write(again.rowCount())
write(again.toCSV() == table.toCSV())
`},
			expected: "[\"name\", \"department\", \"score\"]\n4\ntrue\n",
		},
		{
			name: "structural mistakes raise DataError with actionable text",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name,score\nAli,91\n")
attempt { Data.fromRows(["a", "a"], []) } except DataError as error { write(error.message) }
attempt { Data.fromRows([""], []) } except DataError as error { write(error.message) }
attempt { Data.fromRows(["a", "b"], [["1"]]) } except DataError as error { write(error.message) }
attempt { table.column("age") } except DataError as error { write(error.message) }
attempt { table.select(["name", "name"]) } except DataError as error { write(error.message) }
attempt { table.drop(["age"]) } except DataError as error { write(error.message) }
attempt { table.rename("name", "score") } except DataError as error { write(error.message) }
attempt { table.head(-1) } except DataError as error { write(error.message) }
attempt { table.tail(-1) } except DataError as error { write(error.message) }
attempt { table.derive("name", lambda (row: Pair<String, String>) -> "x") } except DataError as error { write(error.message) }
attempt { Data.fromRecords([{"a": "1"}, {"b": "2"}]) } except DataError as error { write(error.message) }
`},
			expected: "duplicate column \"a\"\ncolumn name is empty\n" +
				"row 0 has 1 cell(s); the table has 2 column(s)\n" +
				"Table has no column \"age\"\nduplicate column \"name\" in select\n" +
				"Table has no column \"age\"\nduplicate column \"score\"\n" +
				"head requires a non-negative row count\ntail requires a non-negative row count\n" +
				"column \"name\" already exists; use transform to rewrite an existing column\n" +
				"record 1 has no key \"a\"\n",
		},
		{
			name: "an invalid row index is the ordinary IndexError",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name\nAli\n")
attempt { table.row(9) } except IndexError as error { write("index") }
`},
			expected: "index\n",
		},
		{
			name: "CSV and filesystem failures keep their own error types",
			sources: map[string]string{"main.ahd": preamble + `from CSV bring CSVError
from File bring FileError

table: Table := Data.fromCSV("name\nAli\n")
attempt { Data.fromCSV("a,\"unfinished") } except CSVError as error { write("csv") }
attempt { table.toCSV("!!") } except CSVError as error { write("delimiter") }
attempt { Data.readCSV("no_such_data_file.csv") } except FileError as error { write("file") }
`},
			expected: "csv\ndelimiter\nfile\n",
		},
		{
			name: "a failing callback leaves the source table untouched",
			sources: map[string]string{"main.ahd": preamble + `table: Table := Data.fromCSV("name,score\nAli,91\nAyse,notanumber\n")
attempt {
    table.filter(lambda (row: Pair<String, String>) -> int(row["score"]) > 0)
} except Error as error {
    write("callback failed")
}
write(table.rowCount())
write(table.column("name"))
`},
			expected: "callback failed\n2\n[\"Ali\", \"Ayse\"]\n",
		},
	}
	runProgramCases(t, cases)
}

// TestDataReadsAndWritesCSVFilesRelativeToTheProcess covers the filesystem
// pair, which the in-memory cases cannot reach.
func TestDataReadsAndWritesCSVFilesRelativeToTheProcess(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "people.csv")
	if err := os.WriteFile(path, []byte("name,score\nAli,91\nAyse,78\n"), 0o600); err != nil {
		t.Fatalf("could not seed the CSV: %v", err)
	}
	output := filepath.Join(directory, "written.csv")
	runProgramCases(t, []program{{
		name: "readCSV and writeCSV round trip a real file",
		sources: map[string]string{"main.ahd": "bring Data\nfrom Data bring Table\n\n" +
			"table: Table := Data.readCSV(\"" + path + "\")\n" +
			"write(table.columns())\n" +
			"write(table.rowCount())\n" +
			"table.filter(lambda (row: Pair<String, String>) -> int(row[\"score\"]) > 80).writeCSV(\"" + output + "\")\n" +
			"again: Table := Data.readCSV(\"" + output + "\")\n" +
			"write(again.rowCount())\n" +
			"write(again.row(0)[\"name\"])\n"},
		expected: "[\"name\", \"score\"]\n2\n1\nAli\n",
	}})
}

// TestRejectedLeadingDotChainProducesOneDiagnostic is the end-to-end guard for
// PAR013 recovery. A leading-dot continuation is still invalid; the point is
// that one malformed chain explains itself once, without a bracket cascade and
// without a derived complaint about the receiver the parser had to truncate.
func TestRejectedLeadingDotChainProducesOneDiagnostic(t *testing.T) {
	source := `entries: List<Int> := [1, 2]

labels: List<String> := entries
    .filter(
        lambda (value: Int) -> value > 1
    )
    .map(
        lambda (value: Int) -> str(value)
    )

for label in labels {
    write(label)
}
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	path, result := BuildProgram(filepath.Join(directory, "main.ahd"), filepath.Join(t.TempDir(), "program"))
	if path != "" || !result.HasErrors() {
		t.Fatal("a leading-dot continuation must still fail the build")
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "PAR013" {
		t.Fatalf("diagnostics = %+v, want exactly one PAR013", result.Diagnostics)
	}
}

// TestTablePivotCountRunsNatively covers the strict count cross-tabulation.
// It is deliberately not a general pivot: no aggregation callback, no value
// column, no missing-data model, and every cell stays a String.
func TestTablePivotCountRunsNatively(t *testing.T) {
	preamble := "bring Data\nfrom Data bring Table\nfrom Data bring DataError\n\n"
	sample := `"name,department,grade\nAli,Math,A\nAyse,Physics,B\nMehmet,Math,A\nZeynep,Physics,A\n"`
	runProgramCases(t, []program{
		{
			name: "counts cross-tabulate in first-occurrence order on both axes",
			sources: map[string]string{"main.ahd": preamble + `students: Table := Data.fromCSV(` + sample + `)
pivoted: Table := students.pivotCount("department", "grade")
write(pivoted.columns())
write(pivoted.toCSV())
`},
			expected: "[\"department\", \"A\", \"B\"]\ndepartment,A,B\nMath,2,0\nPhysics,1,1\n\n",
		},
		{
			name: "an absent combination counts zero and stays a String",
			sources: map[string]string{"main.ahd": preamble + `students: Table := Data.fromCSV(` + sample + `)
pivoted: Table := students.pivotCount("department", "grade")
write(pivoted.row(0)["B"])
write(type(pivoted.row(0)["B"]))
write(int(pivoted.row(0)["A"]) + 1)
`},
			expected: "0\nString\n3\n",
		},
		{
			name: "the source Table is unchanged",
			sources: map[string]string{"main.ahd": preamble + `students: Table := Data.fromCSV(` + sample + `)
students.pivotCount("department", "grade")
write(students.columns())
write(students.rowCount())
`},
			expected: "[\"name\", \"department\", \"grade\"]\n4\n",
		},
		{
			name: "empty and header-only tables keep well-defined shapes",
			sources: map[string]string{"main.ahd": preamble + `header: Table := Data.fromCSV("a,b\n")
write(header.pivotCount("a", "b").columns())
write(header.pivotCount("a", "b").rowCount())
write("[{header.pivotCount("a", "b").toCSV()}]")
`},
			expected: "[\"a\"]\n0\n[a\n]\n",
		},
		{
			name: "Unicode category values survive a CSV round trip",
			sources: map[string]string{"main.ahd": preamble + `source: Table := Data.fromCSV("k,v\nşehir,İstanbul\nşehir,Ankara\nköy,Ankara\n")
text: String := source.pivotCount("k", "v").toCSV()
write(text)
write(Data.fromCSV(text).columns())
`},
			expected: "k,İstanbul,Ankara\nşehir,1,1\nköy,0,1\n\n[\"k\", \"İstanbul\", \"Ankara\"]\n",
		},
		{
			name: "unknown or repeated columns raise DataError naming the column",
			sources: map[string]string{"main.ahd": preamble + `students: Table := Data.fromCSV(` + sample + `)
attempt { students.pivotCount("nope", "grade") } except DataError as error { write(error.message) }
attempt { students.pivotCount("department", "nope") } except DataError as error { write(error.message) }
attempt { students.pivotCount("grade", "grade") } except DataError as error { write(error.message) }
`},
			expected: "Table has no column \"nope\"\nTable has no column \"nope\"\n" +
				"pivotCount needs two different columns; received \"grade\" twice\n",
		},
	})
}
