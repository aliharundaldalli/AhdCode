package analysis

import (
	"path/filepath"
	"strings"
	"testing"
)

// The v1.7 standard-library additions reach editors through the same
// semantic metadata the compiler checks calls against.

const fundamentalsSource = `bring Math
bring Time
bring Numeric
bring Statistics
bring Data
bring Security
moment := Time.parseISO("2026-09-18T10:30:00Z")
span := Time.duration(1000)
later := moment.add(span)
text := moment.toISO()
longer := span.add(other: span)
v := Numeric.vector([1, 2, 3])
m := Numeric.matrix([[1, 2], [3, 4]])
first := v.at(0)
entry := m.at(row: 0, column: 1)
applied := m.matvec(v)
table := Data.fromRows(["id"], [["1"]])
joined := table.innerJoin(table, "id")
joinedByName := table.innerJoin(other: table, leftKey: "id", rightKey: "id")
stacked := table.concat(table)
angle := Math.atan2(1.0, 2.0)
divisor := Math.gcd(12, 18)
fit := Statistics.linearRegression([1, 2], [2.0, 4.0])
spread := Statistics.covariance([1, 2], [3, 4])
hash := Security.bcryptHash("pw")
matched := Security.bcryptVerify("pw", hash)
`

func TestCompletionOffersFundamentalsMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	for _, testCase := range []struct {
		prefix string
		want   []string
	}{
		{"bring Math\nMath.", []string{"asin", "acos", "atan", "atan2", "sinh", "cosh", "tanh", "hypot", "log2", "cbrt", "radians", "degrees", "gcd", "lcm"}},
		{"bring Time\nTime.", []string{"parseISO"}},
		{"bring Statistics\nStatistics.", []string{"covariance", "sampleCovariance", "correlation", "linearRegression"}},
		{"bring Security\nSecurity.", []string{"bcryptHash", "bcryptVerify", "passwordHash", "passwordVerify"}},
		{"bring Time\nmoment := Time.parseISO(\"2026-09-18T10:30:00Z\")\nmoment.", []string{"toISO", "add", "subtract"}},
		{"bring Time\nspan := Time.duration(1)\nspan.", []string{"add", "subtract", "negate", "abs"}},
		{"bring Numeric\nv := Numeric.vector([1])\nv.", []string{"at", "norm", "outer", "cross"}},
		{"bring Numeric\nm := Numeric.matrix([[1]])\nm.", []string{"at", "row", "column", "diagonal", "norm", "hadamard", "matvec"}},
		{"bring Data\ntable := Data.fromRows([\"id\"], [])\ntable.", []string{"concat", "innerJoin"}},
	} {
		text := testCase.prefix + "\n"
		store.Open(path, text)
		items := store.Completion(path, len(testCase.prefix))
		for _, name := range testCase.want {
			if !hasLabel(items, name) {
				t.Fatalf("after %q expected %q, got %#v", testCase.prefix, name, items)
			}
		}
	}
	for _, missing := range []string{"pow", "passwordVerifyAny", "leftJoin", "rSquared"} {
		for _, prefix := range []string{"bring Math\nMath.", "bring Security\nSecurity.", "bring Statistics\nStatistics."} {
			store.Open(path, prefix+"\n")
			if hasLabel(store.Completion(path, len(prefix)), missing) {
				t.Fatalf("completion after %q offers %q", prefix, missing)
			}
		}
	}
}

func TestHoverShowsFundamentalsSignatures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, fundamentalsSource)
	for _, testCase := range []struct{ target, want string }{
		{"Time.parseISO", "parseISO: (text: String) -> DateTime"},
		{"moment.add", "add: (duration: Duration) -> DateTime"},
		{"moment.toISO", "toISO: () -> String"},
		{"span.add", "add: (other: Duration) -> Duration"},
		{"v.at", "at: (index: Int) -> Real"},
		{"m.at", "at: (row: Int, column: Int) -> Real"},
		{"m.matvec", "matvec: (vector: Vector) -> Vector"},
		{"table.concat", "concat: (other: Table) -> Table"},
		{"Math.atan2", "atan2: (y: Real, x: Real) -> Real"},
		{"Math.gcd", "gcd: (first: Int, second: Int) -> Int"},
		{"Security.bcryptVerify", "bcryptVerify: (password: String, encodedHash: String) -> Bool"},
	} {
		dot := strings.Index(testCase.target, ".")
		hover, ok := store.Hover(path, offsetOf(t, fundamentalsSource, testCase.target)+dot+2)
		if !ok || hover.Text != testCase.want {
			t.Fatalf("hover on %s = %q (%v), want %q", testCase.target, hover.Text, ok, testCase.want)
		}
	}
	// The overloaded members list every signature.
	for _, target := range []string{"table.innerJoin", "Statistics.linearRegression"} {
		dot := strings.Index(target, ".")
		hover, ok := store.Hover(path, offsetOf(t, fundamentalsSource, target)+dot+2)
		if !ok || hover.Text == "" {
			t.Fatalf("no hover on %s", target)
		}
	}
}

func TestSignatureHelpForFundamentals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.ahd")
	store := NewStore()
	store.Open(path, fundamentalsSource)
	for _, testCase := range []struct{ call, want string }{
		{"moment.add(", "(duration: Duration) -> DateTime"},
		{"span.add(", "(other: Duration) -> Duration"},
		{"m.at(", "(row: Int, column: Int) -> Real"},
		{"table.innerJoin(table,", "(other: Table, key: String) -> Table"},
		{"table.innerJoin(other:", "(other: Table, leftKey: String, rightKey: String) -> Table"},
		{"table.concat(", "(other: Table) -> Table"},
		{"Math.atan2(", "(y: Real, x: Real) -> Real"},
		{"Statistics.linearRegression(", "(x: List<Int>, y: List<Real>) -> Pair<String, Real>"},
		{"Statistics.covariance(", "(first: List<Int>, second: List<Int>) -> Real"},
		{"Security.bcryptVerify(", "(password: String, encodedHash: String) -> Bool"},
		{"Time.parseISO(", "(text: String) -> DateTime"},
	} {
		help, ok := store.SignatureHelp(path, offsetOf(t, fundamentalsSource, testCase.call)+len(testCase.call))
		if !ok || help.Label != testCase.want {
			t.Fatalf("signature help for %s = %#v (%v), want %q", testCase.call, help, ok, testCase.want)
		}
	}
}
