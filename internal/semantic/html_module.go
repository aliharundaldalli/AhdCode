package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const htmlModuleID = "builtin:HTML"

var (
	htmlErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
		Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
	htmlErrorClass = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLError", Parent: htmlErrorParent}
	htmlNodeClass  = &types.ClassSymbol{ModuleID: htmlModuleID, Name: "HTMLNode"}
)

// HTMLErrorIdentity and HTMLNodeIdentity expose the canonical identities to
// lowering without coupling the public module interface to a backend.
func HTMLErrorIdentity() *types.ClassSymbol { return htmlErrorClass }
func HTMLNodeIdentity() *types.ClassSymbol  { return htmlNodeClass }

// HTMLNodeOperations is empty: HTMLNode is an immutable builder value with no
// instance members in v0.4.0. The slice exists so has/has not and the IR Class
// stay in agreement with the frontend.
var HTMLNodeOperations = []string{}

func htmlNodeType() types.Type { return types.Class{Symbol: htmlNodeClass} }

func htmlModuleInterface() *ModuleInterface {
	module := standardInterface(htmlModuleID, "HTML")
	classes := []struct {
		name     string
		identity *types.ClassSymbol
	}{
		{"HTMLError", htmlErrorClass}, {"HTMLNode", htmlNodeClass},
	}
	for _, entry := range classes {
		symbol := &Symbol{
			Name: entry.name, Kind: ClassSymbol, Class: entry.identity,
			Type: types.Class{Symbol: entry.identity, Reference: true}, ModuleRoot: true,
			Builtin: true, InitialNull: NonNull, OriginModuleID: htmlModuleID,
			Members: make(map[string]*Symbol),
		}
		if entry.name == "HTMLError" {
			symbol.Constructor = builtinErrorConstructor()
		}
		module.Classes[htmlModuleID+"\x00"+entry.name] = symbol
		addStandardExport(module, symbol)
	}
	node := htmlNodeType()
	attributes := types.Pair{Key: types.String, Value: types.String}
	children := types.List{Element: node}
	addStandardExport(module, standardFunction(htmlModuleID, "text", node,
		types.Parameter{Name: "value", Type: types.String}))
	addStandardExport(module, standardFunction(htmlModuleID, "element", node,
		types.Parameter{Name: "name", Type: types.String},
		types.Parameter{Name: "attributes", Type: attributes},
		types.Parameter{Name: "children", Type: children}))
	addStandardExport(module, standardFunction(htmlModuleID, "render", types.String,
		types.Parameter{Name: "node", Type: node}))
	addStandardExport(module, standardFunction(htmlModuleID, "document", types.String,
		types.Parameter{Name: "title", Type: types.String},
		types.Parameter{Name: "body", Type: children}))
	sort.Strings(module.ExportNames)
	return module
}

func htmlConstructionHint(identity *types.ClassSymbol) (string, bool) {
	if identity == nil || identity.ModuleID != htmlModuleID {
		return "", false
	}
	if identity.Name == "HTMLNode" {
		return "create an HTMLNode with HTML.text(value) or HTML.element(name, attributes, children)", true
	}
	return "", false
}
