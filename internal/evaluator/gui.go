package evaluator

import (
	"fmt"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/ir"
)

// The GUI standard module's REPL implementation. Every member calls the same
// runtime functions a compiled program calls, so validation, layout rules,
// callbacks, and the ahdgui helper protocol are shared exactly.

func guiClassID(name string) ir.ClassID { return ir.ClassID("builtin:GUI::class::" + name) }

func guiHandleField(name string) ir.FieldID {
	return ir.FieldID("builtin:GUI::class::" + name + "::field::handle")
}

func (session *Session) guiCheck(problem string) {
	if problem != "" {
		session.raise("GUIError", problem)
	}
}

func guiInstance(className string, handle int64) *Instance {
	return &Instance{Class: guiClassID(className), Fields: map[ir.FieldID]any{guiHandleField(className): handle}}
}

func (session *Session) guiBuiltin(name string, arguments []any) any {
	text := func(index int, fallback string) string { return graphicsText(arguments, index, fallback) }
	switch name {
	case "window":
		handle, problem := ahdruntime.AhdGUIWindowOpen(text(0, "AhdCode"), graphicsInt(arguments, 1, 800), graphicsInt(arguments, 2, 600))
		session.guiCheck(problem)
		return guiInstance("Window", handle)
	case "openFile", "selectFolder", "saveFile":
		title := map[string]string{"openFile": "Open File", "selectFolder": "Select Folder", "saveFile": "Save File"}[name]
		suggested, extensions := "", 1
		if name == "saveFile" {
			suggested, extensions = text(1, ""), 2
		}
		var offered []string
		if name != "selectFolder" {
			offered = guiStrings(arguments, extensions)
		}
		path, present, problem := ahdruntime.AhdGUIDialogPath(name, text(0, title), suggested, offered)
		session.guiCheck(problem)
		if !present {
			return nil
		}
		return path
	case "openFiles":
		result, problem := ahdruntime.AhdGUIDialog("openFiles", text(0, "Open Files"), "", "", guiStrings(arguments, 1))
		session.guiCheck(problem)
		return stringList(result.Paths)
	case "message":
		_, problem := ahdruntime.AhdGUIDialog("message", text(0, ""), text(1, ""), "", nil)
		session.guiCheck(problem)
		return Nothing
	case "confirm":
		result, problem := ahdruntime.AhdGUIDialog("confirm", text(0, ""), text(1, ""), "", nil)
		session.guiCheck(problem)
		return result.Confirmed
	}
	session.raise("Error", "unsupported GUI function "+name)
	return nil
}

// guiStrings reads a List<String> argument; an omitted one is empty.
func guiStrings(arguments []any, index int) []string {
	if index >= len(arguments) {
		return nil
	}
	list, ok := arguments[index].(*List)
	if !ok || list == nil {
		return nil
	}
	values := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		text, _ := item.(string)
		values = append(values, text)
	}
	return values
}

// guiRows reads a List<List<String>> argument; an omitted one is empty.
func guiRows(arguments []any, index int) [][]string {
	if index >= len(arguments) {
		return nil
	}
	list, ok := arguments[index].(*List)
	if !ok || list == nil {
		return nil
	}
	rows := make([][]string, 0, len(list.Items))
	for _, item := range list.Items {
		rows = append(rows, guiStrings([]any{item}, 0))
	}
	return rows
}

func rowsList(rows [][]string) *List {
	items := make([]any, len(rows))
	for index, row := range rows {
		items[index] = stringList(row)
	}
	return &List{Items: items}
}

// guiClassPrefixes are the GUI Classes whose members are type operations.
var guiClassPrefixes = []string{"Window.", "Container.", "Label.", "Button.", "TextInput.", "Checkbox.",
	"ListBox.", "Select.", "TextArea.", "PasswordInput.", "TableView."}

// isGUIOperation reports whether a type operation belongs to a GUI Class.
func isGUIOperation(name string) bool {
	for _, class := range guiClassPrefixes {
		if strings.HasPrefix(name, class) {
			return true
		}
	}
	return false
}

func (session *Session) guiOperation(name string, receiver any, arguments []any) any {
	className := name[:strings.IndexByte(name, '.')]
	handle := session.graphicsHandle(receiver, guiHandleField(className))
	text := func(index int, fallback string) string { return graphicsText(arguments, index, fallback) }
	integer := func(index int, fallback int64) int64 { return graphicsInt(arguments, index, fallback) }
	flag := func(index int, fallback bool) bool {
		if index < len(arguments) {
			if value, ok := arguments[index].(bool); ok {
				return value
			}
		}
		return fallback
	}
	child := func(className string, created int64, problem string) any {
		session.guiCheck(problem)
		return guiInstance(className, created)
	}
	switch name {
	case "Window.column", "Window.row":
		created, problem := ahdruntime.AhdGUIRoot(handle, strings.TrimPrefix(name, "Window."), integer(0, 8), integer(1, 12))
		return child("Container", created, problem)
	case "Window.onKey":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnKey(handle, func(key string) { session.invokeHandler(handler, key) }))
	case "Window.wait":
		// Output written before wait must be visible while the window is open.
		session.flushOutput()
		session.guiCheck(ahdruntime.AhdGUIWait(handle))
	case "Window.close":
		session.guiCheck(ahdruntime.AhdGUIClose(handle))
	case "Window.isOpen":
		return ahdruntime.AhdGUIIsOpen(handle)
	case "Window.setTitle":
		session.guiCheck(ahdruntime.AhdGUISetTitle(handle, text(0, "")))
	case "Container.column", "Container.row":
		created, problem := ahdruntime.AhdGUIContainerLayout(handle, strings.TrimPrefix(name, "Container."), integer(0, 8), integer(1, 0))
		return child("Container", created, problem)
	case "Container.label":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "label", text(0, ""), "", false)
		return child("Label", created, problem)
	case "Container.button":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "button", text(0, ""), "", false)
		return child("Button", created, problem)
	case "Container.textInput":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "textInput", "", text(0, ""), false)
		return child("TextInput", created, problem)
	case "Container.checkbox":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "checkbox", text(0, ""), "", flag(1, false))
		return child("Checkbox", created, problem)
	case "Label.text", "Button.text", "TextInput.text":
		value, problem := ahdruntime.AhdGUIText(handle)
		session.guiCheck(problem)
		return value
	case "Label.setText", "Button.setText", "TextInput.setText":
		session.guiCheck(ahdruntime.AhdGUISetText(handle, text(0, "")))
	case "Button.onClick":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnClick(handle, func() { session.invokeHandler(handler) }))
	case "Checkbox.checked":
		value, problem := ahdruntime.AhdGUIChecked(handle)
		session.guiCheck(problem)
		return value
	case "Checkbox.setChecked":
		session.guiCheck(ahdruntime.AhdGUISetChecked(handle, flag(0, false)))
	case "Window.setBackground":
		session.guiCheck(ahdruntime.AhdGUIWindowSetBackground(handle, text(0, "")))
	case "Container.setBackground", "Label.setBackground", "Button.setBackground", "TextInput.setBackground", "Checkbox.setBackground":
		session.guiCheck(ahdruntime.AhdGUISetColor(handle, "background", text(0, "")))
	case "Label.setForeground", "Button.setForeground", "TextInput.setForeground", "Checkbox.setForeground":
		session.guiCheck(ahdruntime.AhdGUISetColor(handle, "foreground", text(0, "")))
	case "Button.setEnabled", "TextInput.setEnabled", "Checkbox.setEnabled":
		session.guiCheck(ahdruntime.AhdGUISetEnabled(handle, flag(0, true)))
	case "Button.isEnabled", "TextInput.isEnabled", "Checkbox.isEnabled":
		enabled, problem := ahdruntime.AhdGUIIsEnabled(handle)
		session.guiCheck(problem)
		return enabled
	default:
		return session.guiWidgetOperation(name, handle, arguments)
	}
	return Nothing
}

// guiWidgetOperation runs the v2.0 members: the new widgets, change
// callbacks, and resizable Windows.
func (session *Session) guiWidgetOperation(name string, handle int64, arguments []any) any {
	text := func(index int, fallback string) string { return graphicsText(arguments, index, fallback) }
	child := func(className string, created int64, problem string) any {
		session.guiCheck(problem)
		return guiInstance(className, created)
	}
	nullableIndex := func(index int) (int64, bool) {
		if index < len(arguments) {
			if value, ok := arguments[index].(int64); ok {
				return value, true
			}
		}
		return 0, false
	}
	selection := func(index int64, present bool, problem string) any {
		session.guiCheck(problem)
		if !present {
			return nil
		}
		return index
	}
	member := name[strings.IndexByte(name, '.')+1:]
	switch name {
	case "Window.setResizable":
		flag, _ := arguments[0].(bool)
		session.guiCheck(ahdruntime.AhdGUISetResizable(handle, flag))
		return Nothing
	case "Window.isResizable":
		resizable, problem := ahdruntime.AhdGUIIsResizable(handle)
		session.guiCheck(problem)
		return resizable
	case "Container.passwordInput":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "passwordInput", "", text(0, ""), false)
		return child("PasswordInput", created, problem)
	case "Container.textArea":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "textArea", "", text(0, ""), false)
		return child("TextArea", created, problem)
	case "Container.listBox":
		created, problem := ahdruntime.AhdGUIAddList(handle, "listBox", guiStrings(arguments, 0), -1)
		return child("ListBox", created, problem)
	case "Container.select":
		selected := int64(-1)
		if index, present := nullableIndex(1); present {
			selected = index
			if index < 0 {
				session.guiCheck(fmt.Sprintf("selectedIndex %d is outside the items", index))
			}
		}
		created, problem := ahdruntime.AhdGUIAddList(handle, "select", guiStrings(arguments, 0), selected)
		return child("Select", created, problem)
	case "Container.table":
		created, problem := ahdruntime.AhdGUIAddTable(handle, guiStrings(arguments, 0), guiRows(arguments, 1))
		return child("TableView", created, problem)
	case "Checkbox.onChange":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnCheckChange(handle, func(checked bool) { session.invokeHandler(handler, checked) }))
		return Nothing
	case "TableView.columns":
		columns, problem := ahdruntime.AhdGUIColumns(handle)
		session.guiCheck(problem)
		return stringList(columns)
	case "TableView.rows":
		rows, problem := ahdruntime.AhdGUIRows(handle)
		session.guiCheck(problem)
		return rowsList(rows)
	case "TableView.setRows":
		session.guiCheck(ahdruntime.AhdGUISetRows(handle, guiRows(arguments, 0)))
		return Nothing
	case "TableView.selectedRow":
		return selection(ahdruntime.AhdGUISelectedIndex(handle))
	case "TableView.selectRow":
		index, present := nullableIndex(0)
		session.guiCheck(ahdruntime.AhdGUISelect(handle, index, present))
		return Nothing
	case "TableView.onSelect":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnRowSelect(handle, func(row *int64) {
			if row == nil {
				session.invokeHandler(handler, nil)
				return
			}
			session.invokeHandler(handler, *row)
		}))
		return Nothing
	}
	switch {
	case member == "onChange" && (strings.HasPrefix(name, "TextInput.") || strings.HasPrefix(name, "PasswordInput.") || strings.HasPrefix(name, "TextArea.")):
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnTextChange(handle, func(value string) { session.invokeHandler(handler, value) }))
	case member == "text":
		value, problem := ahdruntime.AhdGUIText(handle)
		session.guiCheck(problem)
		return value
	case member == "setText":
		session.guiCheck(ahdruntime.AhdGUISetText(handle, text(0, "")))
	case member == "items":
		items, problem := ahdruntime.AhdGUIItems(handle)
		session.guiCheck(problem)
		return stringList(items)
	case member == "setItems":
		session.guiCheck(ahdruntime.AhdGUISetItems(handle, guiStrings(arguments, 0)))
	case member == "selectedIndex":
		return selection(ahdruntime.AhdGUISelectedIndex(handle))
	case member == "selectedText":
		value, present, problem := ahdruntime.AhdGUISelectedText(handle)
		session.guiCheck(problem)
		if !present {
			return nil
		}
		return value
	case member == "select":
		index, present := nullableIndex(0)
		session.guiCheck(ahdruntime.AhdGUISelect(handle, index, present))
	case member == "onChange":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnSelectionChange(handle, func(index *int64, value *string) {
			var first, second any
			if index != nil {
				first = *index
			}
			if value != nil {
				second = *value
			}
			session.invokeHandler(handler, first, second)
		}))
	case member == "setBackground":
		session.guiCheck(ahdruntime.AhdGUISetColor(handle, "background", text(0, "")))
	case member == "setForeground":
		session.guiCheck(ahdruntime.AhdGUISetColor(handle, "foreground", text(0, "")))
	case member == "setEnabled":
		flag, _ := arguments[0].(bool)
		session.guiCheck(ahdruntime.AhdGUISetEnabled(handle, flag))
	case member == "isEnabled":
		enabled, problem := ahdruntime.AhdGUIIsEnabled(handle)
		session.guiCheck(problem)
		return enabled
	default:
		session.raise("Error", "unsupported GUI operation "+name)
	}
	return Nothing
}

// CloseGUI closes every open Window; the REPL calls it when it ends.
func CloseGUI() { ahdruntime.AhdGUICloseAll() }
