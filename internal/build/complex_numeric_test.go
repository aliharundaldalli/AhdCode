package build

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestComplexAndNumericRunNatively(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "ahdnumeric")
	if runtime.GOOS == "windows" {
		helper += ".exe"
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", helper, "./cmd/ahdnumeric")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("could not build ahdnumeric: %v\n%s", err, output)
	}
	t.Setenv("AHDCODE_NUMERIC_RUNTIME", helper)

	runProgramCases(t, []program{
		{
			name: "Complex scalar semantics",
			sources: map[string]string{"main.ahd": `z := 2 + 3I
w: Complex := 2
write(z)
write(type(z))
write(z.conjugate())
write(z.real())
write(z.imag())
write(z.magnitude())
write(z == (2.0 + 3I))
write((1 + 1I) ^ 2)
write(w)
`},
			expected: "2.0+3.0I\nComplex\n2.0-3.0I\n2.0\n3.0\n3.6055512754639896\ntrue\n0.0+2.0I\n2.0+0.0I\n",
		},
		{
			name: "Numeric vector and matrix core",
			sources: map[string]string{"main.ahd": `bring Numeric
v := Numeric.vector([1, 2, 3])
w := Numeric.ones(3)
write(v.values())
write(v.add(w).values())
write(v.dot(w))
m := Numeric.matrix([[1, 2], [3, 4]])
write(m.rows())
write(m.transpose().rows())
write(m.determinant())
write(m.inverse().rows())
write(m.matmul(Numeric.identity(2)).rows())
write(Numeric.matrix([[0, -1], [1, 0]]).eigenvalues())
`},
			expected: "[1.0, 2.0, 3.0]\n[2.0, 3.0, 4.0]\n6.0\n[[1.0, 2.0], [3.0, 4.0]]\n[[1.0, 3.0], [2.0, 4.0]]\n-2.0\n[[-1.9999999999999996, 0.9999999999999998], [1.4999999999999998, -0.4999999999999999]]\n[[1.0, 2.0], [3.0, 4.0]]\n[0.0+1.0I, 0.0-1.0I]\n",
		},
	})
}
