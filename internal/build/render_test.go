package build

import (
	"bytes"
	"path/filepath"
	"testing"
)

// TestFirstPartyValuesRenderTheirContents checks str, write, interpolation,
// nesting, nullable values, and Terminal.pretty for Vector, Matrix, DateTime,
// Duration, and Table, compiled and in the evaluator.
func TestFirstPartyValuesRenderTheirContents(t *testing.T) {
	source := `bring Numeric
bring Time
bring Data
bring Terminal
from Numeric bring (Vector)
v := Numeric.vector([3, 4, 12])
write(v)
write(str(v) == "Vector([3.0, 4.0, 12.0])")
Terminal.emit([str(v)])
write(Numeric.matrix([[1.0, 2.0], [3.5, 4.0]]))
write(Numeric.zeros(0))
write(Time.parseISO("2026-09-18T13:30:00.35+03:00"))
write(Time.dateTimeUTC(2026, 1, 2))
write(Time.duration(-1500))
write(Data.fromRows(["id", "name"], [["1", "Ada \"A\""], ["2", ""]]))
write(Data.fromRows(["id"], []))
vectors: List<Vector> := [v, Numeric.vector([1])]
write(vectors)
write({"a": v})
maybe: Vector? := null
write(maybe)
maybe = v
write(maybe)
Terminal.pretty({"v": v, "w": Numeric.vector([0.5])})
write("interpolated {v} and {Time.duration(1)}")
`
	expected := `Vector([3.0, 4.0, 12.0])
true
Vector([3.0, 4.0, 12.0])
Matrix([[1.0, 2.0], [3.5, 4.0]])
Vector([])
DateTime(2026-09-18T13:30:00.350+03:00)
DateTime(2026-01-02T00:00:00.000Z)
Duration(-1500 ms)
Table(["id", "name"], [["1", "Ada \"A\""], ["2", ""]])
Table(["id"], [])
[Vector([3.0, 4.0, 12.0]), Vector([1.0])]
{"a": Vector([3.0, 4.0, 12.0])}
null
Vector([3.0, 4.0, 12.0])
{
    "v": Vector([3.0, 4.0, 12.0]),
    "w": Vector([0.5])
}
interpolated Vector([3.0, 4.0, 12.0]) and Duration(1 ms)
`
	directory := writeSources(t, map[string]string{"main.ahd": source})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stdout != expected {
		t.Fatalf("native (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, expected)
	}
	var output, errorOutput bytes.Buffer
	runTerminalEvaluator(t, source, &output, &errorOutput)
	if output.String() != expected {
		t.Fatalf("evaluator:\n have %q\n want %q", output.String(), expected)
	}
}
