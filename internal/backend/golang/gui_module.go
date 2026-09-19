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

// isGUIOperation reports whether a type operation belongs to a GUI Class.
func isGUIOperation(name string) bool {
	for _, class := range []string{"Window.", "Container.", "Label.", "Button.", "TextInput.", "Checkbox."} {
		if strings.HasPrefix(name, class) {
			return true
		}
	}
	return false
}

// guiCall lowers GUI.window.
func (generator *generator) guiCall(value *ir.CallExpr) string {
	meta := value.ExprMeta()
	name := strings.TrimPrefix(string(value.Callable), guiModulePrefix)
	arguments := graphicsArguments{generator: generator, value: value}
	if name != "window" {
		return generator.unsupported("GUI function "+name, meta.Span)
	}
	generator.usesGUI = true
	handle := "AhdGUIWindowChecked(" + arguments.text(0, `"AhdCode"`) + ", " + arguments.integer(1, "int64(800)") + ", " + arguments.integer(2, "int64(600)") + ")"
	return generator.graphicsValue(guiClass("Window"), handle, meta)
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
	return generator.unsupported("GUI operation "+name, meta.Span)
}
