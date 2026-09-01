package lowering

import "ahdcode/internal/ir"

// KeyValueModuleID is the synthetic module that carries the KeyValue standard
// library's KeyValueError Class declaration into the IR. Like Lists, KeyValue
// publishes no data-carrying Class: it transforms the core Pair type and hands
// back ordinary Pair and List values.
const KeyValueModuleID = "builtin:KeyValue"

const keyValueErrorClassID = ir.ClassID(KeyValueModuleID + "::class::KeyValueError")

func keyValueModule(id ir.ModuleID, name, path string) *ir.Module {
	return collectionErrorModule(id, name, path, keyValueErrorClassID, "KeyValueError")
}
