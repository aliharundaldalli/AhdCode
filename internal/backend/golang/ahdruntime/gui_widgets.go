package ahdruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// GUI v2.0: ListBox, Select, TextArea, PasswordInput, TableView, change
// callbacks, resizable Windows, and dialogs
// ---------------------------------------------------------------------------
//
// These functions extend gui.go with the same conventions: every function
// reports a problem as a message that becomes a GUIError, compiled programs
// and the evaluator call the same functions, and a callback's own error
// propagates unchanged. Selections are indexes; "no selection" is null in
// AhdCode and -1 on the helper protocol. Only the user's actions call a
// change callback: select(), setItems(), setRows(), setText(), and
// setChecked() never do.

const (
	// AhdGUIMaxAreaRunes bounds a TextArea's text.
	AhdGUIMaxAreaRunes = 100_000
	// AhdGUIMaxItems bounds a ListBox's or Select's items, and
	// AhdGUIMaxItemRunes each item's text.
	AhdGUIMaxItems     = 20_000
	AhdGUIMaxItemRunes = 1024
	// AhdGUIMaxColumns, AhdGUIMaxRows, and AhdGUIMaxCells bound a TableView;
	// every header and cell is at most AhdGUIMaxItemRunes characters.
	AhdGUIMaxColumns = 64
	AhdGUIMaxRows    = 20_000
	AhdGUIMaxCells   = 200_000
	// Dialog bounds: a suggested file name, and the file extensions offered.
	ahdGUIMaxNameRunes  = 255
	ahdGUIMaxExtensions = 32
	ahdGUIMaxExtension  = 16
)

// ahdGUIAreaText normalizes a TextArea's line breaks to "\n".
func ahdGUIAreaText(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

func ahdGUIItemsProblem(items []string) string {
	if len(items) > AhdGUIMaxItems {
		return fmt.Sprintf("a list holds at most %d items", AhdGUIMaxItems)
	}
	for _, item := range items {
		if !ahdGUIValidText(item, AhdGUIMaxItemRunes) {
			return fmt.Sprintf("every item must be text of at most %d characters", AhdGUIMaxItemRunes)
		}
	}
	return ""
}

// ahdGUITableProblem checks a TableView's columns and rows: at least one
// column, every row exactly as wide as the columns, and the bounds above.
func ahdGUITableProblem(columns []string, rows [][]string) string {
	if len(columns) == 0 {
		return "a TableView needs at least one column"
	}
	if len(columns) > AhdGUIMaxColumns {
		return fmt.Sprintf("a TableView has at most %d columns", AhdGUIMaxColumns)
	}
	if len(rows) > AhdGUIMaxRows || len(rows)*len(columns) > AhdGUIMaxCells {
		return fmt.Sprintf("a TableView holds at most %d rows and %d cells", AhdGUIMaxRows, AhdGUIMaxCells)
	}
	for _, column := range columns {
		if !ahdGUIValidText(column, AhdGUIMaxItemRunes) {
			return fmt.Sprintf("every column name must be text of at most %d characters", AhdGUIMaxItemRunes)
		}
	}
	for index, row := range rows {
		if len(row) != len(columns) {
			return fmt.Sprintf("row %d has %d cells; every row needs one cell per column (%d)", index, len(row), len(columns))
		}
		for _, cell := range row {
			if !ahdGUIValidText(cell, AhdGUIMaxItemRunes) {
				return fmt.Sprintf("every cell must be text of at most %d characters", AhdGUIMaxItemRunes)
			}
		}
	}
	return ""
}

func ahdGUICopyRows(rows [][]string) [][]string {
	copied := make([][]string, len(rows))
	for index, row := range rows {
		copied[index] = append([]string{}, row...)
	}
	return copied
}

// AhdGUIAddList adds a ListBox or Select with its items and selection (-1
// for none).
func AhdGUIAddList(container int64, kind string, items []string, selected int64) (int64, string) {
	if problem := ahdGUIItemsProblem(items); problem != "" {
		return 0, problem
	}
	if selected < -1 || selected >= int64(len(items)) {
		return 0, fmt.Sprintf("selectedIndex %d is outside the %d items", selected, len(items))
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	index := int(selected)
	return ahdGUIAdd(parent.window, parent.id, kind, ahdGUIRequest{Items: append([]string{}, items...), Selected: &index})
}

// AhdGUIAddTable adds a TableView.
func AhdGUIAddTable(container int64, columns []string, rows [][]string) (int64, string) {
	if problem := ahdGUITableProblem(columns, rows); problem != "" {
		return 0, problem
	}
	parent, problem := ahdGUIContainer(container)
	if problem != "" {
		return 0, problem
	}
	return ahdGUIAdd(parent.window, parent.id, "table", ahdGUIRequest{Columns: append([]string{}, columns...), Rows: ahdGUICopyRows(rows)})
}

// ahdGUIChange sends one change of a widget's contents and records it.
func ahdGUIChange(handle int64, value ahdGUIRequest, record func(*ahdGUIWidget)) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	value.Op, value.Widget = "set", widget.id
	if _, problem := window.request(value, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	ahdGUI.Lock()
	record(widget)
	ahdGUI.Unlock()
	return ""
}

// AhdGUIItems is a ListBox's or Select's items, a copy.
func AhdGUIItems(handle int64) ([]string, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return nil, problem
	}
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	return append([]string{}, widget.items...), ""
}

// AhdGUISetItems replaces a ListBox's or Select's items; a selection that no
// longer exists is cleared.
func AhdGUISetItems(handle int64, items []string) string {
	if problem := ahdGUIItemsProblem(items); problem != "" {
		return problem
	}
	copied := append([]string{}, items...)
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "items", Items: copied}, func(widget *ahdGUIWidget) { widget.items = copied })
}

// AhdGUIColumns and AhdGUIRows are a TableView's columns and rows, copies.
func AhdGUIColumns(handle int64) ([]string, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return nil, problem
	}
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	return append([]string{}, widget.columns...), ""
}

func AhdGUIRows(handle int64) ([][]string, string) {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return nil, problem
	}
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	return ahdGUICopyRows(widget.rows), ""
}

// AhdGUISetRows replaces a TableView's rows, keeping its columns.
func AhdGUISetRows(handle int64, rows [][]string) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	ahdGUI.Lock()
	columns := widget.columns
	ahdGUI.Unlock()
	if problem := ahdGUITableProblem(columns, rows); problem != "" {
		return problem
	}
	copied := ahdGUICopyRows(rows)
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "rows", Rows: copied}, func(widget *ahdGUIWidget) { widget.rows = copied })
}

// AhdGUISelectedIndex reads a ListBox's, Select's, or TableView's
// selection; present is false for none. After the Window closed, the last
// selection is returned.
func AhdGUISelectedIndex(handle int64) (int64, bool, string) {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return 0, false, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if final, ok := window.finals[widget.id]; ok || !window.alive {
		if final.Selected == nil || *final.Selected < 0 {
			return 0, false, ""
		}
		return int64(*final.Selected), true, ""
	}
	reply, problem := window.request(ahdGUIRequest{Op: "get", Widget: widget.id}, ahdGUIRequestTimeout)
	if problem != "" {
		return 0, false, problem
	}
	if reply.Selected == nil {
		return 0, false, "the GUI window helper sent an invalid response"
	}
	if *reply.Selected < 0 {
		return 0, false, ""
	}
	return int64(*reply.Selected), true, ""
}

// AhdGUISelectedText is the text of a ListBox's or Select's selected item;
// present is false for none.
func AhdGUISelectedText(handle int64) (string, bool, string) {
	index, present, problem := AhdGUISelectedIndex(handle)
	if problem != "" || !present {
		return "", false, problem
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return "", false, problem
	}
	ahdGUI.Lock()
	defer ahdGUI.Unlock()
	if index >= int64(len(widget.items)) {
		return "", false, ""
	}
	return widget.items[index], true, ""
}

// AhdGUISelect selects an item or row for the program; present false clears
// the selection. It never calls the change callback.
func AhdGUISelect(handle int64, index int64, present bool) string {
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	ahdGUI.Lock()
	count := len(widget.items)
	if widget.kind == "table" {
		count = len(widget.rows)
	}
	ahdGUI.Unlock()
	selected := -1
	if present {
		if index < 0 || index >= int64(count) {
			return fmt.Sprintf("index %d is outside the %d entries", index, count)
		}
		selected = int(index)
	}
	return ahdGUIChange(handle, ahdGUIRequest{Kind: "selected", Selected: &selected}, func(*ahdGUIWidget) {})
}

// ahdGUIListen registers a change callback, replacing any earlier one.
func ahdGUIListen(handle int64, what string, handler func(ahdGUIEvent)) string {
	widget, window, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, known := window.changes[widget.id]; !known {
		if _, problem := window.request(ahdGUIRequest{Op: "listen", Kind: "change", Widget: widget.id}, ahdGUIRequestTimeout); problem != "" {
			return problem
		}
	}
	window.changes[widget.id] = handler
	return ""
}

// AhdGUIOnTextChange registers a TextInput's, PasswordInput's, or
// TextArea's change callback; it receives the new text.
func AhdGUIOnTextChange(handle int64, handler func(text string)) string {
	if handler == nil {
		return "onChange needs a Function"
	}
	return ahdGUIListen(handle, "text", func(event ahdGUIEvent) { handler(event.Text) })
}

// AhdGUIOnCheckChange registers a Checkbox's change callback; it receives
// the new state.
func AhdGUIOnCheckChange(handle int64, handler func(checked bool)) string {
	if handler == nil {
		return "Checkbox.onChange needs a Function"
	}
	return ahdGUIListen(handle, "checked", func(event ahdGUIEvent) { handler(event.Checked) })
}

// AhdGUIOnSelectionChange registers a ListBox's or Select's change callback;
// it receives the selected index and text, both nil for none.
func AhdGUIOnSelectionChange(handle int64, handler func(index *int64, text *string)) string {
	if handler == nil {
		return "onChange needs a Function"
	}
	widget, _, problem := ahdGUIWidgetOf(handle)
	if problem != "" {
		return problem
	}
	return ahdGUIListen(handle, "selection", func(event ahdGUIEvent) {
		if event.Selected == nil || *event.Selected < 0 {
			handler(nil, nil)
			return
		}
		index := int64(*event.Selected)
		ahdGUI.Lock()
		var text *string
		if index < int64(len(widget.items)) {
			item := widget.items[index]
			text = &item
		}
		ahdGUI.Unlock()
		handler(&index, text)
	})
}

// AhdGUIOnRowSelect registers a TableView's selection callback; it receives
// the selected row, nil for none.
func AhdGUIOnRowSelect(handle int64, handler func(row *int64)) string {
	if handler == nil {
		return "TableView.onSelect needs a Function"
	}
	return ahdGUIListen(handle, "row", func(event ahdGUIEvent) {
		if event.Selected == nil || *event.Selected < 0 {
			handler(nil)
			return
		}
		row := int64(*event.Selected)
		handler(&row)
	})
}

// AhdGUISetResizable lets the user resize the Window, or stops it.
func AhdGUISetResizable(handle int64, resizable bool) string {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	if !window.usable {
		return ahdGUIClosedMessage
	}
	if _, problem := window.request(ahdGUIRequest{Op: "resizable", Enabled: &resizable}, ahdGUIRequestTimeout); problem != "" {
		return problem
	}
	window.resizable = resizable
	return ""
}

// AhdGUIIsResizable reports Window.setResizable's last setting.
func AhdGUIIsResizable(handle int64) (bool, string) {
	window, problem := ahdGUIWindowOf(handle)
	if problem != "" {
		return false, problem
	}
	window.mu.Lock()
	defer window.mu.Unlock()
	return window.resizable, ""
}

// ---- dialogs -----------------------------------------------------------------

// Dialog results. A cancelled file dialog is not a problem: the caller
// returns null or an empty List.
type AhdGUIDialogResult struct {
	Paths     []string
	Cancelled bool
	Confirmed bool
}

// ahdGUIScripted counts the headless test answers used; a new answer list
// starts again from its first answer.
var ahdGUIScripted = struct {
	sync.Mutex
	list string
	next int
}{}

// ahdGUIExtensions checks and normalizes file extensions: letters and digits,
// with at most one leading dot dropped.
func ahdGUIExtensions(extensions []string) ([]string, string) {
	if len(extensions) > ahdGUIMaxExtensions {
		return nil, fmt.Sprintf("at most %d file extensions can be offered", ahdGUIMaxExtensions)
	}
	normalized := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		plain := strings.TrimPrefix(extension, ".")
		valid := plain != "" && len(plain) <= ahdGUIMaxExtension
		for _, character := range plain {
			if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9') {
				valid = false
			}
		}
		if !valid {
			return nil, fmt.Sprintf("%q is not a file extension; use letters and digits, such as \"csv\"", extension)
		}
		normalized = append(normalized, plain)
	}
	return normalized, ""
}

// AhdGUIDialog shows one dialog in its own helper process and waits for the
// user. kind is openFile, openFiles, selectFolder, saveFile, message, or
// confirm. Cancelling is a result, not a problem; a problem means the dialog
// could not be shown.
func AhdGUIDialog(kind, title, text, name string, extensions []string) (AhdGUIDialogResult, string) {
	if !ahdGUIValidText(title, ahdGUIMaxTitleRunes) {
		return AhdGUIDialogResult{}, fmt.Sprintf("a dialog title must be text of at most %d characters", ahdGUIMaxTitleRunes)
	}
	if !ahdGUIValidText(text, AhdGUIMaxTextRunes) {
		return AhdGUIDialogResult{}, ahdGUITextProblem("dialog text")
	}
	if name != "" && (!ahdGUIValidText(name, ahdGUIMaxNameRunes) || strings.ContainsAny(name, `/\:`) || name == "." || name == "..") {
		return AhdGUIDialogResult{}, "suggestedName must be a file name without a folder"
	}
	normalized, problem := ahdGUIExtensions(extensions)
	if problem != "" {
		return AhdGUIDialogResult{}, problem
	}
	value := ahdGUIRequest{Op: "dialog", Version: ahdGUIProtocolVersion, Kind: kind, Title: title, Text: text,
		Name: name, Extensions: normalized}
	if os.Getenv("AHDCODE_GUI_HEADLESS") == "1" {
		// Automated tests answer headless dialogs in order.
		list := os.Getenv("AHDCODE_GUI_HEADLESS_DIALOGS")
		var answers []json.RawMessage
		if json.Unmarshal([]byte(list), &answers) != nil {
			return AhdGUIDialogResult{}, "AHDCODE_GUI_HEADLESS_DIALOGS is not a JSON list of answers"
		}
		ahdGUIScripted.Lock()
		if list != ahdGUIScripted.list {
			ahdGUIScripted.list, ahdGUIScripted.next = list, 0
		}
		index := ahdGUIScripted.next
		ahdGUIScripted.next++
		ahdGUIScripted.Unlock()
		if index >= len(answers) {
			return AhdGUIDialogResult{}, "no scripted answer is left for this headless dialog"
		}
		value.Headless, value.Answer = true, answers[index]
	}
	path, err := ahdGUIDiscoverRuntime()
	if err != nil {
		return AhdGUIDialogResult{}, err.Error()
	}
	// Output written before the dialog must be visible while it is open.
	AhdFlush()
	link, err := ahdStartHelper(path, ahdGUIMaxLineBytes)
	if err != nil {
		return AhdGUIDialogResult{}, "could not start the GUI dialog helper"
	}
	defer link.shutdown(ahdGUICloseTimeout)
	// The user may take as long as they like.
	line, err := link.call(value, 0)
	if err != nil {
		return AhdGUIDialogResult{}, "the GUI dialog helper stopped unexpectedly"
	}
	var reply ahdGUIResponse
	if json.Unmarshal(line, &reply) != nil {
		return AhdGUIDialogResult{}, "the GUI dialog helper sent an invalid response"
	}
	if !reply.OK {
		if reply.Error == "" {
			reply.Error = "the dialog could not be shown"
		}
		return AhdGUIDialogResult{}, "the dialog could not be shown: " + reply.Error
	}
	result := AhdGUIDialogResult{Cancelled: reply.Cancelled || len(reply.Paths) == 0}
	if reply.Confirmed != nil {
		result.Confirmed = *reply.Confirmed
	}
	for _, chosen := range reply.Paths {
		if chosen == "" || !utf8.ValidString(chosen) || strings.ContainsRune(chosen, 0) || !filepath.IsAbs(chosen) {
			return AhdGUIDialogResult{}, "the GUI dialog helper returned an invalid path"
		}
		result.Paths = append(result.Paths, chosen)
	}
	if kind != "openFiles" && len(result.Paths) > 1 {
		return AhdGUIDialogResult{}, "the GUI dialog helper returned several paths for one"
	}
	return result, ""
}

// AhdGUIDialogPath runs a one-path dialog: the chosen path, or present false
// when the user cancelled.
func AhdGUIDialogPath(kind, title, name string, extensions []string) (string, bool, string) {
	result, problem := AhdGUIDialog(kind, title, "", name, extensions)
	if problem != "" || result.Cancelled || len(result.Paths) == 0 {
		return "", false, problem
	}
	return result.Paths[0], true, ""
}

// ---- compiled-program wrappers ----

func ahdGUIStringsOf(list *AhdList[string]) []string {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	return list.Snapshot()
}

func ahdGUIRowsOf(list *AhdList[*AhdList[string]]) [][]string {
	if list == nil {
		AhdRaiseClass(AhdClassNullError, "List value is null")
	}
	rows := list.Snapshot()
	result := make([][]string, len(rows))
	for index, row := range rows {
		result[index] = ahdGUIStringsOf(row)
	}
	return result
}

func ahdGUIRowsList(rows [][]string) *AhdList[*AhdList[string]] {
	lists := make([]*AhdList[string], len(rows))
	for index, row := range rows {
		lists[index] = AhdNewList(row...)
	}
	return AhdNewList(lists...)
}

func AhdGUIAddListChecked(container int64, kind string, items *AhdList[string], selected *int64) int64 {
	index := int64(-1)
	if selected != nil {
		index = *selected
		if index < 0 {
			AhdGUICheck(fmt.Sprintf("selectedIndex %d is outside the items", index))
		}
	}
	handle, problem := AhdGUIAddList(container, kind, ahdGUIStringsOf(items), index)
	AhdGUICheck(problem)
	return handle
}

func AhdGUIAddTableChecked(container int64, columns *AhdList[string], rows *AhdList[*AhdList[string]]) int64 {
	handle, problem := AhdGUIAddTable(container, ahdGUIStringsOf(columns), ahdGUIRowsOf(rows))
	AhdGUICheck(problem)
	return handle
}

func AhdGUIItemsChecked(handle int64) *AhdList[string] {
	items, problem := AhdGUIItems(handle)
	AhdGUICheck(problem)
	return AhdNewList(items...)
}

func AhdGUISetItemsChecked(handle int64, items *AhdList[string]) {
	AhdGUICheck(AhdGUISetItems(handle, ahdGUIStringsOf(items)))
}

func AhdGUIColumnsChecked(handle int64) *AhdList[string] {
	columns, problem := AhdGUIColumns(handle)
	AhdGUICheck(problem)
	return AhdNewList(columns...)
}

func AhdGUIRowsChecked(handle int64) *AhdList[*AhdList[string]] {
	rows, problem := AhdGUIRows(handle)
	AhdGUICheck(problem)
	return ahdGUIRowsList(rows)
}

func AhdGUISetRowsChecked(handle int64, rows *AhdList[*AhdList[string]]) {
	AhdGUICheck(AhdGUISetRows(handle, ahdGUIRowsOf(rows)))
}

func AhdGUISelectedIndexChecked(handle int64) *int64 {
	index, present, problem := AhdGUISelectedIndex(handle)
	AhdGUICheck(problem)
	if !present {
		return nil
	}
	return &index
}

func AhdGUISelectedTextChecked(handle int64) *string {
	text, present, problem := AhdGUISelectedText(handle)
	AhdGUICheck(problem)
	if !present {
		return nil
	}
	return &text
}

func AhdGUISelectChecked(handle int64, index *int64) {
	if index == nil {
		AhdGUICheck(AhdGUISelect(handle, 0, false))
		return
	}
	AhdGUICheck(AhdGUISelect(handle, *index, true))
}

func AhdGUIIsResizableChecked(handle int64) bool {
	resizable, problem := AhdGUIIsResizable(handle)
	AhdGUICheck(problem)
	return resizable
}

// AhdGUIDialogPathChecked is GUI.openFile, selectFolder, and saveFile.
func AhdGUIDialogPathChecked(kind, title, name string, extensions *AhdList[string]) *string {
	var offered []string
	if extensions != nil {
		offered = extensions.Snapshot()
	}
	path, present, problem := AhdGUIDialogPath(kind, title, name, offered)
	AhdGUICheck(problem)
	if !present {
		return nil
	}
	return &path
}

// AhdGUIOpenFilesChecked is GUI.openFiles: the chosen files, or an empty
// List when the user cancelled.
func AhdGUIOpenFilesChecked(title string, extensions *AhdList[string]) *AhdList[string] {
	var offered []string
	if extensions != nil {
		offered = extensions.Snapshot()
	}
	result, problem := AhdGUIDialog("openFiles", title, "", "", offered)
	AhdGUICheck(problem)
	return AhdNewList(result.Paths...)
}

// AhdGUIMessageChecked is GUI.message and AhdGUIConfirmChecked GUI.confirm.
func AhdGUIMessageChecked(title, text string) {
	_, problem := AhdGUIDialog("message", title, text, "", nil)
	AhdGUICheck(problem)
}

func AhdGUIConfirmChecked(title, text string) bool {
	result, problem := AhdGUIDialog("confirm", title, text, "", nil)
	AhdGUICheck(problem)
	return result.Confirmed
}
