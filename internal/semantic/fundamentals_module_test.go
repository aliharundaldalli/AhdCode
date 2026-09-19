package semantic

import (
	"testing"

	"ahdcode/internal/types"
)

// The v1.7 standard-library additions are ordinary statically typed calls:
// every argument is checked, calls are all positional or all named, and the
// overloads are explicit.

const fundamentalsPreamble = `bring Math
bring Time
bring Numeric
bring Statistics
bring Data
bring Security
from Time bring (DateTime, Duration)
from Numeric bring (Vector, Matrix)
from Data bring (Table)
moment: DateTime := Time.parseISO("2026-09-18T10:30:00Z")
span: Duration := Time.duration(1000)
v: Vector := Numeric.vector([1, 2, 3])
m: Matrix := Numeric.matrix([[1, 2], [3, 4]])
table: Table := Data.fromRows(["id", "name"], [["1", "Ada"]])
ints: List<Int> := [1, 2, 3]
reals: List<Real> := [1.5, 2.5, 3.5]
`

func TestFundamentalsValidUsage(t *testing.T) {
	result := analyzeWithStandardModules(t, fundamentalsPreamble+`angle: Real := Math.asin(0.5) + Math.acos(value: 0.5) + Math.atan(1.0)
turn: Real := Math.atan2(1.0, 2.0) + Math.atan2(y: 1, x: 2)
hyperbolic: Real := Math.sinh(1.0) + Math.cosh(1) + Math.tanh(1.0)
length: Real := Math.hypot(3, 4) + Math.hypot(x: 3.0, y: 4.0)
logs: Real := Math.log2(8.0) + Math.cbrt(27)
converted: Real := Math.radians(180) + Math.degrees(radians: 3.14)
divisor: Int := Math.gcd(12, 18) + Math.lcm(first: 4, second: 6)

text: String := moment.toISO()
later: DateTime := moment.add(span)
earlier: DateTime := moment.subtract(duration: span)
total: Duration := span.add(span).subtract(other: span).negate().abs()

component: Real := v.at(0) + v.at(index: -1) + v.norm() + m.at(0, 1) + m.at(row: 1, column: 0) + m.norm()
product: Matrix := v.outer(v).hadamard(other: v.outer(v))
crossed: Vector := v.cross(v)
line: Vector := m.row(0)
column: Vector := m.column(index: 1)
diagonal: Vector := m.diagonal()
applied: Vector := m.matvec(Numeric.vector([1, 1]))

c1: Real := Statistics.covariance(ints, ints)
c2: Real := Statistics.covariance(ints, reals)
c3: Real := Statistics.sampleCovariance(reals, ints)
c4: Real := Statistics.correlation(first: reals, second: reals)
fit: Pair<String, Real> := Statistics.linearRegression(ints, reals)
fit2: Pair<String, Real> := Statistics.linearRegression(x: reals, y: ints)

joined: Table := table.innerJoin(table.rename("name", "other"), "id")
joined2: Table := table.innerJoin(other: table, leftKey: "id", rightKey: "id")
joined3: Table := table.innerJoin(other: table, key: "id")
stacked: Table := table.concat(table).concat(other: table)

hash: String := Security.bcryptHash("password")
ok: Bool := Security.bcryptVerify(password: "password", encodedHash: hash)
`)
	requireSemanticClean(t, result)
}

// Static mistakes are compiler diagnostics, never runtime errors.
func TestFundamentalsRejectStaticMistakes(t *testing.T) {
	for _, source := range []string{
		// Math: fixed arity and types; no pow or angle type.
		`Math.asin("1")`,
		`Math.asin()`,
		`Math.asin(1.0, 2.0)`,
		`Math.atan2(1.0)`,
		`Math.atan2(x: 1.0, value: 2.0)`,
		`Math.hypot(1.0, 2.0, 3.0)`,
		`Math.gcd(1.5, 2)`,
		`Math.lcm(1, 2.0)`,
		`count: Int := Math.log2(8.0)`,
		`Math.pow(2.0, 3.0)`,
		`Math.radians(degrees: 1.0, normalize: true)`,
		// Time: strict ISO text and Duration arithmetic only.
		`Time.parseISO(20260918)`,
		`Time.parseISO("2026-09-18T10:30:00Z", "UTC")`,
		`Time.parseISO(value: "2026-09-18T10:30:00Z")`,
		`moment.toISO("Z")`,
		`moment.add(1000)`,
		`moment.add(moment)`,
		`moment.add(span, span)`,
		`moment.subtract(duration: span, zone: "UTC")`,
		`moment.add(milliseconds: 1000)`,
		`span.add(1000)`,
		`span.add(moment)`,
		`span.subtract(span, span)`,
		`span.negate(span)`,
		`span.abs(1)`,
		`span.multiply(2)`,
		`moment.plus(span)`,
		`text: String := span.add(span)`,
		`mover := moment.add`,
		// Numeric: fixed shapes; no norm selection, broadcasting, or assignment.
		`v.at(0.5)`,
		`v.at()`,
		`v.at(0, 1)`,
		`v.norm(2)`,
		`v.cross(m)`,
		`v.outer(1.0)`,
		`m.at(0)`,
		`m.at(row: 0, col: 1)`,
		`m.row(0.0)`,
		`m.column("0")`,
		`m.diagonal(0)`,
		`m.norm("fro")`,
		`m.hadamard(v)`,
		`m.hadamard(2.0)`,
		`m.matvec(m)`,
		`m.matvec([1.0, 2.0])`,
		`single: Real := v.cross(v)`,
		// Statistics: two numeric Lists; explicit overloads only.
		`Statistics.covariance(ints)`,
		`Statistics.covariance(ints, ints, ints)`,
		`Statistics.covariance(["a"], ints)`,
		`Statistics.correlation(ints, 1.0)`,
		`Statistics.correlation(first: ints, y: ints)`,
		`Statistics.linearRegression(first: ints, second: ints)`,
		`slope: Real := Statistics.linearRegression(ints, ints)`,
		`Statistics.rSquared(ints, ints)`,
		`Statistics.tTest(ints, ints)`,
		// Data: inner join and concat only.
		`table.concat()`,
		`table.concat(table, table)`,
		`table.concat([table])`,
		`table.concat(other: table, fill: "")`,
		`table.innerJoin(table)`,
		`table.innerJoin(table, 1)`,
		`table.innerJoin("id", table)`,
		`table.innerJoin(table, "id", "id", "id")`,
		`table.innerJoin(other: table, leftKey: "id")`,
		`table.leftJoin(table, "id")`,
		`table.merge(table, "id")`,
		// Security: explicit functions, no cost knob or auto-detection.
		`Security.bcryptHash()`,
		`Security.bcryptHash("pw", 12)`,
		`Security.bcryptHash(password: "pw", cost: 12)`,
		`Security.bcryptVerify("pw")`,
		`Security.bcryptVerify("pw", 1)`,
		`Security.passwordVerifyAny("pw", "hash")`,
		`Security.passwordHashAlgorithm("hash")`,
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, fundamentalsPreamble+source+"\n"))
	}
}

// TestPublishedMemberGeneralizationKeepsGraphicsAndLegacyMembers pins the
// v1.6 behavior the v1.7 published-member generalization must not change:
// Canvas and Turtle publish exactly their original Symbols in their original
// order, identities from other modules publish nothing, and members that
// never published parameter names still reject named arguments.
func TestPublishedMemberGeneralizationKeepsGraphicsAndLegacyMembers(t *testing.T) {
	for _, testCase := range []struct {
		identity *types.ClassSymbol
		names    []string
	}{
		{graphicsCanvasClass, []string{"clear", "line", "circle", "rectangle", "save", "wait", "close", "isOpen", "turtle", "onClick", "onKey"}},
		{graphicsTurtleClass, []string{"forward", "backward", "left", "right", "moveTo", "setHeading", "penUp", "penDown", "setColor", "setWidth", "home", "x", "y", "heading"}},
	} {
		members := BuiltinClassMembers(testCase.identity)
		if len(members) != len(testCase.names) {
			t.Fatalf("%s members = %d, want %d", testCase.identity.Name, len(members), len(testCase.names))
		}
		for index, name := range testCase.names {
			operation := TypeOperation(testCase.identity.Name + "." + name)
			if members[index] == nil || members[index].Name != name || members[index] != graphicsMembers[operation] {
				t.Fatalf("%s member %d = %v, want the v1.6 Symbol %q", testCase.identity.Name, index, members[index], name)
			}
			if members[index].OverloadSet != nil || !TypeOperationBindsArguments(operation) {
				t.Fatalf("%s.%s changed shape", testCase.identity.Name, name)
			}
		}
	}
	for _, identity := range []*types.ClassSymbol{
		{ModuleID: "user:main", Name: "Canvas"}, {ModuleID: "user:main", Name: "Table"}, {ModuleID: "user:main", Name: "Duration"},
		timeCalendarClass, numericErrorClass, nil,
	} {
		if members := BuiltinClassMembers(identity); len(members) != 0 {
			t.Fatalf("identity %v publishes %d members", identity, len(members))
		}
	}
	for _, operation := range []TypeOperation{DateTimeBefore, DateTimeToOffset, DataRowCount, DataRename, NumericVectorDot, NumericMatrixSolve, CalendarWeekday} {
		if TypeOperationBindsArguments(operation) {
			t.Fatalf("%s now binds named arguments", operation)
		}
	}
	for _, source := range []string{
		"moment.before(other: moment)",
		"moment.toOffset(offsetMinutes: 60)",
		"table.rename(old: \"id\", new: \"key\")",
		"v.dot(other: v)",
	} {
		requireSemanticFailure(t, analyzeWithStandardModules(t, fundamentalsPreamble+source+"\n"))
	}
	requireSemanticClean(t, analyzeWithStandardModules(t, fundamentalsPreamble+"flag: Bool := moment.before(moment)\nshifted: DateTime := moment.toOffset(60)\n"))
}
