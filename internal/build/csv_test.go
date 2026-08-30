package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCSVStandardLibraryRunsAsNativeExecutables(t *testing.T) {
	preamble := "bring CSV\nfrom CSV bring CSVError\n\n"
	cases := []program{
		{
			name: "raw parse preserves strings quoting unicode and variable widths",
			sources: map[string]string{"main.ahd": preamble + `rows: List<List<String>> := CSV.parse("name,note\nAli,\"hello, dünya\"\nsolo\n")
write(rows[0][0])
write(rows[1][1])
write(len(rows[2]))
`},
			expected: "name\nhello, dünya\n1\n",
		},
		{
			name: "custom delimiter and escaped quotes round trip",
			sources: map[string]string{"main.ahd": preamble + `rows: List<List<String>> := [["a;b", "say \"hi\""], ["", "line1\nline2"]]
text: String := CSV.stringify(rows, ";")
again: List<List<String>> := CSV.parse(text, ";")
write(again[0][0])
write(again[0][1])
write(again[1][0] == "")
write(again[1][1] == "line1\nline2")
`},
			expected: "a;b\nsay \"hi\"\ntrue\ntrue\n",
		},
		{
			name: "tab delimiter and CRLF input",
			sources: map[string]string{"main.ahd": preamble + `rows: List<List<String>> := CSV.parse("name\tcity\r\nAli\tİstanbul\r\n", "\t")
write(rows[1][0])
write(rows[1][1])
`},
			expected: "Ali\nİstanbul\n",
		},
		{
			name: "records use headers and first-record key order",
			sources: map[string]string{"main.ahd": preamble + `records: List<Pair<String, String>> := CSV.parseRecords("name,age\nAli,42\nAda,36\n")
write(records[0]["name"])
write(records[1]["age"])
text: String := CSV.stringifyRecords([{"name": "Ali", "age": "42"}, {"age": "36", "name": "Ada"}])
write(text == "name,age\nAli,42\nAda,36\n")
`},
			expected: "Ali\n36\ntrue\n",
		},
		{
			name: "empty and header-only records are empty",
			sources: map[string]string{"main.ahd": preamble + `write(len(CSV.parseRecords("")))
write(len(CSV.parseRecords("name,age\n")))
empty: List<Pair<String, String>> := []
write(CSV.stringifyRecords(empty) == "")
`},
			expected: "0\n0\ntrue\n",
		},
		{
			name: "malformed CSV is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.parse("a,\"broken")
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "invalid delimiter is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.parse("a,b", "::")
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "duplicate record header is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.parseRecords("name,name\nAli,Ada\n")
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "empty record header is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.parseRecords("name,\nAli,42\n")
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "record width mismatch is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.parseRecords("name,age\nAli\n")
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "record key mismatch is CSVError",
			sources: map[string]string{"main.ahd": preamble + `CSV.stringifyRecords([{"name": "Ali"}, {"age": "42"}])
`},
			exitCode: 1, errorClass: "CSVError",
		},
		{
			name: "file access failure remains FileError",
			sources: map[string]string{"main.ahd": preamble + `CSV.read("definitely-missing-v011.csv")
`},
			exitCode: 1, errorClass: "FileError",
		},
		{
			name: "CSV file failure remains catchable as IOError",
			sources: map[string]string{"main.ahd": preamble + `attempt {
    CSV.read("definitely-missing-v011.csv")
}
except IOError as error {
    write("caught IO")
}
`},
			expected: "caught IO\n",
		},
	}
	runProgramCases(t, cases)
}

func TestCSVFileAPIsRoundTripAsNativeCode(t *testing.T) {
	directory := t.TempDir()
	rawPath := filepath.Join(directory, "raw.csv")
	recordPath := filepath.Join(directory, "records.csv")
	source := fmt.Sprintf(`bring CSV

CSV.write(%q, [["name", "note"], ["Ali", "hello, world"]])
rows: List<List<String>> := CSV.read(%q)
write(rows[1][1])

CSV.writeRecords(%q, [{"name": "Ali", "city": "İstanbul"}])
records: List<Pair<String, String>> := CSV.readRecords(%q)
write(records[0]["city"])
`, rawPath, rawPath, recordPath, recordPath)
	entryDirectory := writeSources(t, map[string]string{"main.ahd": source})
	out, errorOutput, code := buildAndRun(t, filepath.Join(entryDirectory, "main.ahd"), "")
	if code != 0 || out != "hello, world\nİstanbul\n" {
		t.Fatalf("CSV file round trip: stdout %q, stderr %q, exit %d", out, errorOutput, code)
	}
	if content, err := os.ReadFile(rawPath); err != nil || string(content) != "name,note\nAli,\"hello, world\"\n" {
		t.Fatalf("raw CSV file = %q, %v", content, err)
	}
	if content, err := os.ReadFile(recordPath); err != nil || string(content) != "name,city\nAli,İstanbul\n" {
		t.Fatalf("record CSV file = %q, %v", content, err)
	}
}
