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
	guiImageClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Image"}
	guiButtonClass    = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Button"}
	guiTextInputClass = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TextInput"}
	guiCheckboxClass  = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Checkbox"}
	// v2.0
	guiListBoxClass       = &types.ClassSymbol{ModuleID: guiModuleID, Name: "ListBox"}
	guiMenuClass          = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Menu"}
	guiMenuItemClass      = &types.ClassSymbol{ModuleID: guiModuleID, Name: "MenuItem"}
	guiSelectClass        = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Select"}
	guiTextAreaClass      = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TextArea"}
	guiPasswordInputClass = &types.ClassSymbol{ModuleID: guiModuleID, Name: "PasswordInput"}
	guiTableViewClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "TableView"}
	guiTabsClass          = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Tabs"}
	guiRadioGroupClass    = &types.ClassSymbol{ModuleID: guiModuleID, Name: "RadioGroup"}
	guiRadioButtonClass   = &types.ClassSymbol{ModuleID: guiModuleID, Name: "RadioButton"}
	guiProgressClass      = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Progress"}
	guiDateFieldClass     = &types.ClassSymbol{ModuleID: guiModuleID, Name: "DateField"}
	guiCanvasClass        = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Canvas"}
	guiTaskClass          = &types.ClassSymbol{ModuleID: guiModuleID, Name: "Task"}
)

// GUIErrorIdentity and the other identities expose the canonical GUI Classes
// to the lowering layer.
func GUIErrorIdentity() *types.ClassSymbol { return guiErrorClass }

// GUIClassNames lists the GUI Classes that hold a runtime handle, in
// declaration order.
var GUIClassNames = []string{"Window", "Container", "Label", "Button", "TextInput", "Checkbox",
	"ListBox", "Menu", "MenuItem", "Select", "TextArea", "PasswordInput", "TableView", "Image",
	"Tabs", "RadioGroup", "RadioButton", "Progress", "DateField", "Canvas", "Task"}

func guiClasses() []*types.ClassSymbol {
	return []*types.ClassSymbol{guiWindowClass, guiContainerClass, guiLabelClass, guiButtonClass, guiTextInputClass, guiCheckboxClass,
		guiListBoxClass, guiMenuClass, guiMenuItemClass, guiSelectClass, guiTextAreaClass, guiPasswordInputClass, guiTableViewClass, guiImageClass,
		guiTabsClass, guiRadioGroupClass, guiRadioButtonClass, guiProgressClass, guiDateFieldClass, guiCanvasClass, guiTaskClass}
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

func TaskWorkType() types.Type {
	return types.Function{Signature: &types.Signature{Return: types.Int}}
}

func TaskDoneType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "value", Type: types.Int}, {Name: "problem", Type: types.String}}, Return: types.Nothing}}
}

func CanvasPointerType() types.Type {
	return types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "x", Type: types.Real}, {Name: "y", Type: types.Real}}, Return: types.Nothing}}
}

// guiNullableCallbacks names the members whose callback receives a null
// selection, and which of its parameters are nullable: the callback must
// declare exactly those parameters nullable, so a program always handles "no
// selection".
var guiNullableCallbacks = map[TypeOperation][]bool{
	"ListBox.onChange":    {true, true},
	"Select.onChange":     {true, true},
	"TableView.onSelect":  {true},
	"RadioGroup.onChange": {true, true},
	"Task.onDone":         {true, true},
}

// The GUI members each Class publishes, so has/has not reports what a value
// really offers. Every member publishes real parameter names and defaults:
// a call is entirely positional or entirely named, like a module function.
var GUIOperations = map[string][]string{
	"Window": {"column", "row", "scroll", "grid", "onKey", "wait", "close", "isOpen", "setTitle", "setBackground",
		"setResizable", "isResizable", "menu", "every", "cancel", "task"},
	"Menu":     {"menu", "item", "checkItem", "separator"},
	"MenuItem": {"onClick", "setEnabled", "isEnabled", "setChecked", "checked", "setText"},
	"Container": {"column", "row", "grid", "scroll", "label", "image", "button", "textInput", "checkbox", "setBackground",
		"passwordInput", "textArea", "listBox", "select", "table", "tabs", "radioGroup", "progress", "date", "canvas",
		"setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Label": {"text", "setText", "setForeground", "setBackground", "setVisible", "isVisible",
		"setSize", "setBold", "setWrap", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Image": {"setSize", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Button":        {"text", "setText", "onClick", "setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"TextInput":     {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Checkbox":      {"checked", "setChecked", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"PasswordInput": {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"TextArea":      {"text", "setText", "onChange", "setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"ListBox": {"items", "setItems", "selectedIndex", "selectedText", "select", "onChange",
		"setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Select": {"items", "setItems", "selectedIndex", "selectedText", "select", "onChange",
		"setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"TableView": {"columns", "rows", "setRows", "selectedRow", "selectRow", "onSelect",
		"sort", "sortedColumn", "sortDescending",
		"setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Tabs": {"page", "select", "selectedIndex", "onChange", "setBackground", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"RadioGroup": {"option", "select", "selectedIndex", "selectedText", "onChange", "setBackground",
		"setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"RadioButton": {"text", "setText", "setForeground", "setBackground", "setEnabled", "isEnabled",
		"setVisible", "isVisible", "setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"Progress": {"setValue", "value", "maximum", "setIndeterminate", "isIndeterminate",
		"setForeground", "setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible",
		"setMinWidth", "setPreferredWidth", "setAlign", "setSpan"},
	"DateField": {"setDate", "year", "month", "day", "onChange", "setForeground", "setBackground",
		"setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth", "setPreferredWidth",
		"setAlign", "setSpan"},
	"Canvas": {"clear", "line", "rectangle", "circle", "text", "onMove", "onClick", "onDrag",
		"setBackground", "setEnabled", "isEnabled", "setVisible", "isVisible", "setMinWidth",
		"setPreferredWidth", "setAlign", "setSpan"},
	"Task": {"onDone", "cancel", "cancelled"},
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
		"Window.column": member(guiModuleID, "column", container, layout()...),
		"Window.row":    member(guiModuleID, "row", container, layout()...),
		"Window.scroll": member(guiModuleID, "scroll", container, layout()...),
		"Window.grid": member(guiModuleID, "grid", container, guiParameter("columns", types.Int, false),
			guiParameter("columnGap", types.Int, true), guiParameter("rowGap", types.Int, true)),
		"Window.onKey":    member(guiModuleID, "onKey", types.Nothing, guiParameter("handler", KeyHandlerType(), false)),
		"Window.wait":     member(guiModuleID, "wait", types.Nothing),
		"Window.close":    member(guiModuleID, "close", types.Nothing),
		"Window.isOpen":   member(guiModuleID, "isOpen", types.Bool),
		"Window.setTitle": member(guiModuleID, "setTitle", types.Nothing, text),

		"Container.column": member(guiModuleID, "column", container, layout()...),
		"Container.row":    member(guiModuleID, "row", container, layout()...),
		"Container.scroll": member(guiModuleID, "scroll", container, layout()...),
		"Container.grid": member(guiModuleID, "grid", container, guiParameter("columns", types.Int, false),
			guiParameter("columnGap", types.Int, true), guiParameter("rowGap", types.Int, true)),
		"Container.label": member(guiModuleID, "label", guiType(guiLabelClass), text),
		"Container.image": member(guiModuleID, "image", guiType(guiImageClass),
			guiParameter("path", types.String, false), guiParameter("width", types.Int, true), guiParameter("height", types.Int, true)),
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
	for _, class := range []string{"Label", "Button", "TextInput", "Checkbox", "PasswordInput", "TextArea", "ListBox", "Select", "TableView", "RadioButton", "Progress", "DateField"} {
		members[TypeOperation(class+".setForeground")] = member(guiModuleID, "setForeground", types.Nothing, color)
		members[TypeOperation(class+".setBackground")] = member(guiModuleID, "setBackground", types.Nothing, color)
	}
	for _, class := range []string{"Button", "TextInput", "Checkbox", "PasswordInput", "TextArea", "ListBox", "Select", "TableView", "Image", "RadioButton", "Progress", "DateField", "Canvas"} {
		members[TypeOperation(class+".setEnabled")] = member(guiModuleID, "setEnabled", types.Nothing, enabled)
		members[TypeOperation(class+".isEnabled")] = member(guiModuleID, "isEnabled", types.Bool)
	}
	visible := guiParameter("visible", types.Bool, false)
	for _, class := range []string{"Container", "Label", "Button", "TextInput", "Checkbox", "PasswordInput", "TextArea", "ListBox", "Select", "TableView", "Image", "Tabs", "RadioGroup", "RadioButton", "Progress", "DateField", "Canvas"} {
		members[TypeOperation(class+".setVisible")] = member(guiModuleID, "setVisible", types.Nothing, visible)
		members[TypeOperation(class+".isVisible")] = member(guiModuleID, "isVisible", types.Bool)
		members[TypeOperation(class+".setMinWidth")] = member(guiModuleID, "setMinWidth", types.Nothing, guiParameter("width", types.Int, false))
		members[TypeOperation(class+".setPreferredWidth")] = member(guiModuleID, "setPreferredWidth", types.Nothing, guiParameter("width", types.Int, false))
		members[TypeOperation(class+".setAlign")] = member(guiModuleID, "setAlign", types.Nothing,
			guiParameter("horizontal", types.String, false), guiParameter("vertical", types.String, true))
		members[TypeOperation(class+".setSpan")] = member(guiModuleID, "setSpan", types.Nothing,
			guiParameter("columns", types.Int, false), guiParameter("rows", types.Int, true))
	}
	members["Window.menu"] = member(guiModuleID, "menu", guiType(guiMenuClass), guiParameter("title", types.String, false))
	members["Menu.menu"] = member(guiModuleID, "menu", guiType(guiMenuClass), guiParameter("title", types.String, false))
	members["Menu.item"] = member(guiModuleID, "item", guiType(guiMenuItemClass),
		guiParameter("label", types.String, false), guiParameter("shortcut", types.String, true))
	members["Menu.checkItem"] = member(guiModuleID, "checkItem", guiType(guiMenuItemClass),
		guiParameter("label", types.String, false), guiParameter("shortcut", types.String, true), guiParameter("checked", types.Bool, true))
	members["Menu.separator"] = member(guiModuleID, "separator", types.Nothing)
	members["MenuItem.onClick"] = member(guiModuleID, "onClick", types.Nothing, guiParameter("handler", ClickCallbackType(), false))
	members["MenuItem.setEnabled"] = member(guiModuleID, "setEnabled", types.Nothing, enabled)
	members["MenuItem.isEnabled"] = member(guiModuleID, "isEnabled", types.Bool)
	members["MenuItem.setChecked"] = member(guiModuleID, "setChecked", types.Nothing, guiParameter("checked", types.Bool, false))
	members["MenuItem.checked"] = member(guiModuleID, "checked", types.Bool)
	members["MenuItem.setText"] = member(guiModuleID, "setText", types.Nothing, text)
	members["Label.setSize"] = member(guiModuleID, "setSize", types.Nothing, guiParameter("size", types.String, false))
	members["Label.setBold"] = member(guiModuleID, "setBold", types.Nothing, guiParameter("bold", types.Bool, false))
	members["Label.setWrap"] = member(guiModuleID, "setWrap", types.Nothing, guiParameter("wrap", types.Bool, false))
	members["Image.setSize"] = member(guiModuleID, "setSize", types.Nothing, guiParameter("width", types.Int, true), guiParameter("height", types.Int, true))
	members["Image.setBackground"] = member(guiModuleID, "setBackground", types.Nothing, color)

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
	members["TableView.sort"] = member(guiModuleID, "sort", types.Nothing,
		guiParameter("column", types.Int, false), guiParameter("descending", types.Bool, true))
	members["TableView.sortedColumn"] = nullableMember(member(guiModuleID, "sortedColumn", types.Int), nil, true)
	members["TableView.sortDescending"] = member(guiModuleID, "sortDescending", types.Bool)

	year := guiParameter("year", types.Int, false)
	month := guiParameter("month", types.Int, false)
	day := guiParameter("day", types.Int, false)
	members["Container.tabs"] = member(guiModuleID, "tabs", guiType(guiTabsClass))
	members["Container.radioGroup"] = member(guiModuleID, "radioGroup", guiType(guiRadioGroupClass))
	members["Container.progress"] = member(guiModuleID, "progress", guiType(guiProgressClass), guiParameter("maximum", types.Int, true))
	members["Container.date"] = member(guiModuleID, "date", guiType(guiDateFieldClass),
		guiParameter("year", types.Int, true), guiParameter("month", types.Int, true), guiParameter("day", types.Int, true))
	members["Tabs.page"] = member(guiModuleID, "page", container, guiParameter("title", types.String, false))
	members["Tabs.select"] = member(guiModuleID, "select", types.Nothing, guiParameter("index", types.Int, false))
	members["Tabs.selectedIndex"] = member(guiModuleID, "selectedIndex", types.Int)
	members["Tabs.onChange"] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", types.Function{Signature: &types.Signature{Parameters: []types.Parameter{{Name: "index", Type: types.Int}}, Return: types.Nothing}}, false))
	members["Tabs.setBackground"] = member(guiModuleID, "setBackground", types.Nothing, color)
	members["RadioGroup.option"] = member(guiModuleID, "option", guiType(guiRadioButtonClass), text)
	members["RadioGroup.select"] = nullableMember(member(guiModuleID, "select", types.Nothing, guiParameter("index", types.Int, false)), []bool{true}, false)
	members["RadioGroup.selectedIndex"] = nullableMember(member(guiModuleID, "selectedIndex", types.Int), nil, true)
	members["RadioGroup.selectedText"] = nullableMember(member(guiModuleID, "selectedText", types.String), nil, true)
	members["RadioGroup.onChange"] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", SelectionCallbackType(), false))
	members["RadioGroup.setBackground"] = member(guiModuleID, "setBackground", types.Nothing, color)
	members["RadioButton.text"] = member(guiModuleID, "text", types.String)
	members["RadioButton.setText"] = member(guiModuleID, "setText", types.Nothing, text)
	members["Progress.setValue"] = member(guiModuleID, "setValue", types.Nothing, guiParameter("value", types.Int, false))
	members["Progress.value"] = member(guiModuleID, "value", types.Int)
	members["Progress.maximum"] = member(guiModuleID, "maximum", types.Int)
	members["Progress.setIndeterminate"] = member(guiModuleID, "setIndeterminate", types.Nothing, guiParameter("indeterminate", types.Bool, false))
	members["Progress.isIndeterminate"] = member(guiModuleID, "isIndeterminate", types.Bool)
	members["DateField.setDate"] = member(guiModuleID, "setDate", types.Nothing, year, month, day)
	members["DateField.year"] = member(guiModuleID, "year", types.Int)
	members["DateField.month"] = member(guiModuleID, "month", types.Int)
	members["DateField.day"] = member(guiModuleID, "day", types.Int)
	members["DateField.onChange"] = member(guiModuleID, "onChange", types.Nothing, guiParameter("handler", types.Function{Signature: &types.Signature{
		Parameters: []types.Parameter{{Name: "year", Type: types.Int}, {Name: "month", Type: types.Int}, {Name: "day", Type: types.Int}},
		Return:     types.Nothing,
	}}, false))
	members["Window.every"] = member(guiModuleID, "every", types.Int, guiParameter("milliseconds", types.Int, false), guiParameter("handler", ClickCallbackType(), false))
	members["Window.cancel"] = member(guiModuleID, "cancel", types.Nothing, guiParameter("timer", types.Int, false))
	members["Window.task"] = member(guiModuleID, "task", guiType(guiTaskClass), guiParameter("work", TaskWorkType(), false))
	members["Task.onDone"] = member(guiModuleID, "onDone", types.Nothing, guiParameter("handler", TaskDoneType(), false))
	members["Task.cancel"] = member(guiModuleID, "cancel", types.Nothing)
	members["Task.cancelled"] = member(guiModuleID, "cancelled", types.Bool)
	members["Container.canvas"] = member(guiModuleID, "canvas", guiType(guiCanvasClass), guiParameter("width", types.Int, true), guiParameter("height", types.Int, true))
	members["Canvas.clear"] = member(guiModuleID, "clear", types.Nothing, guiParameter("color", types.String, true))
	members["Canvas.line"] = member(guiModuleID, "line", types.Nothing,
		guiParameter("x1", types.Real, false), guiParameter("y1", types.Real, false), guiParameter("x2", types.Real, false), guiParameter("y2", types.Real, false),
		guiParameter("stroke", types.String, true), guiParameter("width", types.Real, true))
	members["Canvas.rectangle"] = member(guiModuleID, "rectangle", types.Nothing,
		guiParameter("x", types.Real, false), guiParameter("y", types.Real, false), guiParameter("width", types.Real, false), guiParameter("height", types.Real, false),
		guiParameter("stroke", types.String, true), guiParameter("fill", types.String, true))
	members["Canvas.circle"] = member(guiModuleID, "circle", types.Nothing,
		guiParameter("x", types.Real, false), guiParameter("y", types.Real, false), guiParameter("radius", types.Real, false),
		guiParameter("stroke", types.String, true), guiParameter("fill", types.String, true))
	members["Canvas.text"] = member(guiModuleID, "text", types.Nothing,
		guiParameter("x", types.Real, false), guiParameter("y", types.Real, false), guiParameter("text", types.String, false),
		guiParameter("fill", types.String, true), guiParameter("size", types.Real, true))
	pointer := guiParameter("handler", CanvasPointerType(), false)
	members["Canvas.onMove"] = member(guiModuleID, "onMove", types.Nothing, pointer)
	members["Canvas.onClick"] = member(guiModuleID, "onClick", types.Nothing, pointer)
	members["Canvas.onDrag"] = member(guiModuleID, "onDrag", types.Nothing, pointer)
	members["Canvas.setBackground"] = member(guiModuleID, "setBackground", types.Nothing, color)
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
		return "create a Container with window.column(), window.row(), container.column()/row(), or tabs.page()", true
	case "RadioButton":
		return "create a RadioButton with radioGroup.option", true
	case "Task":
		return "start a background task with window.task", true
	case "GUIError":
		return "", false
	case "Menu":
		return "create a Menu with window.menu or menu.menu", true
	case "MenuItem":
		return "create a MenuItem with menu.item or menu.checkItem", true
	}
	return "create a " + identity.Name + " inside a Container, as in container." + map[string]string{
		"Label": "label", "Button": "button", "TextInput": "textInput", "Checkbox": "checkbox",
		"ListBox": "listBox", "Select": "select", "TextArea": "textArea", "PasswordInput": "passwordInput",
		"TableView": "table", "Image": "image", "Tabs": "tabs", "RadioGroup": "radioGroup",
		"Progress": "progress", "DateField": "date", "Canvas": "canvas"}[identity.Name] + "(...)", true
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
	addStandardExport(module, standardFunction(guiModuleID, "clipboard", types.String))
	addStandardExport(module, standardFunction(guiModuleID, "setClipboard", types.Nothing,
		types.Parameter{Name: "text", Type: types.String}))
	addStandardExport(module, standardFunction(guiModuleID, "checkpoint", types.Bool))
	addStandardExport(module, standardFunction(guiModuleID, "resource", types.String,
		types.Parameter{Name: "name", Type: types.String}))

	sort.Strings(module.ExportNames)
	return module
}
