package golang

import (
	"strings"

	"ahdcode/internal/ir"
)

const guiModulePrefix = "builtin:GUI::"

func guiClass(name string) ir.ClassID { return ir.ClassID("builtin:GUI::class::" + name) }

func guiHandleField(name string) ir.FieldID {
	return ir.FieldID("builtin:GUI::class::" + name + "::field::handle")
}

// guiCanvasReceiver reports a call on GUI's Canvas. Graphics has a class of
// the same name, and its members are lowered first unless the receiver is
// the GUI class.
func (generator *generator) guiCanvasReceiver(value *ir.CallExpr) bool {
	if value.Callee == nil {
		return false
	}
	return value.Callee.ExprMeta().Type.Class == guiClass("Canvas")
}

// guiClassNames are the GUI Classes whose members are type operations.
var guiClassNames = []string{"Window.", "Container.", "Label.", "Button.", "TextInput.", "Checkbox.",
	"ListBox.", "Menu.", "MenuItem.", "Select.", "TextArea.", "PasswordInput.", "TableView.", "Image.",
	"Tabs.", "RadioGroup.", "RadioButton.", "Progress.", "DateField.", "Canvas.", "Task."}

// isGUIOperation reports whether a type operation belongs to a GUI Class.
func isGUIOperation(name string) bool {
	for _, class := range guiClassNames {
		if strings.HasPrefix(name, class) {
			return true
		}
	}
	return false
}

// guiCall lowers GUI.window and the dialogs.
func (generator *generator) guiCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), guiModulePrefix)
	arguments := graphicsArguments{generator: generator, value: value}
	list := func(index int) string {
		if !arguments.present(index) {
			return "nil"
		}
		return generator.expr(value.Arguments[index].Value)
	}
	generator.usesGUI = true
	switch name {
	case "window":
		handle := "AhdGUIWindowChecked(" + arguments.text(0, `"AhdCode"`) + ", " + arguments.integer(1, "int64(800)") + ", " + arguments.integer(2, "int64(600)") + ")"
		return generator.graphicsValue(guiClass("Window"), handle, meta)
	case "openFile":
		return `AhdGUIDialogPathChecked("openFile", ` + arguments.text(0, `"Open File"`) + `, "", ` + list(1) + ")"
	case "openFiles":
		return "AhdGUIOpenFilesChecked(" + arguments.text(0, `"Open Files"`) + ", " + list(1) + ")"
	case "selectFolder":
		return `AhdGUIDialogPathChecked("selectFolder", ` + arguments.text(0, `"Select Folder"`) + `, "", nil)`
	case "saveFile":
		return `AhdGUIDialogPathChecked("saveFile", ` + arguments.text(0, `"Save File"`) + ", " + arguments.text(1, `""`) + ", " + list(2) + ")"
	case "message":
		return "AhdGUIMessageChecked(" + arguments.text(0, `""`) + ", " + arguments.text(1, `""`) + ")"
	case "confirm":
		return "AhdGUIConfirmChecked(" + arguments.text(0, `""`) + ", " + arguments.text(1, `""`) + ")"
	case "clipboard":
		return "AhdGUIClipboard()"
	case "setClipboard":
		return "AhdGUISetClipboardChecked(" + arguments.text(0, `""`) + ")"
	case "checkpoint":
		return "AhdGUICheckpointChecked()"
	case "resource":
		return "AhdGUIResourceChecked(" + arguments.text(0, `""`) + ")"
	}
	return generator.unsupported("GUI function "+name, meta.Span)
}

// guiOperation lowers the GUI members. The receiver is read once, for its
// handle; each member is one runtime call, and a reported problem becomes a
// GUIError through AhdGUICheck.
func (generator *generator) guiOperation(name string, value *ir.CallExpr) string {
	generator.usesGUI = true
	meta := value.ExprMeta()
	if value.Callee == nil {
		generator.fail(CodeGenerationFailure, name+" has no receiver", meta.Span, "the IR call is malformed")
		return "nil"
	}
	className := name[:strings.IndexByte(name, '.')]
	handle := generator.expr(value.Callee) + "." + generator.fieldName(guiHandleField(className)) + "_get()"
	arguments := graphicsArguments{generator: generator, value: value}
	check := func(call string) string { return "AhdGUICheck(" + call + ")" }
	flag := func(index int, fallback string) string {
		if !arguments.present(index) {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	child := func(className, call string) string { return generator.graphicsValue(guiClass(className), call, meta) }
	handlerAt := func(index int, params ...ir.Type) string {
		if index >= len(value.Arguments) || value.Arguments[index].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		return generator.adaptHandler(value.Arguments[index].Value, params...)
	}
	handler := func(params ...ir.Type) string { return handlerAt(0, params...) }
	// taskWork runs the function on the task goroutine and turns a raised
	// Error into the problem string. asIs passes a nullable callback through
	// unchanged, because its parameters are already pointers.
	taskWork := func() string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		call := generator.expr(value.Arguments[0].Value) + "()"
		// A Function value returns a nullable Int, so the task reads the
		// pointer. Null becomes a reported problem rather than a zero result.
		return "func() (result int64, problem string) { func() { defer func() { if recovered := recover(); recovered != nil { problem = AhdSignalOf(recovered).Message } }(); if value := " + call + "; value != nil { result = *value } else if problem == \"\" { problem = \"the background task returned null\" } }(); return }"
	}
	asIs := func() string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		return generator.expr(value.Arguments[0].Value)
	}
	switch name {
	case "Window.column", "Window.row":
		return child("Container", "AhdGUIRootChecked("+handle+", "+`"`+strings.TrimPrefix(name, "Window.")+`", `+arguments.integer(0, "int64(8)")+", "+arguments.integer(1, "int64(12)")+")")
	case "Window.scroll":
		return child("Container", "AhdGUIRootChecked("+handle+`, "scroll", `+arguments.integer(0, "int64(8)")+", "+arguments.integer(1, "int64(12)")+")")
	case "Window.grid":
		return child("Container", "AhdGUIGridChecked("+handle+", true, "+arguments.integer(0, "int64(1)")+", "+arguments.integer(1, "int64(8)")+", "+arguments.integer(2, "int64(8)")+")")
	case "Window.onKey":
		return check("AhdGUIOnKey(" + handle + ", " + handler(ir.Type{Kind: ir.StringType}) + ")")
	case "Window.wait":
		return check("AhdGUIWait(" + handle + ")")
	case "Window.close":
		return check("AhdGUIClose(" + handle + ")")
	case "Window.isOpen":
		return "AhdGUIIsOpen(" + handle + ")"
	case "Window.setTitle":
		return check("AhdGUISetTitle(" + handle + ", " + arguments.text(0, `""`) + ")")
	case "Container.column", "Container.row":
		return child("Container", "AhdGUIContainerLayoutChecked("+handle+", "+`"`+strings.TrimPrefix(name, "Container.")+`", `+arguments.integer(0, "int64(8)")+", "+arguments.integer(1, "int64(0)")+")")
	case "Container.scroll":
		return child("Container", "AhdGUIContainerLayoutChecked("+handle+`, "scroll", `+arguments.integer(0, "int64(8)")+", "+arguments.integer(1, "int64(0)")+")")
	case "Container.grid":
		return child("Container", "AhdGUIGridChecked("+handle+", false, "+arguments.integer(0, "int64(1)")+", "+arguments.integer(1, "int64(8)")+", "+arguments.integer(2, "int64(8)")+")")
	case "Container.label":
		return child("Label", "AhdGUIAddLeafChecked("+handle+`, "label", `+arguments.text(0, `""`)+`, "", false)`)
	case "Container.image":
		return child("Image", "AhdGUIAddImageChecked("+handle+", "+arguments.text(0, `""`)+", "+arguments.integer(1, "int64(0)")+", "+arguments.integer(2, "int64(0)")+")")
	case "Container.button":
		return child("Button", "AhdGUIAddLeafChecked("+handle+`, "button", `+arguments.text(0, `""`)+`, "", false)`)
	case "Container.textInput":
		return child("TextInput", "AhdGUIAddLeafChecked("+handle+`, "textInput", "", `+arguments.text(0, `""`)+`, false)`)
	case "Container.checkbox":
		return child("Checkbox", "AhdGUIAddLeafChecked("+handle+`, "checkbox", `+arguments.text(0, `""`)+`, "", `+flag(1, "false")+")")
	case "Label.text", "Button.text", "TextInput.text":
		return "AhdGUITextChecked(" + handle + ")"
	case "Label.setText", "Button.setText", "TextInput.setText":
		return check("AhdGUISetText(" + handle + ", " + arguments.text(0, `""`) + ")")
	case "Button.onClick":
		return check("AhdGUIOnClick(" + handle + ", " + handler() + ")")
	case "Checkbox.checked":
		return "AhdGUICheckedChecked(" + handle + ")"
	case "Checkbox.setChecked":
		return check("AhdGUISetChecked(" + handle + ", " + flag(0, "false") + ")")
	case "Window.setBackground":
		return check("AhdGUIWindowSetBackground(" + handle + ", " + arguments.text(0, `""`) + ")")
	case "Container.setBackground", "Label.setBackground", "Button.setBackground", "TextInput.setBackground", "Checkbox.setBackground":
		return check("AhdGUISetColor(" + handle + `, "background", ` + arguments.text(0, `""`) + ")")
	case "Label.setForeground", "Button.setForeground", "TextInput.setForeground", "Checkbox.setForeground":
		return check("AhdGUISetColor(" + handle + `, "foreground", ` + arguments.text(0, `""`) + ")")
	case "Button.setEnabled", "TextInput.setEnabled", "Checkbox.setEnabled":
		return check("AhdGUISetEnabled(" + handle + ", " + flag(0, "true") + ")")
	case "Button.isEnabled", "TextInput.isEnabled", "Checkbox.isEnabled":
		return "AhdGUIIsEnabledChecked(" + handle + ")"
	case "Window.every":
		return "AhdGUIEveryChecked(" + handle + ", " + arguments.integer(0, "int64(0)") + ", " + handlerAt(1) + ")"
	case "Window.cancel":
		return check("AhdGUICancel(" + handle + ", " + arguments.integer(0, "int64(0)") + ")")
	case "Window.task":
		return child("Task", "AhdGUITaskChecked("+handle+", "+taskWork()+")")
	case "Task.onDone":
		return check("AhdGUITaskOnDone(" + handle + ", " + asIs() + ")")
	case "Task.cancel":
		return check("AhdGUITaskCancel(" + handle + ")")
	case "Task.cancelled":
		return "AhdGUITaskCancelledChecked(" + handle + ")"
	case "Container.canvas":
		return child("Canvas", "AhdGUIAddCanvasChecked("+handle+", "+arguments.integer(0, "int64(240)")+", "+arguments.integer(1, "int64(160)")+")")
	case "Canvas.clear":
		return check("AhdGUICanvasDraw(" + handle + `, "clear", 0, 0, 0, 0, 0, 0, ` + arguments.text(0, `""`) + `, "", "")`)
	case "Canvas.line":
		return check("AhdGUICanvasDraw(" + handle + `, "line", ` + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", " + arguments.real(2, "0") + ", " + arguments.real(3, "0") + ", 0, " + arguments.real(5, "1") + ", " + arguments.text(4, `"black"`) + `, "", "")`)
	case "Canvas.rectangle":
		return check("AhdGUICanvasDraw(" + handle + `, "rectangle", ` + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", " + arguments.real(2, "0") + ", " + arguments.real(3, "0") + ", 0, 1, " + arguments.text(4, `"black"`) + ", " + arguments.text(5, `""`) + `, "")`)
	case "Canvas.circle":
		return check("AhdGUICanvasDraw(" + handle + `, "circle", ` + arguments.real(0, "0") + ", " + arguments.real(1, "0") + ", 0, 0, " + arguments.real(2, "0") + ", 1, " + arguments.text(3, `"black"`) + ", " + arguments.text(4, `""`) + `, "")`)
	case "Canvas.text":
		return check("AhdGUICanvasDraw(" + handle + `, "text", ` + arguments.real(0, "0") + ", " + arguments.real(1, "0") + `, 0, 0, 0, ` + arguments.real(4, "16") + ", " + arguments.text(3, `"black"`) + `, "", ` + arguments.text(2, `""`) + ")")
	case "Canvas.onMove":
		return check("AhdGUICanvasOnMove(" + handle + ", " + handler(ir.Type{Kind: ir.RealType}, ir.Type{Kind: ir.RealType}) + ")")
	case "Canvas.onClick":
		return check("AhdGUICanvasOnClick(" + handle + ", " + handler(ir.Type{Kind: ir.RealType}, ir.Type{Kind: ir.RealType}) + ")")
	case "Canvas.onDrag":
		return check("AhdGUICanvasOnDrag(" + handle + ", " + handler(ir.Type{Kind: ir.RealType}, ir.Type{Kind: ir.RealType}) + ")")
	}
	if call, ok := generator.guiWidgetOperation(name, value, handle, arguments); ok {
		return call
	}
	return generator.unsupported("GUI operation "+name, meta.Span)
}

// guiWidgetOperation lowers the v2.0 members: the new widgets, change
// callbacks, and resizable Windows.
func (generator *generator) guiWidgetOperation(name string, value *ir.CallExpr, handle string, arguments graphicsArguments) (string, bool) {
	meta := value.ExprMeta()
	check := func(call string) string { return "AhdGUICheck(" + call + ")" }
	child := func(className, call string) string { return generator.graphicsValue(guiClass(className), call, meta) }
	list := func(index int, fallback string) string {
		if !arguments.present(index) {
			return fallback
		}
		return generator.expr(value.Arguments[index].Value)
	}
	nullableInt := func(index int) string {
		if !arguments.present(index) {
			return "nil"
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.IntType}, true)
	}
	// A Function value's scalar parameters already use the nullable
	// representation, so a callback with Int? and String? parameters is
	// passed as it is; one with plain parameters is adapted.
	callback := func() string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		return generator.expr(value.Arguments[0].Value)
	}
	handler := func(params ...ir.Type) string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		return generator.adaptHandler(value.Arguments[0].Value, params...)
	}
	flag := func(index int, fallback string) string {
		if !arguments.present(index) {
			return fallback
		}
		return generator.value(value.Arguments[index].Value, ir.Type{Kind: ir.BoolType}, false)
	}
	className := name[:strings.IndexByte(name, '.')]
	member := name[strings.IndexByte(name, '.')+1:]
	switch name {
	case "Window.menu":
		return child("Menu", "AhdGUIAddMenuChecked("+handle+", "+arguments.text(0, `""`)+", true)"), true
	case "Menu.menu":
		return child("Menu", "AhdGUIAddMenuChecked("+handle+", "+arguments.text(0, `""`)+", false)"), true
	case "Menu.item":
		return child("MenuItem", "AhdGUIMenuItemChecked("+handle+`, "item", `+arguments.text(0, `""`)+", "+arguments.text(1, `""`)+", false)"), true
	case "Menu.checkItem":
		return child("MenuItem", "AhdGUIMenuItemChecked("+handle+`, "check", `+arguments.text(0, `""`)+", "+arguments.text(1, `""`)+", "+flag(2, "false")+")"), true
	case "Menu.separator":
		return check("AhdGUIMenuSeparator(" + handle + ")"), true
	case "MenuItem.onClick":
		return check("AhdGUIOnMenu(" + handle + ", " + handler() + ")"), true
	case "MenuItem.setEnabled":
		return check("AhdGUIMenuSetEnabled(" + handle + ", " + flag(0, "true") + ")"), true
	case "MenuItem.isEnabled":
		return "AhdGUIMenuIsEnabledChecked(" + handle + ")", true
	case "MenuItem.setChecked":
		return check("AhdGUIMenuSetChecked(" + handle + ", " + flag(0, "false") + ")"), true
	case "MenuItem.checked":
		return "AhdGUIMenuCheckedChecked(" + handle + ")", true
	case "MenuItem.setText":
		return check("AhdGUIMenuSetText(" + handle + ", " + arguments.text(0, `""`) + ")"), true
	case "Window.setResizable":
		return check("AhdGUISetResizable(" + handle + ", " + flag(0, "true") + ")"), true
	case "Window.isResizable":
		return "AhdGUIIsResizableChecked(" + handle + ")", true
	case "Container.passwordInput":
		return child("PasswordInput", "AhdGUIAddLeafChecked("+handle+`, "passwordInput", "", `+arguments.text(0, `""`)+`, false)`), true
	case "Container.textArea":
		return child("TextArea", "AhdGUIAddLeafChecked("+handle+`, "textArea", "", `+arguments.text(0, `""`)+`, false)`), true
	case "Container.listBox":
		return child("ListBox", "AhdGUIAddListChecked("+handle+`, "listBox", `+list(0, "AhdNewList[string]()")+", nil)"), true
	case "Container.select":
		return child("Select", "AhdGUIAddListChecked("+handle+`, "select", `+list(0, "AhdNewList[string]()")+", "+nullableInt(1)+")"), true
	case "Container.table":
		return child("TableView", "AhdGUIAddTableChecked("+handle+", "+list(0, "AhdNewList[string]()")+", "+list(1, "AhdNewList[*AhdList[string]]()")+")"), true
	case "Checkbox.onChange":
		return check("AhdGUIOnCheckChange(" + handle + ", " + handler(ir.Type{Kind: ir.BoolType}) + ")"), true
	case "TableView.columns":
		return "AhdGUIColumnsChecked(" + handle + ")", true
	case "TableView.rows":
		return "AhdGUIRowsChecked(" + handle + ")", true
	case "TableView.setRows":
		return "AhdGUISetRowsChecked(" + handle + ", " + list(0, "AhdNewList[*AhdList[string]]()") + ")", true
	case "TableView.selectedRow":
		return "AhdGUISelectedIndexChecked(" + handle + ")", true
	case "TableView.selectRow":
		return "AhdGUISelectChecked(" + handle + ", " + nullableInt(0) + ")", true
	case "TableView.onSelect":
		return check("AhdGUIOnRowSelect(" + handle + ", " + callback() + ")"), true
	case "TableView.sort":
		return "AhdGUISortChecked(" + handle + ", " + arguments.integer(0, "int64(0)") + ", " + flag(1, "false") + ")", true
	case "TableView.sortedColumn":
		return "AhdGUISortedColumnChecked(" + handle + ")", true
	case "TableView.sortDescending":
		return "AhdGUISortDescendingChecked(" + handle + ")", true
	case "Container.tabs":
		return child("Tabs", "AhdGUIAddTabsChecked("+handle+")"), true
	case "Container.radioGroup":
		return child("RadioGroup", "AhdGUIAddRadioGroupChecked("+handle+")"), true
	case "Container.progress":
		return child("Progress", "AhdGUIAddProgressChecked("+handle+", "+arguments.integer(0, "int64(100)")+")"), true
	case "Container.date":
		return child("DateField", "AhdGUIAddDateChecked("+handle+", "+arguments.integer(0, "int64(2000)")+", "+arguments.integer(1, "int64(1)")+", "+arguments.integer(2, "int64(1)")+")"), true
	case "Tabs.page":
		return child("Container", "AhdGUIAddPageChecked("+handle+", "+arguments.text(0, `""`)+")"), true
	case "Tabs.select":
		return check("AhdGUISelectTab(" + handle + ", " + arguments.integer(0, "int64(0)") + ")"), true
	case "Tabs.selectedIndex":
		return "AhdGUITabIndexChecked(" + handle + ")", true
	case "Tabs.onChange":
		return check("AhdGUIOnTabChange(" + handle + ", " + handler(ir.Type{Kind: ir.IntType}) + ")"), true
	case "RadioGroup.option":
		return child("RadioButton", "AhdGUIAddOptionChecked("+handle+", "+arguments.text(0, `""`)+")"), true
	case "RadioButton.setText":
		return check("AhdGUISetRadioText(" + handle + ", " + arguments.text(0, `""`) + ")"), true
	case "Progress.setValue":
		return check("AhdGUISetProgressValue(" + handle + ", " + arguments.integer(0, "int64(0)") + ")"), true
	case "Progress.value":
		return "AhdGUIProgressValueChecked(" + handle + ")", true
	case "Progress.maximum":
		return "AhdGUIProgressMaximumChecked(" + handle + ")", true
	case "Progress.setIndeterminate":
		return check("AhdGUISetProgressSpin(" + handle + ", " + flag(0, "false") + ")"), true
	case "Progress.isIndeterminate":
		return "AhdGUIProgressSpinChecked(" + handle + ")", true
	case "DateField.setDate":
		return check("AhdGUISetDate(" + handle + ", " + arguments.integer(0, "int64(0)") + ", " + arguments.integer(1, "int64(0)") + ", " + arguments.integer(2, "int64(0)") + ")"), true
	case "DateField.year":
		return "AhdGUIDateYearChecked(" + handle + ")", true
	case "DateField.month":
		return "AhdGUIDateMonthChecked(" + handle + ")", true
	case "DateField.day":
		return "AhdGUIDateDayChecked(" + handle + ")", true
	case "DateField.onChange":
		return check("AhdGUIOnDateChange(" + handle + ", " + handler(ir.Type{Kind: ir.IntType}, ir.Type{Kind: ir.IntType}, ir.Type{Kind: ir.IntType}) + ")"), true
	}
	switch className + "." + member {
	case "TextInput.onChange", "PasswordInput.onChange", "TextArea.onChange":
		return check("AhdGUIOnTextChange(" + handle + ", " + handler(ir.Type{Kind: ir.StringType}) + ")"), true
	case "PasswordInput.text", "TextArea.text":
		return "AhdGUITextChecked(" + handle + ")", true
	case "PasswordInput.setText", "TextArea.setText":
		return check("AhdGUISetText(" + handle + ", " + arguments.text(0, `""`) + ")"), true
	case "ListBox.items", "Select.items":
		return "AhdGUIItemsChecked(" + handle + ")", true
	case "ListBox.setItems", "Select.setItems":
		return "AhdGUISetItemsChecked(" + handle + ", " + list(0, "AhdNewList[string]()") + ")", true
	case "ListBox.selectedIndex", "Select.selectedIndex", "RadioGroup.selectedIndex":
		return "AhdGUISelectedIndexChecked(" + handle + ")", true
	case "ListBox.selectedText", "Select.selectedText", "RadioGroup.selectedText":
		return "AhdGUISelectedTextChecked(" + handle + ")", true
	case "ListBox.select", "Select.select", "RadioGroup.select":
		return "AhdGUISelectChecked(" + handle + ", " + nullableInt(0) + ")", true
	case "ListBox.onChange", "Select.onChange", "RadioGroup.onChange":
		return check("AhdGUIOnSelectionChange(" + handle + ", " + callback() + ")"), true
	}
	switch member {
	case "setBackground":
		return check("AhdGUISetColor(" + handle + `, "background", ` + arguments.text(0, `""`) + ")"), true
	case "setForeground":
		return check("AhdGUISetColor(" + handle + `, "foreground", ` + arguments.text(0, `""`) + ")"), true
	case "setEnabled":
		return check("AhdGUISetEnabled(" + handle + ", " + flag(0, "true") + ")"), true
	case "isEnabled":
		return "AhdGUIIsEnabledChecked(" + handle + ")", true
	case "setVisible":
		visible := "true"
		if arguments.present(0) {
			visible = generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.BoolType}, false)
		}
		return check("AhdGUISetVisible(" + handle + ", " + visible + ")"), true
	case "isVisible":
		return "AhdGUIIsVisibleChecked(" + handle + ")", true
	case "setSize":
		if className == "Image" {
			return check("AhdGUISetImageSize(" + handle + ", " + arguments.integer(0, "int64(0)") + ", " + arguments.integer(1, "int64(0)") + ")"), true
		}
		return check("AhdGUISetLabelSize(" + handle + ", " + arguments.text(0, `""`) + ")"), true
	case "setWrap":
		return check("AhdGUISetWrap(" + handle + ", " + flag(0, "false") + ")"), true
	case "setBold":
		bold := "false"
		if arguments.present(0) {
			bold = generator.value(value.Arguments[0].Value, ir.Type{Kind: ir.BoolType}, false)
		}
		return check("AhdGUISetLabelBold(" + handle + ", " + bold + ")"), true
	case "setMinWidth":
		return check("AhdGUISetMinWidth(" + handle + ", " + arguments.integer(0, "int64(0)") + ")"), true
	case "setPreferredWidth":
		return check("AhdGUISetPreferredWidth(" + handle + ", " + arguments.integer(0, "int64(0)") + ")"), true
	case "setAlign":
		return check("AhdGUISetAlign(" + handle + ", " + arguments.text(0, `""`) + ", " + arguments.text(1, `""`) + ")"), true
	case "setSpan":
		return check("AhdGUISetSpan(" + handle + ", " + arguments.integer(0, "int64(1)") + ", " + arguments.integer(1, "int64(1)") + ")"), true
	}
	return "", false
}
