package semantic

import "testing"

func TestComplexAndNumericStaticContracts(t *testing.T) {
	tests := []struct {
		name, source string
		ok           bool
	}{
		{"Complex inference and widening", `z := 2 + 3I
a: Complex := 2
b: Complex := 3.5
c: Real := z.magnitude()
d: Complex := z ^ 2
isEqual: Bool := z == (2.0 + 3I)`, true},
		{"Complex ordering is absent", `z := 2 + 3I
ordered: Bool := z < z`, false},
		{"Complex power accepts only an Int exponent", `z: Complex := 2 + 3I
r: Real := 2.0
w := z ^ r`, false},
		{"Numeric constructors and operations", `bring Numeric
from Numeric bring Vector
from Numeric bring Matrix
v: Vector := Numeric.vector([1, 2, 3])
m: Matrix := Numeric.matrix([[1.0, 2.0], [3.0, 4.0]])
x: Vector := m.solve(v)
e: List<Complex> := m.eigenvalues()`, true},
		{"Numeric rejects String conversion", `bring Numeric
v := Numeric.vector(["1", "2"])`, false},
		{"Numeric Vector integrates with Plot", `bring Numeric
bring Plot
x := Numeric.linspace(0.0, 1.0, 3)
chart := Plot.line(x, x)
chart = chart.scatter(x, x, "samples")`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := analyzeWithStandardModules(t, test.source)
			if test.ok && result.HasErrors() {
				t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
			}
			if !test.ok && !result.HasErrors() {
				t.Fatal("expected a semantic diagnostic")
			}
		})
	}
}
