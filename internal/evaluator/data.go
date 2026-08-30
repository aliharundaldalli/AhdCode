package evaluator

import (
	"os"
	"sort"
	"strconv"

	"ahdcode/internal/ir"
)

// The Data standard module's REPL implementation. It mirrors the native
// backend's runtime helpers exactly: a Table is an ordered column list plus one
// row of String cells per record, every operation is pure, and every value
// handed back to the session is a fresh copy.

const dataTableClassID = ir.ClassID("builtin:Data::class::Table")

var (
	dataColumnsField = ir.FieldID(string(dataTableClassID) + "::field::columns")
	dataCellsField   = ir.FieldID(string(dataTableClassID) + "::field::cells")
)

// dataTable is the working shape one Table operation reads and writes.
type dataTable struct {
	columns []string
	cells   [][]string
}

// tableOf reads the two hidden storage fields of a Table instance. The stored
// Lists are never handed out, so a caller cannot reach them to mutate.
func (session *Session) tableOf(value any) dataTable {
	instance := session.requireInstance(value)
	stored, ok := instance.Fields[dataColumnsField].(*List)
	if !ok {
		session.raise("DataError", "value is not a Table")
	}
	table := dataTable{}
	for _, name := range stored.Items {
		table.columns = append(table.columns, name.(string))
	}
	grid, ok := instance.Fields[dataCellsField].(*List)
	if !ok {
		session.raise("DataError", "value is not a Table")
	}
	for _, row := range grid.Items {
		cells := session.requireList(row)
		line := make([]string, len(cells.Items))
		for index, cell := range cells.Items {
			line[index] = cell.(string)
		}
		table.cells = append(table.cells, line)
	}
	return table
}

// tableValue materializes a validated table as a new Table instance.
func (session *Session) tableValue(table dataTable) *Instance {
	columns := &List{Items: make([]any, len(table.columns))}
	for index, name := range table.columns {
		columns.Items[index] = name
	}
	grid := &List{Items: make([]any, len(table.cells))}
	for index, row := range table.cells {
		cells := &List{Items: make([]any, len(row))}
		for position, cell := range row {
			cells.Items[position] = cell
		}
		grid.Items[index] = cells
	}
	return &Instance{Class: dataTableClassID, Fields: map[ir.FieldID]any{
		dataColumnsField: columns, dataCellsField: grid,
	}}
}

func (session *Session) dataRequireSchema(columns []string) {
	seen := make(map[string]bool, len(columns))
	for _, name := range columns {
		if name == "" {
			session.raise("DataError", "column name is empty")
		}
		if seen[name] {
			session.raise("DataError", "duplicate column "+strconv.Quote(name))
		}
		seen[name] = true
	}
}

func (session *Session) dataColumnPosition(columns []string, name string) int {
	for position, column := range columns {
		if column == name {
			return position
		}
	}
	session.raise("DataError", "Table has no column "+strconv.Quote(name))
	return -1
}

func (session *Session) dataRequireWidth(columns []string, cells [][]string) {
	for number, row := range cells {
		if len(row) != len(columns) {
			session.raise("DataError", "row "+strconv.Itoa(number)+" has "+
				strconv.Itoa(len(row))+" cell(s); the table has "+
				strconv.Itoa(len(columns))+" column(s)")
		}
	}
}

// dataRecord builds one row snapshot as an ordinary Pair.
func dataRecord(columns, row []string) *Pair {
	record := &Pair{Values: make(map[any]any, len(columns))}
	for position, name := range columns {
		record.Keys = append(record.Keys, name)
		record.Values[name] = row[position]
	}
	return record
}

func (session *Session) dataStrings(value any) []string {
	list := session.requireList(value)
	result := make([]string, len(list.Items))
	for index, item := range list.Items {
		result[index] = item.(string)
	}
	return result
}

// dataBuiltin dispatches the Data module's factory functions.
func (session *Session) dataBuiltin(name string, arguments []any) any {
	delimiter := func(index int) string {
		if index >= len(arguments) || arguments[index] == nil {
			return ","
		}
		return arguments[index].(string)
	}
	switch name {
	case "fromRows":
		columns := session.dataStrings(arguments[0])
		session.dataRequireSchema(columns)
		grid := session.requireList(arguments[1])
		var cells [][]string
		for _, row := range grid.Items {
			cells = append(cells, session.dataStrings(row))
		}
		session.dataRequireWidth(columns, cells)
		return session.tableValue(dataTable{columns: columns, cells: cells})
	case "fromRecords":
		return session.dataFromRecords(arguments[0])
	case "fromCSV":
		return session.dataFromGrid(session.csvRows(arguments[0].(string), delimiter(1)))
	case "readCSV":
		path := arguments[0].(string)
		content, err := os.ReadFile(path)
		if err != nil {
			session.raise("FileError", "read "+strconv.Quote(path)+" failed: "+err.Error())
		}
		return session.dataFromGrid(session.csvRows(string(content), delimiter(1)))
	}
	session.raise("Error", "unsupported Data function "+name)
	return nil
}

// dataFromRecords normalizes records into the first record's column order.
func (session *Session) dataFromRecords(value any) any {
	records := session.requireList(value)
	if len(records.Items) == 0 {
		// No record means no schema to infer; an empty Table is the only
		// honest answer, rather than inventing column names.
		return session.tableValue(dataTable{})
	}
	first := session.requirePair(records.Items[0])
	var columns []string
	for _, key := range first.Keys {
		columns = append(columns, key.(string))
	}
	session.dataRequireSchema(columns)
	var cells [][]string
	for number, item := range records.Items {
		record := session.requirePair(item)
		if len(record.Keys) != len(columns) {
			session.raise("DataError", "record "+strconv.Itoa(number)+" has "+
				strconv.Itoa(len(record.Keys))+" key(s); the first record has "+
				strconv.Itoa(len(columns)))
		}
		row := make([]string, len(columns))
		for position, name := range columns {
			cell, known := record.Values[any(name)]
			if !known {
				session.raise("DataError", "record "+strconv.Itoa(number)+
					" has no key "+strconv.Quote(name))
			}
			row[position] = cell.(string)
		}
		cells = append(cells, row)
	}
	return session.tableValue(dataTable{columns: columns, cells: cells})
}

// dataFromGrid treats the first CSV row as the header, so unlike
// CSV.parseRecords a header-only document keeps its schema.
func (session *Session) dataFromGrid(grid [][]string) any {
	if len(grid) == 0 {
		return session.tableValue(dataTable{})
	}
	columns := grid[0]
	session.dataRequireSchema(columns)
	cells := grid[1:]
	session.dataRequireWidth(columns, cells)
	return session.tableValue(dataTable{columns: columns, cells: cells})
}

// dataOperation dispatches one Table member.
func (session *Session) dataOperation(name string, receiver any, arguments []any) any {
	table := session.tableOf(receiver)
	text := func(index int, fallback string) string {
		if index >= len(arguments) || arguments[index] == nil {
			return fallback
		}
		return arguments[index].(string)
	}
	integer := func(index int, fallback int64) int64 {
		if index >= len(arguments) || arguments[index] == nil {
			return fallback
		}
		return arguments[index].(int64)
	}
	switch name {
	case "Table.rowCount":
		return int64(len(table.cells))
	case "Table.columnCount":
		return int64(len(table.columns))
	case "Table.columns":
		result := &List{Items: make([]any, len(table.columns))}
		for index, column := range table.columns {
			result.Items[index] = column
		}
		return result
	case "Table.rows":
		result := &List{Items: make([]any, len(table.cells))}
		for index, row := range table.cells {
			result.Items[index] = dataRecord(table.columns, row)
		}
		return result
	case "Table.row":
		return session.dataRow(table, integer(0, 0))
	case "Table.column":
		position := session.dataColumnPosition(table.columns, text(0, ""))
		result := &List{Items: make([]any, len(table.cells))}
		for index, row := range table.cells {
			result.Items[index] = row[position]
		}
		return result
	case "Table.head", "Table.tail":
		return session.dataSlice(name, table, integer(0, 5))
	case "Table.select":
		return session.dataSelect(table, session.dataStrings(arguments[0]))
	case "Table.drop":
		return session.dataDrop(table, session.dataStrings(arguments[0]))
	case "Table.rename":
		return session.dataRename(table, text(0, ""), text(1, ""))
	case "Table.reverse":
		result := make([][]string, len(table.cells))
		for index, row := range table.cells {
			result[len(table.cells)-1-index] = row
		}
		return session.tableValue(dataTable{columns: table.columns, cells: result})
	case "Table.filter":
		return session.dataFilter(table, arguments[0])
	case "Table.sort":
		return session.dataSort(table, arguments[0])
	case "Table.transform":
		return session.dataTransform(table, text(0, ""), arguments[1])
	case "Table.derive":
		return session.dataDerive(table, text(0, ""), arguments[1])
	case "Table.unique":
		return session.dataUnique(table, text(0, ""))
	case "Table.valueCounts":
		return session.dataValueCounts(table, text(0, ""))
	case "Table.groupBy":
		return session.dataGroupBy(table, text(0, ""))
	case "Table.pivotCount":
		return session.dataPivotCount(table, text(0, ""), text(1, ""))
	case "Table.toCSV":
		return session.csvStringify(session.dataGrid(table), text(0, ","))
	case "Table.writeCSV":
		content := session.csvStringify(session.dataGrid(table), text(1, ","))
		path := text(0, "")
		if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
			session.raise("FileError", "write "+strconv.Quote(path)+" failed: "+err.Error())
		}
		return Nothing
	}
	session.raise("Error", "unsupported Table operation "+name)
	return nil
}

func (session *Session) dataRow(table dataTable, index int64) any {
	position := index
	if position < 0 {
		position += int64(len(table.cells))
	}
	if position < 0 || position >= int64(len(table.cells)) {
		session.raise("IndexError", "row index "+strconv.FormatInt(index, 10)+" is out of range")
	}
	return dataRecord(table.columns, table.cells[position])
}

func (session *Session) dataSlice(name string, table dataTable, count int64) any {
	if count < 0 {
		operation := "head"
		if name == "Table.tail" {
			operation = "tail"
		}
		session.raise("DataError", operation+" requires a non-negative row count")
	}
	if count > int64(len(table.cells)) {
		count = int64(len(table.cells))
	}
	cells := table.cells[:count]
	if name == "Table.tail" {
		cells = table.cells[int64(len(table.cells))-count:]
	}
	return session.tableValue(dataTable{columns: table.columns, cells: cells})
}

func (session *Session) dataSelect(table dataTable, names []string) any {
	seen := make(map[string]bool, len(names))
	positions := make([]int, len(names))
	for index, name := range names {
		if seen[name] {
			session.raise("DataError", "duplicate column "+strconv.Quote(name)+" in select")
		}
		seen[name] = true
		positions[index] = session.dataColumnPosition(table.columns, name)
	}
	result := make([][]string, len(table.cells))
	for index, row := range table.cells {
		selected := make([]string, len(positions))
		for target, position := range positions {
			selected[target] = row[position]
		}
		result[index] = selected
	}
	return session.tableValue(dataTable{columns: names, cells: result})
}

func (session *Session) dataDrop(table dataTable, names []string) any {
	removed := make(map[string]bool, len(names))
	for _, name := range names {
		if removed[name] {
			session.raise("DataError", "duplicate column "+strconv.Quote(name)+" in drop")
		}
		session.dataColumnPosition(table.columns, name)
		removed[name] = true
	}
	var kept []string
	var positions []int
	for position, name := range table.columns {
		if !removed[name] {
			kept = append(kept, name)
			positions = append(positions, position)
		}
	}
	result := make([][]string, len(table.cells))
	for index, row := range table.cells {
		remaining := make([]string, len(positions))
		for target, position := range positions {
			remaining[target] = row[position]
		}
		result[index] = remaining
	}
	return session.tableValue(dataTable{columns: kept, cells: result})
}

func (session *Session) dataRename(table dataTable, oldName, newName string) any {
	position := session.dataColumnPosition(table.columns, oldName)
	if newName == "" {
		session.raise("DataError", "column name is empty")
	}
	if newName != oldName {
		for _, name := range table.columns {
			if name == newName {
				session.raise("DataError", "duplicate column "+strconv.Quote(newName))
			}
		}
	}
	renamed := append([]string(nil), table.columns...)
	renamed[position] = newName
	return session.tableValue(dataTable{columns: renamed, cells: table.cells})
}

func (session *Session) dataFilter(table dataTable, callback any) any {
	keep := callback.(*FunctionValue)
	var result [][]string
	for _, row := range table.cells {
		verdict := session.invoke(keep, []argumentValue{{value: dataRecord(table.columns, row)}})
		if session.boolean(verdict) {
			result = append(result, row)
		}
	}
	return session.tableValue(dataTable{columns: table.columns, cells: result})
}

// dataSort orders by a column name or by an Int, Real, or String key
// Function. The key runs exactly once per row, before any comparison.
func (session *Session) dataSort(table dataTable, argument any) any {
	order := make([]int, len(table.cells))
	for index := range order {
		order[index] = index
	}
	if name, isColumn := argument.(string); isColumn {
		position := session.dataColumnPosition(table.columns, name)
		sort.SliceStable(order, func(left, right int) bool {
			return table.cells[order[left]][position] < table.cells[order[right]][position]
		})
	} else {
		key := argument.(*FunctionValue)
		keys := make([]any, len(table.cells))
		for index, row := range table.cells {
			keys[index] = session.invoke(key, []argumentValue{{value: dataRecord(table.columns, row)}})
			if keys[index] == nil {
				session.raise("NullError", "sort key Function returned null")
			}
		}
		sort.SliceStable(order, func(left, right int) bool {
			return orderedLess(keys[order[left]], keys[order[right]])
		})
	}
	result := make([][]string, len(table.cells))
	for index, original := range order {
		result[index] = table.cells[original]
	}
	return session.tableValue(dataTable{columns: table.columns, cells: result})
}

func (session *Session) dataTransform(table dataTable, name string, callback any) any {
	position := session.dataColumnPosition(table.columns, name)
	convert := callback.(*FunctionValue)
	result := make([][]string, len(table.cells))
	for index, row := range table.cells {
		replaced := append([]string(nil), row...)
		computed := session.invoke(convert, []argumentValue{{value: row[position]}})
		if computed == nil {
			session.raise("NullError", "transform Function returned null")
		}
		replaced[position] = computed.(string)
		result[index] = replaced
	}
	return session.tableValue(dataTable{columns: table.columns, cells: result})
}

func (session *Session) dataDerive(table dataTable, name string, callback any) any {
	if name == "" {
		session.raise("DataError", "column name is empty")
	}
	for _, column := range table.columns {
		if column == name {
			session.raise("DataError", "column "+strconv.Quote(name)+
				" already exists; use transform to rewrite an existing column")
		}
	}
	build := callback.(*FunctionValue)
	result := make([][]string, len(table.cells))
	for index, row := range table.cells {
		computed := session.invoke(build, []argumentValue{{value: dataRecord(table.columns, row)}})
		if computed == nil {
			session.raise("NullError", "derive Function returned null")
		}
		result[index] = append(append([]string(nil), row...), computed.(string))
	}
	return session.tableValue(dataTable{
		columns: append(append([]string(nil), table.columns...), name), cells: result,
	})
}

func (session *Session) dataUnique(table dataTable, name string) any {
	position := session.dataColumnPosition(table.columns, name)
	seen := make(map[string]bool, len(table.cells))
	result := &List{}
	for _, row := range table.cells {
		if !seen[row[position]] {
			seen[row[position]] = true
			result.Items = append(result.Items, row[position])
		}
	}
	return result
}

func (session *Session) dataValueCounts(table dataTable, name string) any {
	position := session.dataColumnPosition(table.columns, name)
	counts := &Pair{Values: make(map[any]any)}
	for _, row := range table.cells {
		key := any(row[position])
		if current, known := counts.Values[key]; known {
			counts.Values[key] = current.(int64) + 1
			continue
		}
		counts.Keys = append(counts.Keys, key)
		counts.Values[key] = int64(1)
	}
	return counts
}

// dataGroupBy partitions rows, keeping first-occurrence key order and source
// row order inside each group.
func (session *Session) dataGroupBy(table dataTable, name string) any {
	position := session.dataColumnPosition(table.columns, name)
	groups := &Pair{Values: make(map[any]any)}
	collected := make(map[string][][]string)
	for _, row := range table.cells {
		key := row[position]
		if _, known := collected[key]; !known {
			groups.Keys = append(groups.Keys, any(key))
		}
		collected[key] = append(collected[key], row)
	}
	for _, key := range groups.Keys {
		groups.Values[key] = session.tableValue(dataTable{
			columns: table.columns, cells: collected[key.(string)],
		})
	}
	return groups
}

// dataGrid renders the header followed by the data rows, the shape the CSV
// writer serializes.
func (session *Session) dataGrid(table dataTable) *List {
	if len(table.columns) == 0 && len(table.cells) == 0 {
		return &List{}
	}
	grid := &List{}
	header := &List{Items: make([]any, len(table.columns))}
	for index, name := range table.columns {
		header.Items[index] = name
	}
	grid.Items = append(grid.Items, header)
	for _, row := range table.cells {
		cells := &List{Items: make([]any, len(row))}
		for position, cell := range row {
			cells.Items[position] = cell
		}
		grid.Items = append(grid.Items, cells)
	}
	return grid
}

// dataPivotCount is a strict count cross-tabulation. It matches the native
// helper exactly: first-occurrence order on both axes, zero for an absent
// combination, and String cells throughout.
func (session *Session) dataPivotCount(table dataTable, rowName, columnName string) any {
	rowPosition := session.dataColumnPosition(table.columns, rowName)
	columnPosition := session.dataColumnPosition(table.columns, columnName)
	if rowName == columnName {
		session.raise("DataError", "pivotCount needs two different columns; received "+
			strconv.Quote(rowName)+" twice")
	}
	var rowOrder, columnOrder []string
	seenRow := make(map[string]bool)
	seenColumn := make(map[string]bool)
	counts := make(map[string]map[string]int64)
	for _, row := range table.cells {
		rowKey, columnKey := row[rowPosition], row[columnPosition]
		if !seenRow[rowKey] {
			seenRow[rowKey] = true
			rowOrder = append(rowOrder, rowKey)
			counts[rowKey] = make(map[string]int64)
		}
		if !seenColumn[columnKey] {
			seenColumn[columnKey] = true
			columnOrder = append(columnOrder, columnKey)
		}
		counts[rowKey][columnKey]++
	}
	schema := append([]string{rowName}, columnOrder...)
	result := make([][]string, 0, len(rowOrder))
	for _, rowKey := range rowOrder {
		line := make([]string, 0, len(schema))
		line = append(line, rowKey)
		for _, columnKey := range columnOrder {
			line = append(line, strconv.FormatInt(counts[rowKey][columnKey], 10))
		}
		result = append(result, line)
	}
	return session.tableValue(dataTable{columns: schema, cells: result})
}
