package semantic

import (
	"strings"
	"testing"

	"ahdcode/internal/types"
)

// `name: Constant := value` infers its type from the value. v2.7.0 left the
// predeclared module Constant without a type, which surfaced as IR002.
func TestInferredConstantTakesItsInitializerType(t *testing.T) {
	_, result := analyzeText(t, "radius: Constant := 20.0\nnames: Constant := [\"a\"]\n"+
		"area: Function := () -> Real uses [@radius] {\n    half: Local Constant := 2\n    return radius * radius / half\n}\nwrite(area())\n")
	if result.HasErrors() {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	found := map[string]*Symbol{}
	for _, symbol := range result.Symbols {
		found[symbol.Name] = symbol
	}
	if radius := found["radius"]; radius == nil || !radius.Constant || !types.Equal(radius.Type, types.Real) {
		t.Fatalf("radius = %+v", found["radius"])
	}
	if half := found["half"]; half == nil || !half.Constant || !types.Equal(half.Type, types.Int) {
		t.Fatalf("half = %+v", found["half"])
	}

	for _, text := range []string{
		"radius: Constant := 20.0\nradius = 3.0\n",
		"names: Constant := [\"a\"]\nnames.add(\"b\")\n",
		"f: Function := () -> Nothing {\n    half: Local Constant := 2\n    half = 3\n}\n",
	} {
		_, rejected := analyzeText(t, text)
		if len(rejected.Diagnostics) != 1 || rejected.Diagnostics[0].Code != codeConstantAssignment {
			t.Fatalf("%q: %+v", text, rejected.Diagnostics)
		}
	}
}

// An inferred Constant is held to the same constant-expression rule as a
// Constant whose type is written out. A later Constant is resolved. A cycle
// is SEM010, never an untyped internal error.
func TestInferredConstantFollowsTheConstantExpressionRule(t *testing.T) {
	_, mutable := analyzeText(t, "x: Int := 5\ny: Constant := x\n")
	requireSemanticCode(t, mutable, codeConstantInitializer)
	requireSemanticText(t, mutable, "not a scalar constant expression")

	_, forward := analyzeText(t, "y: Constant := x\nx: Constant := 5\n")
	requireSemanticClean(t, forward)
	found := symbolsByName(forward)
	if !types.Equal(found["y"].Type, types.Int) || !types.Equal(found["x"].Type, types.Int) {
		t.Fatalf("types y=%s x=%s", types.Display(found["y"].Type), types.Display(found["x"].Type))
	}
	if constantInt(t, found["y"]) != 5 || constantInt(t, found["x"]) != 5 {
		t.Fatalf("values y=%d x=%d", constantInt(t, found["y"]), constantInt(t, found["x"]))
	}

	_, derived := analyzeText(t, "y: Constant := x + 1\nx: Constant := 5\n")
	requireSemanticClean(t, derived)
	derivedSymbols := symbolsByName(derived)
	if constantInt(t, derivedSymbols["y"]) != 6 || constantInt(t, derivedSymbols["x"]) != 5 {
		t.Fatalf("derived y=%d x=%d", constantInt(t, derivedSymbols["y"]), constantInt(t, derivedSymbols["x"]))
	}

	_, typedForward := analyzeText(t, "a: Constant Int := b\nb: Constant := 5\n")
	requireSemanticClean(t, typedForward)
	if constantInt(t, symbolsByName(typedForward)["a"]) != 5 {
		t.Fatal(symbolsByName(typedForward)["a"])
	}

	_, reference := analyzeText(t, "b: Constant := a\na: Constant := [1, 2]\n")
	requireSemanticClean(t, reference)
	if symbolsByName(reference)["b"].Type.Kind() != types.ListKind {
		t.Fatalf("forward reference constant: %+v", symbolsByName(reference)["b"])
	}

	_, local := analyzeText(t, "f: Function := () -> Int {\n    n: Local Int := 1\n    c: Local Constant := n\n    return c\n}\n")
	requireSemanticCode(t, local, codeConstantInitializer)
	requireSemanticText(t, local, "not a scalar constant expression")

	_, nullConstant := analyzeText(t, "y: Constant := null\n")
	requireSemanticCode(t, nullConstant, codeConstantInitializer)
	requireSemanticText(t, nullConstant, "cannot be null")

	_, cyclic := analyzeText(t, "y: Constant := x\nx: Constant := y\n")
	requireSemanticCode(t, cyclic, codeConstantInitializer)
	requireSemanticText(t, cyclic, "cyclic")
	for _, diagnostic := range cyclic.Diagnostics {
		if diagnostic.Code == "IR002" {
			t.Fatalf("internal IR002 leaked: %+v", cyclic.Diagnostics)
		}
	}
}

// A later constant used through uses or Global keeps that constant's type.
// A reference cycle is SEM010. A real type mismatch stays SEM004.
func TestForwardConstantThroughUsesAndGlobal(t *testing.T) {
	_, uses := analyzeText(t, "area: Function := () -> Int uses [@radius] {\n    return radius * radius\n}\nradius: Constant := 20\n")
	requireSemanticClean(t, uses)
	radius := moduleConstant(t, uses, "radius")
	if !types.Equal(radius.Type, types.Int) || constantInt(t, radius) != 20 {
		t.Fatalf("radius = %+v", radius)
	}
	alias := constantAlias(uses, "radius")
	if alias == nil || !types.Equal(alias.Type, types.Int) || alias.Alias != radius {
		t.Fatalf("uses alias = %+v", alias)
	}

	_, derived := analyzeText(t, "scale: Function := () -> Int uses [@base] {\n    edge: Local Constant := base + 1\n    return edge\n}\nbase: Constant := 5\n")
	requireSemanticClean(t, derived)
	if constantInt(t, symbolsByName(derived)["edge"]) != 6 {
		t.Fatalf("edge = %+v", symbolsByName(derived)["edge"])
	}

	_, global := analyzeText(t, "scale: Function := () -> Int {\n    radius: Global\n    return radius + 1\n}\nradius: Constant := 20\n")
	requireSemanticClean(t, global)
	globalAlias := constantAlias(global, "radius")
	if globalAlias == nil || !types.Equal(globalAlias.Type, types.Int) {
		t.Fatalf("Global alias = %+v", globalAlias)
	}

	_, mismatch := analyzeText(t, "scale: Function := () -> Int {\n    radius: Global Int\n    return radius\n}\nradius: Constant := 20.0\n")
	requireSemanticCode(t, mismatch, codeTypeMismatch)

	for _, text := range []string{
		"a: Constant List<Int> := b\nb: Constant List<Int> := a\n",
		"a: Constant := [b]\nb: Constant := [a]\n",
		"a: Constant := a\n",
	} {
		_, cyclic := analyzeText(t, text)
		requireSemanticCode(t, cyclic, codeConstantInitializer)
		requireSemanticText(t, cyclic, "cyclic")
		for _, diagnostic := range cyclic.Diagnostics {
			if diagnostic.Code == "IR002" || diagnostic.Code == "IR005" {
				t.Fatalf("%q leaked %s: %+v", text, diagnostic.Code, cyclic.Diagnostics)
			}
		}
	}
}

// A function declaration is not a constant-value dependency. A call stays
// where it is written. A forward constant that reads a mutable binding cannot
// be observed before that binding's initializer has run.
func TestConstantInitializationContract(t *testing.T) {
	_, call := analyzeText(t, "items: Constant := answer()\nanswer: Function := () -> List<Int> {\n    return [7]\n}\n")
	requireSemanticClean(t, call)

	_, scalarCall := analyzeText(t, "n: Constant := answer()\nanswer: Function := () -> Int {\n    return 7\n}\n")
	requireSemanticCode(t, scalarCall, codeConstantInitializer)
	requireSemanticText(t, scalarCall, "not a scalar constant expression")

	_, earlyCall := analyzeText(t, "write(items[0])\nitems: Constant := answer()\nanswer: Function := () -> List<Int> {\n    return [7]\n}\n")
	requireSemanticCode(t, earlyCall, codeConstantInitializer)
	requireSemanticText(t, earlyCall, "used before it can be initialized")
	if !strings.Contains(earlyCall.Diagnostics[0].Hint, "calls a function") {
		t.Fatalf("hint = %q", earlyCall.Diagnostics[0].Hint)
	}

	_, anchored := analyzeText(t, "v: Int := 1\nb: Constant := [a]\nwrite(b[0][0])\na: Constant := [v]\n")
	requireSemanticCode(t, anchored, codeConstantInitializer)
	requireSemanticText(t, anchored, "used before it can be initialized")
	if !strings.Contains(anchored.Diagnostics[0].Hint, "reads a mutable binding") {
		t.Fatalf("hint = %q", anchored.Diagnostics[0].Hint)
	}
	for _, diagnostic := range anchored.Diagnostics {
		if diagnostic.Code == "IR002" || diagnostic.Code == "LWR002" {
			t.Fatalf("internal diagnostic leaked: %+v", anchored.Diagnostics)
		}
	}

	_, safe := analyzeText(t, "v: Int := 1\nb: Constant := [a]\na: Constant := [v]\nwrite(b[0][0])\n")
	requireSemanticClean(t, safe)

	_, binding := analyzeText(t, "handler: Function := answer\nitems: Constant := handler()\nanswer: Function := () -> List<Int> {\n    return [4]\n}\n")
	requireSemanticClean(t, binding)
}

func moduleConstant(t *testing.T, result Result, name string) *Symbol {
	t.Helper()
	for _, symbol := range result.Symbols {
		if symbol != nil && symbol.Name == name && symbol.ModuleRoot && symbol.Alias == nil {
			return symbol
		}
	}
	t.Fatalf("missing module constant %s", name)
	return nil
}

func constantAlias(result Result, name string) *Symbol {
	for _, symbol := range result.Symbols {
		if symbol != nil && symbol.Name == name && symbol.Alias != nil {
			return symbol
		}
	}
	return nil
}

func symbolsByName(result Result) map[string]*Symbol {
	found := map[string]*Symbol{}
	for _, symbol := range result.Symbols {
		found[symbol.Name] = symbol
	}
	return found
}

func constantInt(t *testing.T, symbol *Symbol) int64 {
	t.Helper()
	if symbol == nil || symbol.ConstValue == nil || symbol.ConstValue.integer == nil {
		t.Fatalf("missing constant integer: %+v", symbol)
	}
	return symbol.ConstValue.integer.Int64()
}

func requireSemanticText(t *testing.T, result Result, fragment string) {
	t.Helper()
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.Message, fragment) {
			return
		}
	}
	t.Fatalf("expected message %q, got %+v", fragment, result.Diagnostics)
}
