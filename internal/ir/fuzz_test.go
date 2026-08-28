package ir

import "testing"

func FuzzValidatorNeverPanics(f *testing.F) {
	f.Add(byte(0), "")
	f.Add(byte(1), "unknown")
	f.Fuzz(func(t *testing.T, kind byte, identity string) {
		kinds := []TypeKind{"", InvalidType, IntType, RealType, BoolType, StringType, NothingType, ListType, PairType, FunctionType, ClassType}
		typeKind := kinds[int(kind)%len(kinds)]
		compilation := &Compilation{Entry: ModuleID(identity), Modules: []*Module{{ID: ModuleID(identity), Globals: []*Global{{ID: SymbolID(identity), Type: Type{Kind: typeKind}}}}}}
		_ = Validate(compilation)
		_ = Dump(compilation)
	})
}
