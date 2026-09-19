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

// guiClassNames are the GUI Classes whose members are type operations.
var guiClassNames = []string{"Window.", "Container.", "Label.", "Button.", "TextInput.", "Checkbox.",
	"ListBox.", "Select.", "TextArea.", "PasswordInput.", "TableView."}

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
	handler := func(params ...ir.Type) string {
		if len(value.Arguments) != 1 || value.Arguments[0].Value == nil {
			return generator.unsupported(name+" with a malformed argument list", meta.Span)
		}
		return generator.adaptHandler(value.Arguments[0].Value, params...)
	}
	switch name {
	case "Window.column", "Window.row":
		return child("Container", "AhdGUIRootChecked("+handle+", "+`"`+strings.TrimPrefix(name, "Window.")+`", `+arguments.integer(0, "int64(8)")+", "+arguments.integer(1, "int64(12)")+")")
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
	case "Container.label":
		return child("Label", "AhdGUIAddLeafChecked("+handle+`, "label", `+arguments.text(0, `""`)+`, "", false)`)
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
	case "ListBox.selectedIndex", "Select.selectedIndex":
		return "AhdGUISelectedIndexChecked(" + handle + ")", true
	case "ListBox.selectedText", "Select.selectedText":
		return "AhdGUISelectedTextChecked(" + handle + ")", true
	case "ListBox.select", "Select.select":
		return "AhdGUISelectChecked(" + handle + ", " + nullableInt(0) + ")", true
	case "ListBox.onChange", "Select.onChange":
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
	}
	return "", false
}
