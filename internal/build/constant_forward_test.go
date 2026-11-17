package build

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// Gemini's Constant examples, as one whole program through the evaluator and
// through a native executable. The REPL is a different session: it compiles
// the text submitted so far and executes only the new spans, so a name that
// has not been submitted yet is unknown. These cases are full programs.
// The write sits before the mutable-dependent constant. Moving that
// constant above the call could change what the mutable holds, so the
// program is rejected instead of observing a zero.
func TestForwardMutableConstantIsRejectedBeforeItRuns(t *testing.T) {
	directory := writeSources(t, map[string]string{
		"main.ahd": "v: Int := 1\nb: Constant := [a]\nwrite(b[0][0])\na: Constant := [v]\n",
	})
	result := Compile(filepath.Join(directory, "main.ahd"))
	if !result.HasErrors() {
		t.Fatal("expected SEM010")
	}
	text := diagnosticText(result.Diagnostics)
	if !strings.Contains(text, "SEM010") || !strings.Contains(text, "used before it can be initialized") {
		t.Fatalf("diagnostics:\n%s", text)
	}
	if strings.Contains(text, "LWR002") || strings.Contains(text, "IR002") {
		t.Fatalf("internal diagnostic leaked:\n%s", text)
	}
}

func TestForwardConstantsMatchInTheEvaluatorAndNatively(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "forward sum",
			source: "y: Constant := x + 1\nx: Constant := 5\nwrite(y)\n",
			want:   "6\n",
		},
		{
			name:   "use before the later declaration",
			source: "y: Constant := x + 1\nwrite(y)\nx: Constant := 5\n",
			want:   "6\n",
		},
		{
			name:   "chain",
			source: "z: Constant := y + 1\ny: Constant := x + 1\nx: Constant := 5\nwrite(z)\n",
			want:   "7\n",
		},
		{
			name:   "typed forward",
			source: "a: Constant Int := b + 1\nb: Constant := 5\nwrite(a)\n",
			want:   "6\n",
		},
		{
			name:   "uses of a later constant",
			source: "area: Function := () -> Int uses [@radius] {\n    return radius * radius\n}\nwrite(area())\nradius: Constant := 20\n",
			want:   "400\n",
		},
		{
			name:   "Global of a later constant",
			source: "scale: Function := () -> Int {\n    radius: Global\n    return radius + 1\n}\nwrite(scale())\nradius: Constant := 20\n",
			want:   "21\n",
		},
		{
			name:   "local constant from a later global",
			source: "scale: Function := () -> Int uses [@base] {\n    edge: Local Constant := base + 1\n    return edge\n}\nwrite(scale())\nbase: Constant := 5\n",
			want:   "6\n",
		},
		{
			name:   "forward list",
			source: "b: Constant := a\na: Constant := [1, 2]\nwrite(b[0])\n",
			want:   "1\n",
		},
		{
			name:   "list of a later constant",
			source: "values: Constant := [x]\nx: Constant := 5\nwrite(values[0])\n",
			want:   "5\n",
		},
		{
			name:   "mutable list after its binding",
			source: "v: Int := 1\nb: Constant := [a]\na: Constant := [v]\nwrite(b[0][0])\n",
			want:   "1\n",
		},
		{
			name:   "side-effecting call stays where it is written",
			source: "write(\"before\")\nitems: Constant := answer()\nwrite(\"after\")\nwrite(items[0])\nanswer: Function := () -> List<Int> {\n    write(\"call\")\n    return [7]\n}\n",
			want:   "before\ncall\nafter\n7\n",
		},
		{
			name:   "pure call stays where it is written",
			source: "write(\"before\")\nitems: Constant := answer()\nwrite(\"after\")\nwrite(items[0])\nanswer: Function := () -> List<Int> {\n    return [7]\n}\n",
			want:   "before\nafter\n7\n",
		},
		{
			name:   "function value binding stays where it is written",
			source: "write(\"before\")\nhandler: Function := answer\nitems: Constant := handler()\nwrite(\"after\")\nwrite(items[0])\nanswer: Function := () -> List<Int> {\n    write(\"call\")\n    return [4]\n}\n",
			want:   "before\ncall\nafter\n4\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var output, errorOutput bytes.Buffer
			runEvaluatorIn(t, t.TempDir(), tc.source, &output, &errorOutput)
			if output.String() != tc.want {
				t.Fatalf("evaluator %q, stderr %q", output.String(), errorOutput.String())
			}
			sources := writeSources(t, map[string]string{"main.ahd": tc.source})
			stdout, stderr, code := buildAndRunIn(t, filepath.Join(sources, "main.ahd"), t.TempDir())
			if code != 0 {
				t.Fatalf("native exit %d: %s", code, stderr)
			}
			if stdout != tc.want {
				t.Fatalf("native %q, evaluator %q", stdout, tc.want)
			}
		})
	}
}
