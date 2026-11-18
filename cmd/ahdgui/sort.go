package main

import (
	"fmt"
	"sort"
)

// TableView rows are ordered by one column at a time. Comparison is Go's
// string order: Unicode code points, case-sensitive, not numeric and not
// locale-aware, so "10" is before "2" and "" is before any other text.
// Equal cells keep their current relative order. A header click sorts
// ascending, and a second click on the same header sorts descending; a
// click on another header sorts that column ascending. The selected row
// stays the same row and moves to its new index. Neither a header click
// nor sortBy counts as a selection change.

// tableCell is one cell, or "" when the row has no such column.
func tableCell(row []string, column int) string {
	if column < 0 || column >= len(row) {
		return ""
	}
	return row[column]
}

// orderRows stably sorts a table by its current column. The caller holds
// m.mu. A column of -1 leaves the rows as they are.
func (m *model) orderRows(target *widget) {
	column := target.sortColumn
	if column < 0 {
		return
	}
	if column >= len(target.columns) {
		target.sortColumn = -1
		target.sortDesc = false
		return
	}
	rows := target.rows
	order := make([]int, len(rows))
	for index := range order {
		order[index] = index
	}
	descending := target.sortDesc
	sort.SliceStable(order, func(i, j int) bool {
		left := tableCell(rows[order[i]], column)
		right := tableCell(rows[order[j]], column)
		if left == right {
			return false
		}
		if descending {
			return left > right
		}
		return left < right
	})
	next := make([][]string, len(rows))
	selected := -1
	for newIndex, oldIndex := range order {
		next[newIndex] = rows[oldIndex]
		if oldIndex == target.selected {
			selected = newIndex
		}
	}
	target.rows = next
	if target.selected >= 0 {
		target.selected = selected
	}
}

// sortBy orders a table for the program. It does not notify onSelect.
func (m *model) sortBy(id int64, column int, descending bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return errNoWidget
	}
	if target.kind != kindTable {
		return errWrongKind
	}
	if column < 0 || column >= len(target.columns) {
		return fmt.Errorf("column %d is outside the %d columns", column, len(target.columns))
	}
	target.sortColumn = column
	target.sortDesc = descending
	m.orderRows(target)
	m.reveal(target)
	m.dirty = true
	return nil
}

// sortHeader orders a table the way a header click does. A missing widget
// or a disabled table does nothing.
func (m *model) sortHeader(id int64, column int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return
	}
	m.userSort(target, column)
}

// userSort orders a table from a header click. The caller holds m.mu.
// A disabled table ignores the click. The click does not notify onSelect.
func (m *model) userSort(target *widget, column int) {
	if target.kind != kindTable || target.disabled {
		return
	}
	if column < 0 || column >= len(target.columns) {
		return
	}
	if target.sortColumn == column {
		target.sortDesc = !target.sortDesc
	} else {
		target.sortColumn = column
		target.sortDesc = false
	}
	m.orderRows(target)
	m.reveal(target)
	m.sortNotice = target.id
	m.dirty = true
}

// headerColumn is the table column under a logical point, or -1. The
// point has to be in the header, inside the body width, and on a column
// that is not scrolled out of that width. The caller holds m.mu.
func (m *model) headerColumn(target *widget, x, y int) int {
	if target.kind != kindTable {
		return -1
	}
	v := m.viewportOf(target)
	if y < target.y+1 || y >= target.y+1+headerHeight || x < v.x || x >= v.x+v.w {
		return -1
	}
	left := v.x - target.scrollX
	for index, width := range target.widths {
		right := left + width
		if right > v.x && left < v.x+v.w && x >= left && x < right {
			return index
		}
		left = right
	}
	return -1
}

// takeSort returns and forgets the table the user sorted from a header.
func (m *model) takeSort() (int64, int, bool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.sortNotice
	m.sortNotice = 0
	if id == 0 {
		return 0, 0, false, false
	}
	target := m.widgets[id]
	if target == nil || target.sortColumn < 0 {
		return 0, 0, false, false
	}
	return id, target.sortColumn, target.sortDesc, true
}

// rowsNow and sortNow are the table's current rows and sort, for a get.
func (m *model) rowsNow(id int64) ([][]string, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return nil, 0, errNoWidget
	}
	if target.kind != kindTable {
		return nil, 0, errWrongKind
	}
	return copyRows(target.rows), target.selected, nil
}

func (m *model) sortNow(id int64) (int, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.widgets[id]
	if target == nil {
		return 0, false, errNoWidget
	}
	if target.kind != kindTable {
		return 0, false, errWrongKind
	}
	if target.sortColumn < 0 {
		return -1, false, nil
	}
	return target.sortColumn, target.sortDesc, nil
}
