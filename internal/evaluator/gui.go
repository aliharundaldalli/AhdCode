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
	case "clipboard":
		return ahdruntime.AhdGUIClipboard()
	case "setClipboard":
		session.guiCheck(ahdruntime.AhdGUISetClipboard(text(0, "")))
		return Nothing
	case "checkpoint":
		return session.guiCheckpoint()
	case "resource":
		path, problem := ahdruntime.AhdGUIResource(text(0, ""))
		session.guiCheck(problem)
		return path
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
	"ListBox.", "Menu.", "MenuItem.", "Select.", "TextArea.", "PasswordInput.", "TableView.", "Image.",
	"Tabs.", "RadioGroup.", "RadioButton.", "Progress.", "DateField.", "Canvas.", "Task."}

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
	real := func(index int, fallback float64) float64 { return graphicsReal(arguments, index, fallback) }
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
	case "Window.scroll":
		created, problem := ahdruntime.AhdGUIRoot(handle, "scroll", integer(0, 8), integer(1, 12))
		return child("Container", created, problem)
	case "Window.grid":
		created, problem := ahdruntime.AhdGUIGrid(handle, true, integer(0, 1), integer(1, 8), integer(2, 8))
		return child("Container", created, problem)
	case "Window.onKey":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnKey(handle, func(key string) { session.invokeHandler(handler, key) }))
	case "Window.wait":
		// Output written before wait must be visible while the window is open.
		session.flushOutput()
		// Let a background task use the interpreter while wait is blocked.
		depth := session.yield()
		problem := ahdruntime.AhdGUIWait(handle)
		session.resume(depth)
		session.guiCheck(problem)
	case "Window.close":
		session.guiCheck(ahdruntime.AhdGUIClose(handle))
	case "Window.isOpen":
		return ahdruntime.AhdGUIIsOpen(handle)
	case "Window.every":
		var handler *FunctionValue
		if len(arguments) > 1 {
			handler, _ = arguments[1].(*FunctionValue)
		}
		if handler == nil {
			session.raise("NullError", name+" needs a Function")
		}
		id, problem := ahdruntime.AhdGUIEvery(handle, integer(0, 0), func() { session.invokeHandler(handler) })
		session.guiCheck(problem)
		return id
	case "Window.cancel":
		session.guiCheck(ahdruntime.AhdGUICancel(handle, integer(0, 0)))
	case "Window.task":
		handler := session.graphicsHandler(arguments, name)
		id, problem := ahdruntime.AhdGUITask(handle, func() (int64, string) {
			var problem string
			var result int64
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						switch value := recovered.(type) {
						case raised:
							problem = value.failure.Message
						case *raised:
							if value.failure != nil {
								problem = value.failure.Message
							}
						default:
							panic(recovered)
						}
					}
				}()
				if number, ok := session.invoke(handler, nil).(int64); ok {
					result = number
				}
			}()
			return result, problem
		})
		return child("Task", id, problem)
	case "Task.onDone":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUITaskOnDone(handle, func(value *int64, problem *string) {
			var first, second any
			if value != nil {
				first = *value
			}
			if problem != nil {
				second = *problem
			}
			session.invokeHandler(handler, first, second)
		}))
	case "Task.cancel":
		session.guiCheck(ahdruntime.AhdGUITaskCancel(handle))
	case "Task.cancelled":
		return ahdruntime.AhdGUITaskCancelled(handle)
	case "Container.canvas":
		created, problem := ahdruntime.AhdGUIAddCanvas(handle, integer(0, 240), integer(1, 160))
		return child("Canvas", created, problem)
	case "Canvas.clear":
		session.guiCheck(ahdruntime.AhdGUICanvasDraw(handle, "clear", 0, 0, 0, 0, 0, 0, text(0, ""), "", ""))
	case "Canvas.line":
		session.guiCheck(ahdruntime.AhdGUICanvasDraw(handle, "line", real(0, 0), real(1, 0), real(2, 0), real(3, 0), 0, real(5, 1), text(4, "black"), "", ""))
	case "Canvas.rectangle":
		session.guiCheck(ahdruntime.AhdGUICanvasDraw(handle, "rectangle", real(0, 0), real(1, 0), real(2, 0), real(3, 0), 0, 1, text(4, "black"), text(5, ""), ""))
	case "Canvas.circle":
		session.guiCheck(ahdruntime.AhdGUICanvasDraw(handle, "circle", real(0, 0), real(1, 0), 0, 0, real(2, 0), 1, text(3, "black"), text(4, ""), ""))
	case "Canvas.text":
		session.guiCheck(ahdruntime.AhdGUICanvasDraw(handle, "text", real(0, 0), real(1, 0), 0, 0, 0, real(4, 16), text(3, "black"), "", text(2, "")))
	case "Canvas.onMove":
		pointer := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUICanvasOnMove(handle, func(x, y float64) { session.invokeHandler(pointer, x, y) }))
	case "Canvas.onClick":
		pointer := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUICanvasOnClick(handle, func(x, y float64) { session.invokeHandler(pointer, x, y) }))
	case "Canvas.onDrag":
		pointer := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUICanvasOnDrag(handle, func(x, y float64) { session.invokeHandler(pointer, x, y) }))
	case "Window.setTitle":
		session.guiCheck(ahdruntime.AhdGUISetTitle(handle, text(0, "")))
	case "Container.column", "Container.row":
		created, problem := ahdruntime.AhdGUIContainerLayout(handle, strings.TrimPrefix(name, "Container."), integer(0, 8), integer(1, 0))
		return child("Container", created, problem)
	case "Container.scroll":
		created, problem := ahdruntime.AhdGUIContainerLayout(handle, "scroll", integer(0, 8), integer(1, 0))
		return child("Container", created, problem)
	case "Container.grid":
		created, problem := ahdruntime.AhdGUIGrid(handle, false, integer(0, 1), integer(1, 8), integer(2, 8))
		return child("Container", created, problem)
	case "Container.label":
		created, problem := ahdruntime.AhdGUIAddLeaf(handle, "label", text(0, ""), "", false)
		return child("Label", created, problem)
	case "Container.image":
		created, problem := ahdruntime.AhdGUIAddImage(handle, text(0, ""), integer(1, 0), integer(2, 0))
		return child("Image", created, problem)
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
	number := func(index int, fallback int64) int64 {
		if index < len(arguments) {
			if value, ok := arguments[index].(int64); ok {
				return value
			}
		}
		return fallback
	}
	member := name[strings.IndexByte(name, '.')+1:]
	switch name {
	case "Window.menu":
		created, problem := ahdruntime.AhdGUIMenu(handle, text(0, ""), true)
		return child("Menu", created, problem)
	case "Menu.menu":
		created, problem := ahdruntime.AhdGUIMenu(handle, text(0, ""), false)
		return child("Menu", created, problem)
	case "Menu.item":
		created, problem := ahdruntime.AhdGUIMenuItem(handle, "item", text(0, ""), text(1, ""), false)
		return child("MenuItem", created, problem)
	case "Menu.checkItem":
		created, problem := ahdruntime.AhdGUIMenuItem(handle, "check", text(0, ""), text(1, ""), flag(2, false))
		return child("MenuItem", created, problem)
	case "Menu.separator":
		session.guiCheck(ahdruntime.AhdGUIMenuSeparator(handle))
		return Nothing
	case "MenuItem.onClick":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnMenu(handle, func() { session.invokeHandler(handler) }))
		return Nothing
	case "MenuItem.setEnabled":
		session.guiCheck(ahdruntime.AhdGUIMenuSetEnabled(handle, flag(0, true)))
		return Nothing
	case "MenuItem.isEnabled":
		enabled, problem := ahdruntime.AhdGUIMenuIsEnabled(handle)
		session.guiCheck(problem)
		return enabled
	case "MenuItem.setChecked":
		session.guiCheck(ahdruntime.AhdGUIMenuSetChecked(handle, flag(0, false)))
		return Nothing
	case "MenuItem.checked":
		checked, problem := ahdruntime.AhdGUIMenuChecked(handle)
		session.guiCheck(problem)
		return checked
	case "MenuItem.setText":
		session.guiCheck(ahdruntime.AhdGUIMenuSetText(handle, text(0, "")))
		return Nothing
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
	case "TableView.sort":
		session.guiCheck(ahdruntime.AhdGUISort(handle, graphicsInt(arguments, 0, 0), flag(1, false)))
		return Nothing
	case "TableView.sortedColumn":
		column, present, problem := ahdruntime.AhdGUISortedColumn(handle)
		session.guiCheck(problem)
		if !present {
			return nil
		}
		return column
	case "TableView.sortDescending":
		descending, problem := ahdruntime.AhdGUISortDescending(handle)
		session.guiCheck(problem)
		return descending
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
	case "Container.tabs":
		created, problem := ahdruntime.AhdGUIAddTabs(handle)
		return child("Tabs", created, problem)
	case "Container.radioGroup":
		created, problem := ahdruntime.AhdGUIAddRadioGroup(handle)
		return child("RadioGroup", created, problem)
	case "Container.progress":
		created, problem := ahdruntime.AhdGUIAddProgress(handle, number(0, 100))
		return child("Progress", created, problem)
	case "Container.date":
		created, problem := ahdruntime.AhdGUIAddDate(handle, number(0, 2000), number(1, 1), number(2, 1))
		return child("DateField", created, problem)
	case "Tabs.page":
		created, problem := ahdruntime.AhdGUIAddPage(handle, text(0, ""))
		return child("Container", created, problem)
	case "Tabs.select":
		session.guiCheck(ahdruntime.AhdGUISelectTab(handle, number(0, 0)))
		return Nothing
	case "Tabs.selectedIndex":
		index, problem := ahdruntime.AhdGUITabIndex(handle)
		session.guiCheck(problem)
		return index
	case "Tabs.onChange":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnTabChange(handle, func(index int64) { session.invokeHandler(handler, index) }))
		return Nothing
	case "RadioGroup.option":
		created, problem := ahdruntime.AhdGUIAddOption(handle, text(0, ""))
		return child("RadioButton", created, problem)
	case "RadioButton.setText":
		session.guiCheck(ahdruntime.AhdGUISetRadioText(handle, text(0, "")))
		return Nothing
	case "Progress.setValue":
		session.guiCheck(ahdruntime.AhdGUISetProgressValue(handle, number(0, 0)))
		return Nothing
	case "Progress.value":
		value, problem := ahdruntime.AhdGUIProgressValue(handle)
		session.guiCheck(problem)
		return value
	case "Progress.maximum":
		value, problem := ahdruntime.AhdGUIProgressMaximum(handle)
		session.guiCheck(problem)
		return value
	case "Progress.setIndeterminate":
		session.guiCheck(ahdruntime.AhdGUISetProgressSpin(handle, flag(0, false)))
		return Nothing
	case "Progress.isIndeterminate":
		spin, problem := ahdruntime.AhdGUIProgressSpin(handle)
		session.guiCheck(problem)
		return spin
	case "DateField.setDate":
		session.guiCheck(ahdruntime.AhdGUISetDate(handle, number(0, 0), number(1, 0), number(2, 0)))
		return Nothing
	case "DateField.year":
		year, _, _, problem := ahdruntime.AhdGUIDate(handle)
		session.guiCheck(problem)
		return year
	case "DateField.month":
		_, month, _, problem := ahdruntime.AhdGUIDate(handle)
		session.guiCheck(problem)
		return month
	case "DateField.day":
		_, _, day, problem := ahdruntime.AhdGUIDate(handle)
		session.guiCheck(problem)
		return day
	case "DateField.onChange":
		handler := session.graphicsHandler(arguments, name)
		session.guiCheck(ahdruntime.AhdGUIOnDateChange(handle, func(year, month, day int64) {
			session.invokeHandler(handler, year, month, day)
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
	case member == "setVisible":
		flag, _ := arguments[0].(bool)
		session.guiCheck(ahdruntime.AhdGUISetVisible(handle, flag))
	case member == "isVisible":
		visible, problem := ahdruntime.AhdGUIIsVisible(handle)
		session.guiCheck(problem)
		return visible
	case member == "setSize" && strings.HasPrefix(name, "Image."):
		session.guiCheck(ahdruntime.AhdGUISetImageSize(handle, number(0, 0), number(1, 0)))
	case member == "setSize":
		session.guiCheck(ahdruntime.AhdGUISetLabelSize(handle, text(0, "")))
	case member == "setWrap":
		session.guiCheck(ahdruntime.AhdGUISetWrap(handle, flag(0, false)))
	case member == "setBold":
		flag, _ := arguments[0].(bool)
		session.guiCheck(ahdruntime.AhdGUISetLabelBold(handle, flag))
	case member == "setMinWidth":
		session.guiCheck(ahdruntime.AhdGUISetMinWidth(handle, number(0, 0)))
	case member == "setPreferredWidth":
		session.guiCheck(ahdruntime.AhdGUISetPreferredWidth(handle, number(0, 0)))
	case member == "setAlign":
		session.guiCheck(ahdruntime.AhdGUISetAlign(handle, text(0, ""), text(1, "")))
	case member == "setSpan":
		session.guiCheck(ahdruntime.AhdGUISetSpan(handle, number(0, 1), number(1, 1)))
	default:
		session.raise("Error", "unsupported GUI operation "+name)
	}
	return Nothing
}

// CloseGUI closes every open Window; the REPL calls it when it ends.
func CloseGUI() { ahdruntime.AhdGUICloseAll() }
