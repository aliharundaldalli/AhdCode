package evaluator

import (
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
	if name != "window" {
		session.raise("Error", "unsupported GUI function "+name)
	}
	handle, problem := ahdruntime.AhdGUIWindowOpen(graphicsText(arguments, 0, "AhdCode"), graphicsInt(arguments, 1, 800), graphicsInt(arguments, 2, 600))
	session.guiCheck(problem)
	return guiInstance("Window", handle)
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
	default:
		session.raise("Error", "unsupported GUI operation "+name)
	}
	return Nothing
}

// CloseGUI closes every open Window; the REPL calls it when it ends.
func CloseGUI() { ahdruntime.AhdGUICloseAll() }
