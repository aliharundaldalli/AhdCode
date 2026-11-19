package migrate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

type schemaTrigger struct {
	name   string
	table  string
	schema string
}

type probeTarget struct {
	schema string
	table  string
}

// compileTriggers authorizes every trigger SQLite will keep after this
// migration. CREATE TRIGGER does not run the authorizer on the body. An
// EXPLAIN of a statement that would fire the trigger does: SQLite compiles
// the body, the authorizer sees it, and the trigger subprogram is marked.
// Coverage is the Init instruction at address 0 whose p4 is exactly
// "-- TRIGGER <name>". That layout is a VDBE convention of the bundled
// SQLite build, not a stable public SQLite API. A trigger that never appears
// there is rejected. Prepare errors and string literals are not proof that
// the body is safe. The marker has no schema, so two triggers whose names
// match under SQLite's ASCII case folding cannot prove which body was compiled.
func compileTriggers(conn *sqlite3.Conn, reason *string) error {
	triggers, err := listTriggers(conn)
	if err != nil {
		return err
	}
	if len(triggers) == 0 {
		return nil
	}
	if err := rejectAmbiguousTriggerNames(triggers); err != nil {
		return err
	}
	targets, err := triggerTargets(conn, triggers)
	if err != nil {
		return err
	}
	covered := map[string]struct{}{}
	var probeErr error
	for _, target := range targets {
		columns, err := assignableColumns(conn, target)
		if err != nil {
			return err
		}
		probes := []string{dmlProbe("DELETE FROM ", target, nil)}
		if len(columns) > 0 {
			probes = append(probes,
				dmlProbe("INSERT", target, columns),
				dmlProbe("UPDATE", target, columns),
			)
		}
		// rowid, oid, and _rowid_ are separate UPDATE OF names on a rowid
		// table. Assigning ordinary columns does not compile those bodies.
		probes = append(probes, rowidProbes(target)...)
		for _, probe := range probes {
			names, err := explainTriggerNames(conn, probe)
			if *reason != "" {
				if err != nil {
					return err
				}
				return fmt.Errorf("trigger authorization failed")
			}
			if err != nil {
				probeErr = err
				continue
			}
			for _, name := range names {
				trigger, ok, err := matchTrigger(triggers, name)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("SQLite compiled trigger %s, which is not in the schema list; trigger coverage cannot be established", name)
				}
				covered[trigger.schema+"\x00"+trigger.name] = struct{}{}
			}
		}
	}
	var missing []string
	for _, trigger := range triggers {
		if _, ok := covered[trigger.schema+"\x00"+trigger.name]; ok {
			continue
		}
		missing = append(missing, trigger.schema+"."+trigger.name)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		if probeErr != nil {
			return fmt.Errorf("trigger %s could not be compiled under the authorizer: %s", missing[0], probeErr.Error())
		}
		return fmt.Errorf("trigger %s could not be compiled under the authorizer", missing[0])
	}
	return nil
}

// sqliteASCIIFold matches SQLite's NOCASE folding: ASCII A-Z only.
func sqliteASCIIFold(name string) string {
	folded := []byte(name)
	for index, char := range folded {
		if char >= 'A' && char <= 'Z' {
			folded[index] = char + ('a' - 'A')
		}
	}
	return string(folded)
}

func rejectAmbiguousTriggerNames(triggers []schemaTrigger) error {
	grouped := map[string][]schemaTrigger{}
	var order []string
	for _, trigger := range triggers {
		key := sqliteASCIIFold(trigger.name)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], trigger)
	}
	for _, key := range order {
		group := grouped[key]
		if len(group) < 2 {
			continue
		}
		labels := make([]string, len(group))
		for index, trigger := range group {
			labels[index] = trigger.schema + "." + trigger.name
		}
		sort.Strings(labels)
		return fmt.Errorf("trigger %s cannot be validated because %s use the same name and the program marker does not include the schema", group[0].name, strings.Join(labels, " and "))
	}
	return nil
}

func matchTrigger(triggers []schemaTrigger, marker string) (schemaTrigger, bool, error) {
	var found []schemaTrigger
	for _, trigger := range triggers {
		if trigger.name == marker || sqliteASCIIFold(trigger.name) == sqliteASCIIFold(marker) {
			found = append(found, trigger)
		}
	}
	switch len(found) {
	case 0:
		return schemaTrigger{}, false, nil
	case 1:
		return found[0], true, nil
	default:
		return schemaTrigger{}, false, fmt.Errorf("trigger %s cannot be validated because that name exists in more than one schema", marker)
	}
}

func listTriggers(conn *sqlite3.Conn) ([]schemaTrigger, error) {
	var triggers []schemaTrigger
	for _, schema := range []string{"main", "temp"} {
		found, err := triggersIn(conn, schema)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, found...)
	}
	return triggers, nil
}

func triggersIn(conn *sqlite3.Conn, schema string) ([]schemaTrigger, error) {
	statement, _, err := conn.Prepare(`SELECT name, tbl_name FROM ` + schema + `.sqlite_master WHERE type = 'trigger'`)
	if err != nil {
		if schema == "temp" && strings.Contains(err.Error(), "no such table") {
			return nil, nil
		}
		return nil, fmt.Errorf("could not list %s triggers: %s", schema, err.Error())
	}
	defer statement.Close()
	var triggers []schemaTrigger
	for statement.Step() {
		if statement.ColumnType(0) != sqlite3.TEXT || statement.ColumnType(1) != sqlite3.TEXT {
			return nil, fmt.Errorf("a %s trigger row is unreadable; its body was not authorized", schema)
		}
		name := statement.ColumnText(0)
		table := statement.ColumnText(1)
		if name == "" || table == "" {
			return nil, fmt.Errorf("a %s trigger row is unreadable; its body was not authorized", schema)
		}
		triggers = append(triggers, schemaTrigger{name: name, table: table, schema: schema})
	}
	if err := statement.Err(); err != nil {
		return nil, fmt.Errorf("could not list %s triggers: %s", schema, err.Error())
	}
	return triggers, nil
}

func triggerTargets(conn *sqlite3.Conn, triggers []schemaTrigger) ([]probeTarget, error) {
	seen := map[string]probeTarget{}
	var ordered []probeTarget
	add := func(target probeTarget) {
		key := target.schema + "\x00" + target.table
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = target
		ordered = append(ordered, target)
	}
	for _, trigger := range triggers {
		found := 0
		for _, schema := range []string{"main", "temp"} {
			name, ok, err := lookupRelation(conn, schema, trigger.table)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			found++
			add(probeTarget{schema: schema, table: name})
		}
		if found == 0 {
			return nil, fmt.Errorf("trigger %s refers to %s, which is not a table or view", trigger.name, trigger.table)
		}
	}
	return ordered, nil
}

func lookupRelation(conn *sqlite3.Conn, schema, name string) (string, bool, error) {
	statement, _, err := conn.Prepare(`SELECT name FROM ` + schema + `.sqlite_master WHERE type IN ('table', 'view') AND name = ? COLLATE NOCASE`)
	if err != nil {
		if schema == "temp" && strings.Contains(err.Error(), "no such table") {
			return "", false, nil
		}
		return "", false, fmt.Errorf("could not look up %s.%s: %s", schema, name, err.Error())
	}
	defer statement.Close()
	if err := statement.BindText(1, name); err != nil {
		return "", false, err
	}
	if !statement.Step() {
		return "", false, statement.Err()
	}
	if statement.ColumnType(0) != sqlite3.TEXT || statement.ColumnText(0) == "" {
		return "", false, fmt.Errorf("%s.%s has an unreadable sqlite_master name", schema, name)
	}
	stored := statement.ColumnText(0)
	if statement.Step() {
		return "", false, fmt.Errorf("%s.%s matches more than one table or view", schema, name)
	}
	return stored, true, statement.Err()
}

func assignableColumns(conn *sqlite3.Conn, target probeTarget) ([]string, error) {
	literal := "'" + strings.ReplaceAll(target.table, "'", "''") + "'"
	statement, _, err := conn.Prepare(`SELECT name, hidden FROM ` + target.schema + `.pragma_table_xinfo(` + literal + `)`)
	if err != nil {
		return nil, fmt.Errorf("could not read columns of %s.%s: %s", target.schema, target.table, err.Error())
	}
	defer statement.Close()
	var columns []string
	seen := map[string]struct{}{}
	for statement.Step() {
		if statement.ColumnType(0) != sqlite3.TEXT || statement.ColumnType(1) != sqlite3.INTEGER {
			return nil, fmt.Errorf("columns of %s.%s are unreadable", target.schema, target.table)
		}
		name := statement.ColumnText(0)
		// 0 is an ordinary column. 1 is a hidden column such as rowid.
		// 2 and 3 are generated columns, which cannot be assigned.
		if statement.ColumnInt64(1) != 0 || name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		columns = append(columns, name)
	}
	if err := statement.Err(); err != nil {
		return nil, fmt.Errorf("could not read columns of %s.%s: %s", target.schema, target.table, err.Error())
	}
	return columns, nil
}

func dmlProbe(kind string, target probeTarget, columns []string) string {
	qualified := quoteIdent(target.schema) + "." + quoteIdent(target.table)
	switch kind {
	case "DELETE FROM ":
		return "DELETE FROM " + qualified + " WHERE 0"
	case "INSERT":
		list := joinIdents(columns)
		return "INSERT INTO " + qualified + " (" + list + ") SELECT " + list + " FROM " + qualified + " WHERE 0"
	default:
		assignments := make([]string, len(columns))
		for index, column := range columns {
			quoted := quoteIdent(column)
			assignments[index] = quoted + " = " + quoted
		}
		return "UPDATE " + qualified + " SET " + strings.Join(assignments, ", ") + " WHERE 0"
	}
}

func rowidProbes(target probeTarget) []string {
	qualified := quoteIdent(target.schema) + "." + quoteIdent(target.table)
	return []string{
		"UPDATE " + qualified + " SET rowid = rowid WHERE 0",
		"UPDATE " + qualified + " SET oid = oid WHERE 0",
		`UPDATE ` + qualified + ` SET "_rowid_" = "_rowid_" WHERE 0`,
	}
}

// explainColumns is the VDBE EXPLAIN row produced by the bundled SQLite.
// A different column layout means this build can no longer prove which
// trigger body was compiled, so coverage fails closed.
var explainColumns = []string{"addr", "opcode", "p1", "p2", "p3", "p4", "p5", "comment"}

const triggerMarkerPrefix = "-- TRIGGER "

func explainTriggerNames(conn *sqlite3.Conn, sql string) ([]string, error) {
	statement, _, err := conn.Prepare("EXPLAIN " + sql)
	if err != nil {
		return nil, err
	}
	if statement == nil {
		return nil, fmt.Errorf("SQLite did not compile %s", sql)
	}
	defer statement.Close()
	if statement.ColumnCount() != len(explainColumns) {
		return nil, fmt.Errorf("SQLite EXPLAIN output has %d columns; trigger coverage cannot be established", statement.ColumnCount())
	}
	for index, name := range explainColumns {
		if statement.ColumnName(index) != name {
			return nil, fmt.Errorf("SQLite EXPLAIN column %d is %q; trigger coverage cannot be established", index, statement.ColumnName(index))
		}
	}
	var names []string
	for statement.Step() {
		// Only the first instruction of a trigger subprogram counts.
		// String literals in DEFAULT, CHECK, and trigger bodies also use
		// p4, but they are not this Init instruction.
		if statement.ColumnText(0) != "0" || statement.ColumnText(1) != "Init" {
			continue
		}
		marker := statement.ColumnText(5)
		if marker == "" {
			continue
		}
		if statement.ColumnType(5) != sqlite3.TEXT || !strings.HasPrefix(marker, triggerMarkerPrefix) {
			return nil, fmt.Errorf("SQLite EXPLAIN Init marker %q is not a trigger subprogram; trigger coverage cannot be established", marker)
		}
		if statement.ColumnText(2) != "0" || statement.ColumnText(4) != "0" || statement.ColumnText(6) != "0" || statement.ColumnText(7) != "" {
			return nil, fmt.Errorf("SQLite EXPLAIN trigger marker %q does not have the expected instruction shape", marker)
		}
		name := strings.TrimPrefix(marker, triggerMarkerPrefix)
		if name == "" || strings.ContainsAny(name, "\x00\n") {
			return nil, fmt.Errorf("SQLite EXPLAIN trigger marker %q does not name one trigger", marker)
		}
		names = append(names, name)
	}
	if err := statement.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func joinIdents(names []string) string {
	quoted := make([]string, len(names))
	for index, name := range names {
		quoted[index] = quoteIdent(name)
	}
	return strings.Join(quoted, ", ")
}
