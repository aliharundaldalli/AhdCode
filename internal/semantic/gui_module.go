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
	// v2.0
	guiListBoxClass       = &types.ClassSymbol{ModuleID: guiModuleID, Name: "ListBox"}
	guiSelectClass        = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Select"}
	guiTextAreaClass      = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TextArea"}
	guiPasswordInputClass = &types.ClassSymbol{ModuleID: guiModuleID, Name: "PasswordInput"}
	guiTableViewClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TableView"}
)

// GUIErrorIdentity and the other identities expose the canonical GUI Classes
// to the lowering layer.
func GUIErrorIdentity() *types.ClassSymbol { return guiErrorClass }

// GUIClassNames lists the GUI Classes that hold a runtime handle, in
// declaration order.
var GUIClassNames = []string{"Window", "Container", "Label", "Button", "TextInput", "Checkbox",
	"ListBox", "Select", "TextArea", "PasswordInput", "TableView"}

func guiClasses() []*types.ClassSymbol {
	return []*types.ClassSymbol{guiWindowClass, guiContainerClass, guiLabelClass, guiButtonClass, guiTextInputClass, guiCheckboxClass,
		guiListBoxClass, guiSelectClass, guiTextAreaClass, guiPasswordInputClass, guiTableViewClass}
}

func guiType(class *types.ClassSymbol) types.Type { return types.Class{Symbol: class} }

// ClickCallbackType is the () -> Nothing shape of a Button click callback.
func ClickCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Return: types.Nothing}}
}

// TextChangeCallbackType is (text: String) -> Nothing, CheckChangeCallbackType
// (checked: Bool) -> Nothing, SelectionCallbackType (index: Int?, text:
// String?) -> Nothing, and RowCallbackType (row: Int?) -> Nothing. A Function
// type does not spell nullability; guiNullableCallbacks below requires the
// callback to declare the Int? and String? parameters nullable.
func TextChangeCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "text", Type: types.String}}, Return: types.Nothing}}
}

func CheckChangeCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "checked", Type: types.Bool}}, Return: types.Nothing}}
}

func SelectionCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "index", Type: types.Int}, {Name: "text", Type: types.String}}, Return: types.Nothing}}
}

func RowCallbackType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "row", Type: types.Int}}, Return: types.Nothing}}
}

// guiNullableCallbacks names the members whose callback receives a null
// selection, and which of its parameters are nullable: the callback must
// declare exactly those parameters nullable, so a program always handles "no
// selection".
var guiNullableCallbacks = map[TypeOperation][]bool{
	"ListBox.onChange":   {true, true},
	"Select.onChange":    {true, true},
	"TableView.onSelect": {true},
}

// The GUI members each Class publishes, so has/has not reports what a value
// really offers. Every member publishes real parameter names and defaults:
// a call is entirely positional or entirely named, like a module function.
var GUIOperations = map[string][]string{
	"Window": {"column", "row", "onKey", "wait", "close", "isOpen", "setTitle", "setBackground",
		"setResizable", "isResizable"},
	"Container": {"column", "row", "label", "button", "textInput", "checkbox", "setBackground",
		"passwordInput", "textArea", "listBox", "select", "table"},
	"Label":         {"text", "setText", "setForeground", "setBackground"},
	"Button":        {"text", "setText", "onClick", "setForeground", "setBackground", "setEnabled", "isEnabled"},
	"TextInput":     {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"},
	"Checkbox":      {"checked", "setChecked", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"},
	"PasswordInput": {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"},
	"TextArea":      {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled"},
	"ListBox": {"items", "setItems", "selectedIndex", "selectedText", "select", "onChange",
		"setForeground", "setBackground", "setEnabled", "isEnabled"},
	"Select": {"items", "setItems", "selectedIndex", "selectedText", "select", "onChange",
		"setForeground", "setBackground", "setEnabled", "isEnabled"},
	"TableView": {"columns", "rows", "setRows", "selectedRow", "selectRow", "onSelect",
		"setForeground", "setBackground", "setEnabled", "isEnabled"},
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
	color := guiParameter("color", types.String, false)
	enabled := guiParameter("enabled", types.Bool, false)
	members := map[TypeOperation]*Symbol{
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

		"Window.setBackground":    member(guiModuleID, "setBackground", types.Nothing, color),
		"Container.setBackground": member(guiModuleID, "setBackground", types.Nothing, color),
	}
	// Colors take the Graphics spellings; only the interactive widgets can be
	// disabled.
	for _, class := range []string{"Label", "Button", "TextInput", "Checkbox", "PasswordInput", "TextArea", "ListBox", "Select", "TableView"} {
		members[TypeOperation(class+".setForeground")] = member(guiModuleID, "setForeground", types.Nothing, color)
		members[TypeOperation(class+".setBackground")] = member(guiModuleID, "setBackground", types.Nothing, color)
	}
	for _, class := range []string{"Button", "TextInput", "Checkbox", "PasswordInput", "TextArea", "ListBox", "Select", "TableView"} {
		members[TypeOperation(class+".setEnabled")] = member(guiModuleID, "setEnabled", types.Nothing, enabled)
		members[TypeOperation(class+".isEnabled")] = member(guiModuleID, "isEnabled", types.Bool)
	}

	// v2.0 widgets, change callbacks, and resizable Windows.
	strings := types.List{Element: types.String}
	rows := types.List{Element: strings}
	placeholder := guiParameter("placeholder", types.String, true)
	members["Window.setResizable"] = member(guiModuleID, "setResizable", types.Nothing, guiParameter("resizable", types.Bool, false))
	members["Window.isResizable"] = member(guiModuleID, "isResizable", types.Bool)
	members["Container.passwordInput"] = member(guiModuleID, "passwordInput", guiType(guiPasswordInputClass), placeholder)
	members["Container.textArea"] = member(guiModuleID, "textArea", guiType(guiTextAreaClass), placeholder)
	members["Container.listBox"] = member(guiModuleID, "listBox", guiType(guiListBoxClass), guiParameter("items", strings, true))
	members["Container.select"] = nullableMember(member(guiModuleID, "select", guiType(guiSelectClass),
		guiParameter("items", strings, false), guiParameter("selectedIndex", types.Int, true)), []bool{false, true}, false)
	members["Container.table"] = member(guiModuleID, "table", guiType(guiTableViewClass),
		guiParameter("columns", strings, false), guiParameter("rows", rows, true))
	for _, class := range []string{"TextInput", "PasswordInput", "TextArea"} {
		members[TypeOperation(class+".text")] = member(guiModuleID, "text", types.String)
		members[TypeOperation(class+".setText")] = member(guiModuleID, "setText", types.Nothing, text)
		members[TypeOperation(class+".onChange")] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", TextChangeCallbackType(), false))
	}
	members["Checkbox.onChange"] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", CheckChangeCallbackType(), false))
	for _, class := range []string{"ListBox", "Select"} {
		members[TypeOperation(class+".items")] = member(guiModuleID, "items", strings)
		members[TypeOperation(class+".setItems")] = member(guiModuleID, "setItems", types.Nothing, guiParameter("items", strings, false))
		members[TypeOperation(class+".selectedIndex")] = nullableMember(member(guiModuleID, "selectedIndex", types.Int), nil, true)
		members[TypeOperation(class+".selectedText")] = nullableMember(member(guiModuleID, "selectedText", types.String), nil, true)
		members[TypeOperation(class+".select")] = nullableMember(member(guiModuleID, "select", types.Nothing,
			guiParameter("index", types.Int, false)), []bool{true}, false)
		members[TypeOperation(class+".onChange")] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", SelectionCallbackType(), false))
	}
	members["TableView.columns"] = member(guiModuleID, "columns", strings)
	members["TableView.rows"] = member(guiModuleID, "rows", rows)
	members["TableView.setRows"] = member(guiModuleID, "setRows", types.Nothing, guiParameter("rows", rows, false))
	members["TableView.selectedRow"] = nullableMember(member(guiModuleID, "selectedRow", types.Int), nil, true)
	members["TableView.selectRow"] = nullableMember(member(guiModuleID, "selectRow", types.Nothing,
		guiParameter("index", types.Int, false)), []bool{true}, false)
	members["TableView.onSelect"] = member(guiModuleID, "onSelect", types.Nothing, guiParameter("handler", RowCallbackType(), false))
	return members
}()

// nullableMember marks which parameters of a published member accept null
// and whether it returns a nullable value.
func nullableMember(symbol *Symbol, parameters []bool, result bool) *Symbol {
	for index, nullable := range parameters {
		if nullable {
			symbol.Callable.ParameterNull[index] = MaybeNull
		}
	}
	if result {
		symbol.Callable.ReturnNull = MaybeNull
	}
	return symbol
}

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
		"Label": "label", "Button": "button", "TextInput": "textInput", "Checkbox": "checkbox",
		"ListBox": "listBox", "Select": "select", "TextArea": "textArea", "PasswordInput": "passwordInput",
		"TableView": "table"}[identity.Name] + "(...)", true
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

	// v2.0 dialogs. A cancelled file dialog returns null (openFiles: an
	// empty List); GUIError means the dialog could not be shown.
	title := func() types.Parameter { return types.Parameter{Name: "title", Type: types.String, HasDefault: true} }
	extensions := func() types.Parameter {
		return types.Parameter{Name: "extensions", Type: types.List{Element: types.String}, HasDefault: true}
	}
	nullable := func(symbol *Symbol) *Symbol {
		symbol.Callable.ReturnNull = MaybeNull
		return symbol
	}
	addStandardExport(module, nullable(standardFunction(guiModuleID, "openFile", types.String, title(), extensions())))
	addStandardExport(module, standardFunction(guiModuleID, "openFiles", types.List{Element: types.String}, title(), extensions()))
	addStandardExport(module, nullable(standardFunction(guiModuleID, "selectFolder", types.String, title())))
	addStandardExport(module, nullable(standardFunction(guiModuleID, "saveFile", types.String, title(),
		types.Parameter{Name: "suggestedName", Type: types.String, HasDefault: true}, extensions())))
	addStandardExport(module, standardFunction(guiModuleID, "message", types.Nothing,
		types.Parameter{Name: "title", Type: types.String}, types.Parameter{Name: "text", Type: types.String}))
	addStandardExport(module, standardFunction(guiModuleID, "confirm", types.Bool,
		types.Parameter{Name: "title", Type: types.String}, types.Parameter{Name: "text", Type: types.String}))

	sort.Strings(module.ExportNames)
	return module
}
