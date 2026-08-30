package evaluator

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (session *Session) csvDelimiter(text string) rune {
	if !utf8.ValidString(text) {
		session.raise("CSVError", "delimiter is not valid UTF-8")
	}
	runes := []rune(text)
	if len(runes) != 1 || runes[0] == '\r' || runes[0] == '\n' || runes[0] == '"' || runes[0] == utf8.RuneError || runes[0] == 0 {
		session.raise("CSVError", "delimiter must be exactly one valid Unicode scalar other than quote, CR, or LF")
	}
	return runes[0]
}

func (session *Session) csvRows(text, delimiter string) [][]string {
	if !utf8.ValidString(text) {
		session.raise("CSVError", "CSV text is not valid UTF-8")
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = session.csvDelimiter(delimiter)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		session.raise("CSVError", "invalid CSV: "+err.Error())
	}
	return rows
}

func evaluatorCSVList(rows [][]string) *List {
	result := &List{Items: make([]any, len(rows))}
	for index, row := range rows {
		items := make([]any, len(row))
		for field := range row {
			items[field] = row[field]
		}
		result.Items[index] = &List{Items: items}
	}
	return result
}

func (session *Session) csvStringify(value any, delimiter string) string {
	comma := session.csvDelimiter(delimiter)
	rows := session.requireList(value)
	var output strings.Builder
	writer := csv.NewWriter(&output)
	writer.Comma = comma
	for _, item := range rows.Items {
		row := session.requireList(item)
		fields := make([]string, len(row.Items))
		for index, field := range row.Items {
			fields[index] = field.(string)
		}
		if err := writer.Write(fields); err != nil {
			session.raise("CSVError", "could not encode CSV: "+err.Error())
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		session.raise("CSVError", "could not encode CSV: "+err.Error())
	}
	return output.String()
}

func (session *Session) csvParseRecords(text, delimiter string) *List {
	rows := session.csvRows(text, delimiter)
	if len(rows) == 0 {
		return &List{}
	}
	headers := rows[0]
	seen := make(map[string]bool, len(headers))
	for _, header := range headers {
		if header == "" {
			session.raise("CSVError", "record headers must not be empty")
		}
		if seen[header] {
			session.raise("CSVError", "record headers must be unique; duplicate "+strconv.Quote(header))
		}
		seen[header] = true
	}
	result := &List{Items: make([]any, 0, len(rows)-1)}
	for rowIndex, row := range rows[1:] {
		if len(row) != len(headers) {
			session.raise("CSVError", fmt.Sprintf("record row %d has %d fields; expected %d", rowIndex+2, len(row), len(headers)))
		}
		record := &Pair{Values: make(map[any]any)}
		for index, header := range headers {
			pairSet(record, header, row[index])
		}
		result.Items = append(result.Items, record)
	}
	return result
}

func (session *Session) csvStringifyRecords(value any, delimiter string) string {
	session.csvDelimiter(delimiter)
	records := session.requireList(value)
	if len(records.Items) == 0 {
		return ""
	}
	first := session.requirePair(records.Items[0])
	if len(first.Keys) == 0 {
		session.raise("CSVError", "records must contain at least one column")
	}
	headers := make([]string, len(first.Keys))
	for index, key := range first.Keys {
		headers[index] = key.(string)
	}
	rows := &List{Items: []any{stringList(headers)}}
	for recordIndex, item := range records.Items {
		record := session.requirePair(item)
		if len(record.Keys) != len(headers) {
			session.raise("CSVError", fmt.Sprintf("record %d does not have the same key set as the first record", recordIndex+1))
		}
		row := make([]string, len(headers))
		for index, header := range headers {
			value, exists := record.Values[header]
			if !exists {
				session.raise("CSVError", fmt.Sprintf("record %d is missing key %s", recordIndex+1, strconv.Quote(header)))
			}
			row[index] = value.(string)
		}
		rows.Items = append(rows.Items, stringList(row))
	}
	return session.csvStringify(rows, delimiter)
}

func stringList(values []string) *List {
	items := make([]any, len(values))
	for index := range values {
		items[index] = values[index]
	}
	return &List{Items: items}
}

func (session *Session) csvBuiltin(name string, arguments []any) any {
	delimiterIndex := map[string]int{"parse": 1, "stringify": 1, "read": 1, "write": 2,
		"parseRecords": 1, "readRecords": 1, "stringifyRecords": 1, "writeRecords": 2}[name]
	delimiter := ","
	if delimiterIndex < len(arguments) && arguments[delimiterIndex] != nil {
		delimiter = arguments[delimiterIndex].(string)
	}
	read := func(path string) string {
		content, err := os.ReadFile(session.sessionPath(path))
		if err != nil {
			session.fileError("read", path, err)
		}
		return string(content)
	}
	write := func(path, content string) {
		if err := os.WriteFile(session.sessionPath(path), []byte(content), 0o666); err != nil {
			session.fileError("write", path, err)
		}
	}
	switch name {
	case "parse":
		return evaluatorCSVList(session.csvRows(arguments[0].(string), delimiter))
	case "stringify":
		return session.csvStringify(arguments[0], delimiter)
	case "read":
		return evaluatorCSVList(session.csvRows(read(arguments[0].(string)), delimiter))
	case "write":
		write(arguments[0].(string), session.csvStringify(arguments[1], delimiter))
		return Nothing
	case "parseRecords":
		return session.csvParseRecords(arguments[0].(string), delimiter)
	case "readRecords":
		return session.csvParseRecords(read(arguments[0].(string)), delimiter)
	case "stringifyRecords":
		return session.csvStringifyRecords(arguments[0], delimiter)
	case "writeRecords":
		write(arguments[0].(string), session.csvStringifyRecords(arguments[1], delimiter))
		return Nothing
	}
	session.raise("Error", "unsupported CSV operation "+name)
	return nil
}
