package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const guiModuleID = "builtin:GUI"

var (
	guiErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	guiErrorClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "GUIError", Parent: guiErrorParent}
	guiWindowClass    = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Window"}
	guiContainerClass = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Container"}
	guiLabelClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Label"}
	guiButtonClass    = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Button"}
	guiTextInputClass = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TextInput"}
	guiCheckboxClass  = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Checkbox"}
)

// GUIErrorIdentity and the other identities expose the canonical GUI Classes
// to the lowering layer.
func GUIErrorIdentity() *types.ClassSymbol { return guiErrorClass }

// GUIClassNames lists the GUI Classes that hold a runtime handle, in
// declaration order.
var GUIClassNames = []string{"Window", "Container", "Label", "Button", "TextInput", "Checkbox"}

func guiClasses() []*types.ClassSymbol {
	return []*types.ClassSymbol{guiWindowClass, guiContainerClass, guiLabelClass, guiButtonClass, guiTextInputClass, guiCheckboxClass}
}

func guiType(class *types.ClassSymbol) types.Type { return types.Class{Symbol: class} }

// ClickCallbackType is the () -> Nothing shape of a Button click callback.
func ClickCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Return: types.Nothing}}
}

// The GUI members each Class publishes, so has/has not reports what a value
// really offers. Every member publishes real parameter names and defaults:
// a call is entirely positional or entirely named, like a module function.
var GUIOperations = map[string][]string{
	"Window":    {"column", "row", "onKey", "wait", "close", "isOpen", "setTitle"},
	"Container": {"column", "row", "label", "button", "textInput", "checkbox"},
	"Label":     {"text", "setText"},
	"Button":    {"text", "setText", "onClick"},
	"TextInput": {"text", "setText"},
	"Checkbox":  {"checked", "setChecked"},
}

func guiParameter(name string, typ types.Type, hasDefault bool) types.Parameter {
	return types.Parameter{Name: name, Type: typ, HasDefault: hasDefault}
}

// guiMembers maps every "Class.member" operation to its Symbol.
var guiMembers = func() map[TypeOperation]*Symbol {
	container := guiType(guiContainerClass)
	text := guiParameter("text", types.String, false)
	// Window.column/row default to padding 12 and nested Containers to 0;
	// the default values live with the runtime calls, as for Graphics.
	layout := func() []types.Parameter {
		return []types.Parameter{guiParameter("spacing", types.Int, true), guiParameter("padding", types.Int, true)}
	}
	member := func(module string, name string, result types.Type, parameters ...types.Parameter) *Symbol {
		return completionMember(module, name, result, parameters...)
	}
	return map[TypeOperation]*Symbol{
		"Window.column":   member(guiModuleID, "column", container, layout()...),
		"Window.row":      member(guiModuleID, "row", container, layout()...),
		"Window.onKey":    member(guiModuleID, "onKey", types.Nothing, guiParameter("handler", KeyHandlerType(), false)),
		"Window.wait":     member(guiModuleID, "wait", types.Nothing),
		"Window.close":    member(guiModuleID, "close", types.Nothing),
		"Window.isOpen":   member(guiModuleID, "isOpen", types.Bool),
		"Window.setTitle": member(guiModuleID, "setTitle", types.Nothing, text),

		"Container.column":    member(guiModuleID, "column", container, layout()...),
		"Container.row":       member(guiModuleID, "row", container, layout()...),
		"Container.label":     member(guiModuleID, "label", guiType(guiLabelClass), text),
		"Container.button":    member(guiModuleID, "button", guiType(guiButtonClass), text),
		"Container.textInput": member(guiModuleID, "textInput", guiType(guiTextInputClass), guiParameter("placeholder", types.String, true)),
		"Container.checkbox":  member(guiModuleID, "checkbox", guiType(guiCheckboxClass), text, guiParameter("checked", types.Bool, true)),

		"Label.text":          member(guiModuleID, "text", types.String),
		"Label.setText":       member(guiModuleID, "setText", types.Nothing, text),
		"Button.text":         member(guiModuleID, "text", types.String),
		"Button.setText":      member(guiModuleID, "setText", types.Nothing, text),
		"Button.onClick":      member(guiModuleID, "onClick", types.Nothing, guiParameter("handler", ClickCallbackType(), false)),
		"TextInput.text":      member(guiModuleID, "text", types.String),
		"TextInput.setText":   member(guiModuleID, "setText", types.Nothing, text),
		"Checkbox.checked":    member(guiModuleID, "checked", types.Bool),
		"Checkbox.setChecked": member(guiModuleID, "setChecked", types.Nothing, guiParameter("checked", types.Bool, false)),
	}
}()

// guiOperationFor names the built-in member a GUI value publishes. Only the
// compiler-supplied identities match.
func guiOperationFor(receiver types.Type, name string) (TypeOperation, bool) {
	class, ok := receiver.(types.Class)
	if !ok || class.Reference || class.Symbol == nil || class.Symbol.ModuleID != guiModuleID {
		return "", false
	}
	operation := TypeOperation(class.Symbol.Name + "." + name)
	_, known := guiMembers[operation]
	return operation, known
}

// guiConstructionHint names how each GUI value is obtained, since none of the
// Classes publishes a constructor.
func guiConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != guiModuleID {
		return "", false
	}
	switch identity.Name {
	case "Window":
		return "open a Window with GUI.window", true
	case "Container":
		return "create a Container with window.column(), window.row(), or container.column()/row()", true
	case "GUIError":
		return "", false
	}
	return "create a " + identity.Name + " inside a Container, as in container." + map[string]string{
		"Label": "label", "Button": "button", "TextInput": "textInput", "Checkbox": "checkbox"}[identity.Name] + "(...)", true
}

func guiModuleInterface() *ModuleInterface {
	module := standardInterface(guiModuleID, "GUI")

	errorSymbol := &Symbol{
		Name: "GUIError", Kind: ClassSymbol, Class: guiErrorClass,
		Type: types.Class{Symbol: guiErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: guiModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[guiModuleID+"\x00GUIError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	for _, class := range guiClasses() {
		symbol := &Symbol{
			Name: class.Name, Kind: ClassSymbol, Class: class,
			Type: types.Class{Symbol: class, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: guiModuleID,
			Members: make(map[string]*Symbol),
		}
		module.Classes[guiModuleID+"\x00"+class.Name] = symbol
		addStandardExport(module, symbol)
	}

	addStandardExport(module, standardFunction(guiModuleID, "window", guiType(guiWindowClass),
		types.Parameter{Name: "title", Type: types.String, HasDefault: true},
		types.Parameter{Name: "width", Type: types.Int, HasDefault: true},
		types.Parameter{Name: "height", Type: types.Int, HasDefault: true}))

	sort.Strings(module.ExportNames)
	return module
}
