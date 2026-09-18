package build

import (
	"bytes"
	"path/filepath"
	"testing"
)

// nearHelper compares Reals with a tolerance, so the expected output of these
// programs never depends on the last bit of a floating-point result.
const nearHelper = `near: Function := (actual: Real, expected: Real) -> Bool {
    difference: Local := actual - expected
    if difference < 0.0 {
        difference = -difference
    }
    return difference <= 0.000000001
}
`

// fundamentalsPrograms exercise every v1.7 standard-library addition with
// positional and named arguments and with each error class it raises.
var fundamentalsPrograms = []struct {
	name, source, expected string
}{
	{
		name: "Math",
		source: "bring Math\n" + nearHelper + `write(near(Math.asin(1.0), Math.PI / 2.0))
write(near(Math.acos(value: -1.0), Math.PI))
write(near(Math.atan(1.0), Math.PI / 4.0))
write(near(Math.atan2(y: 1.0, x: -1.0), 3.0 * Math.PI / 4.0))
write(near(Math.atan2(0.0, 0.0), 0.0))
write(near(Math.sinh(0.0), 0.0))
write(near(Math.cosh(0.0), 1.0))
write(near(Math.tanh(1000.0), 1.0))
write(near(Math.hypot(3.0, 4.0), 5.0))
write(near(Math.log2(8.0), 3.0))
write(near(Math.cbrt(-27.0), -3.0))
write(near(Math.radians(degrees: 180.0), Math.PI))
write(near(Math.degrees(radians: Math.PI), 180.0))
write(near(Math.degrees(Math.radians(720.0)), 720.0))
write(Math.gcd(-12, 18))
write(Math.gcd(first: 0, second: 0))
write(Math.lcm(-4, 6))
write(Math.lcm(5, 0))
for value in [1.5, -1.5] {
    attempt {
        write(Math.asin(value))
    } except DomainError as error {
        write(error.message)
    }
}
attempt {
    write(Math.log2(0.0))
} except DomainError as error {
    write(error.message)
}
attempt {
    write(Math.cosh(1000.0))
} except OverflowError as error {
    write(error.message)
}
attempt {
    write(Math.lcm(9223372036854775807, 2))
} except OverflowError as error {
    write(error.message)
}
attempt {
    write(Math.gcd(-9223372036854775807 - 1, 0))
} except OverflowError as error {
    write(error.message)
}
`,
		expected: "true\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n6\n0\n12\n0\n" +
			"Math.asin requires a value between -1 and 1\nMath.asin requires a value between -1 and 1\n" +
			"Math.log2 requires a value greater than zero\nMath.cosh result exceeds finite Real range\n" +
			"Math.lcm result does not fit Int\nMath.gcd result does not fit Int\n",
	},
	{
		name: "Time",
		source: `bring Time
start := Time.parseISO("2026-09-18T13:30:00.35+03:00")
write(start.toISO())
write(start.offsetMinutes)
write(start.millisecond)
write(Time.parseISO(text: "2026-09-18T10:30:00Z").toISO())
write(Time.parseISO(start.toISO()).sameMoment(start))
write(start.add(Time.duration(3600000)).toISO())
write(start.subtract(duration: Time.duration(86400000)).toISO())
write(start.toUTC().add(Time.duration(1)).toISO())
write(Time.between(start, start.add(Time.duration(1500))).milliseconds)
span := Time.duration(1500)
write(span.add(Time.duration(500)).milliseconds)
write(span.subtract(other: Time.duration(2000)).milliseconds)
write(span.negate().milliseconds)
write(span.negate().abs().milliseconds)
for text in ["2026-09-18", "2026-09-18T10:30:00", "2026-09-18T10:30:00.1234Z", "Sep 18 2026", "18/09/2026", "UTC+3", "Europe/Istanbul", "2026-02-30T00:00:00Z"] {
    attempt {
        write(Time.parseISO(text).toISO())
    } except ValueError as error {
        write(error.message)
    }
}
attempt {
    write(Time.parseISO("9999-12-31T23:59:59.999Z").add(Time.duration(1)).toISO())
} except ValueError as error {
    write(error.message)
}
attempt {
    write(Time.duration(-9223372036854775807 - 1).abs().milliseconds)
} except OverflowError as error {
    write(error.message)
}
attempt {
    write(Time.duration(9223372036854775807).add(Time.duration(1)).milliseconds)
} except OverflowError as error {
    write(error.message)
}
`,
		expected: "2026-09-18T13:30:00.350+03:00\n180\n350\n2026-09-18T10:30:00.000Z\ntrue\n" +
			"2026-09-18T14:30:00.350+03:00\n2026-09-17T13:30:00.350+03:00\n2026-09-18T10:30:00.351Z\n1500\n" +
			"2000\n-500\n-1500\n1500\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"2026-09-18\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"2026-09-18T10:30:00\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"2026-09-18T10:30:00.1234Z\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"Sep 18 2026\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"18/09/2026\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"UTC+3\"\n" +
			"Time.parseISO expects YYYY-MM-DDTHH:MM:SS[.fff] followed by Z or ±HH:MM, got \"Europe/Istanbul\"\n" +
			"Time.parseISO day 30 does not exist in 2026-02\n" +
			"DateTime.add result is outside the supported DateTime range\n" +
			"Duration.abs overflowed Int milliseconds\nDuration.add overflowed Int milliseconds\n",
	},
	{
		name: "Numeric",
		source: "bring Numeric\nfrom Numeric bring (NumericError)\n" + nearHelper + `v := Numeric.vector([3, 4, 12])
write(v.at(0))
write(v.at(index: -1))
write(near(v.norm(), 13.0))
write(Numeric.vector([1, 0, 0]).cross(Numeric.vector([0, 1, 0])).values())
write(Numeric.vector([0, 1, 0]).cross(other: Numeric.vector([1, 0, 0])).values())
write(Numeric.vector([1, 2]).outer(Numeric.vector([3, 4, 5])).rows())
m := Numeric.matrix([[1, 2, 3], [4, 5, 6]])
write(m.at(1, 2))
write(m.at(row: -1, column: 0))
write(m.row(0).values())
write(m.column(index: -1).values())
write(m.diagonal().values())
write(near(m.norm(), 9.539392014169456))
write(m.hadamard(m).rows())
write(m.matvec(Numeric.vector([1, 0, -1])).values())
write(m.rows())
attempt {
    write(v.at(3))
} except IndexError as error {
    write(error.message)
}
attempt {
    write(m.at(0, -4))
} except IndexError as error {
    write(error.message)
}
attempt {
    write(Numeric.vector([1, 2]).cross(Numeric.vector([3, 4])).values())
} except NumericError as error {
    write(error.message)
}
attempt {
    write(m.hadamard(Numeric.identity(2)).rows())
} except NumericError as error {
    write(error.message)
}
attempt {
    write(m.matvec(Numeric.vector([1, 1])).values())
} except NumericError as error {
    write(error.message)
}
`,
		expected: "3.0\n12.0\ntrue\n[0.0, 0.0, 1.0]\n[0.0, 0.0, -1.0]\n[[3.0, 4.0, 5.0], [6.0, 8.0, 10.0]]\n" +
			"6.0\n4.0\n[1.0, 2.0, 3.0]\n[3.0, 6.0]\n[1.0, 5.0]\ntrue\n[[1.0, 4.0, 9.0], [16.0, 25.0, 36.0]]\n[-2.0, -2.0]\n" +
			"[[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]]\n" +
			"Vector index 3 is out of range for length 3\nMatrix column index -4 is out of range for length 3\n" +
			"Vector.cross requires two vectors of length 3, got 2 and 2\nMatrix.hadamard requires matrices of the same shape\n" +
			"Matrix.matvec requires a Vector of length 3, got 2\n",
	},
	{
		name: "Statistics",
		source: "bring Statistics\nfrom Statistics bring (StatisticsError)\n" + nearHelper + `x: List<Int> := [1, 2, 3, 4, 5]
y: List<Real> := [2.0, 4.1, 5.9, 8.2, 9.8]
write(near(Statistics.covariance(x, y), 3.94))
write(near(Statistics.sampleCovariance(first: x, second: y), 4.925))
write(near(Statistics.covariance(y, x), 3.94))
write(near(Statistics.covariance(y, y), Statistics.variance(y)))
write(near(Statistics.correlation(x, y), 0.9988296493298859))
write(near(Statistics.correlation(x, [10, 8, 6, 4, 2]), -1.0))
fit := Statistics.linearRegression(x, y)
for key in fit {
    write(key)
}
write(near(fit["slope"], 1.97))
write(near(fit["intercept"], 0.09))
write(Statistics.linearRegression(x: [1, 2, 3], y: [7, 7, 7]))
write(x)
write(y)
empty: List<Int> := []
attempt {
    write(Statistics.covariance(empty, empty))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.correlation([1, 2, 3], [1, 2]))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.sampleCovariance([1], [2]))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.correlation([1, 2, 3], [4, 4, 4]))
} except StatisticsError as error {
    write(error.message)
}
attempt {
    write(Statistics.linearRegression([2.0, 2.0], [1.0, 3.0]))
} except StatisticsError as error {
    write(error.message)
}
`,
		expected: "true\ntrue\ntrue\ntrue\ntrue\ntrue\nslope\nintercept\ntrue\ntrue\n{\"slope\": 0.0, \"intercept\": 7.0}\n" +
			"[1, 2, 3, 4, 5]\n[2.0, 4.1, 5.9, 8.2, 9.8]\n" +
			"covariance is undefined for empty Lists\ncorrelation requires Lists of the same length, got 3 and 2\n" +
			"sampleCovariance requires at least two pairs of values\ncorrelation is undefined when either List has zero variance\n" +
			"linearRegression is undefined when x has zero variance\n",
	},
	{
		name: "Data",
		source: `bring Data
from Data bring (DataError)
people := Data.fromRows(["id", "name"], [["1", "Ada"], ["2", "Alan"], ["", "Blank"], ["3", "Grace"]])
orders := Data.fromRows(["id", "item"], [["2", "book"], ["1", "pen"], ["2", "lamp"], ["", "ghost"], ["9", "none"]])
joined := people.innerJoin(orders, "id")
write(joined.columns())
write(joined.rows())
cities := Data.fromRows(["person", "city"], [["1", "London"], ["1", "Paris"], ["2", "Rome"]])
write(people.innerJoin(other: cities, leftKey: "id", rightKey: "person").rows())
write(people.innerJoin(Data.fromRows(["id", "x"], []), "id").columns())
write(people.concat(Data.fromRows(["id", "name"], [["4", "Linus"]])).rows())
write(Data.fromRows(["id", "name"], []).concat(other: Data.fromRows(["id", "name"], [])).rowCount())
write(people.rowCount())
attempt {
    write(people.concat(orders).rowCount())
} except DataError as error {
    write(error.message)
}
attempt {
    write(people.innerJoin(people, "id").rowCount())
} except DataError as error {
    write(error.message)
}
attempt {
    write(people.innerJoin(orders, "missing").rowCount())
} except DataError as error {
    write(error.message)
}
`,
		expected: "[\"id\", \"name\", \"item\"]\n" +
			"[{\"id\": \"1\", \"name\": \"Ada\", \"item\": \"pen\"}, {\"id\": \"2\", \"name\": \"Alan\", \"item\": \"book\"}, " +
			"{\"id\": \"2\", \"name\": \"Alan\", \"item\": \"lamp\"}, {\"id\": \"\", \"name\": \"Blank\", \"item\": \"ghost\"}]\n" +
			"[{\"id\": \"1\", \"name\": \"Ada\", \"city\": \"London\"}, {\"id\": \"1\", \"name\": \"Ada\", \"city\": \"Paris\"}, " +
			"{\"id\": \"2\", \"name\": \"Alan\", \"city\": \"Rome\"}]\n" +
			"[\"id\", \"name\", \"x\"]\n" +
			"[{\"id\": \"1\", \"name\": \"Ada\"}, {\"id\": \"2\", \"name\": \"Alan\"}, {\"id\": \"\", \"name\": \"Blank\"}, " +
			"{\"id\": \"3\", \"name\": \"Grace\"}, {\"id\": \"4\", \"name\": \"Linus\"}]\n0\n4\n" +
			"Table.concat requires the same columns in the same order\n" +
			"Table.innerJoin column \"name\" exists in both Tables\n" +
			"Table.innerJoin left key column \"missing\" does not exist\n",
	},
	{
		name: "Security bcrypt",
		source: `bring Security
from Security bring (SecurityError)
hash := Security.bcryptHash("correct horse")
write(len(hash))
write(hash.startsWith("$2a$12$"))
write(Security.bcryptVerify("correct horse", hash))
write(Security.bcryptVerify(password: "wrong", encodedHash: hash))
write(Security.bcryptVerify("rasmuslerdorf", "$2y$10$.vGA1O9wmRjrwAVXD98HNOgsNpDczlqm3Jq7KnEd1rVAGv3Fykk1a"))
write(Security.bcryptVerify("", Security.bcryptHash("")))
argon := Security.passwordHash("correct horse")
write(Security.passwordVerify("correct horse", argon))
for encoded in ["not a hash", argon, "$2y$31$.vGA1O9wmRjrwAVXD98HNOgsNpDczlqm3Jq7KnEd1rVAGv3Fykk1a"] {
    attempt {
        write(Security.bcryptVerify("correct horse", encoded))
    } except SecurityError as error {
        write(error.message)
    }
}
attempt {
    write(Security.passwordVerify("correct horse", hash))
} except SecurityError as error {
    write(error.message)
}
attempt {
    write(Security.bcryptHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
} except SecurityError as error {
    write(error.message)
}
`,
		expected: "60\ntrue\ntrue\nfalse\ntrue\ntrue\ntrue\n" +
			"bcrypt hash is malformed or uses an unsupported format\nbcrypt hash is malformed or uses an unsupported format\n" +
			"bcrypt hash cost is outside the supported range 04..16\nSecurity password hash is malformed\n" +
			"bcrypt accepts a password of at most 72 UTF-8 bytes\n",
	},
}

// TestFundamentalsNativeAndEvaluatorAgree runs each v1.7 program compiled and
// in the evaluator; both must print exactly the expected output.
func TestFundamentalsNativeAndEvaluatorAgree(t *testing.T) {
	for _, testCase := range fundamentalsPrograms {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			directory := writeSources(t, map[string]string{"main.ahd": testCase.source})
			stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
			if code != 0 || stdout != testCase.expected {
				t.Fatalf("native output (exit %d, stderr %q):\n have %q\n want %q", code, stderr, stdout, testCase.expected)
			}
			var output, errorOutput bytes.Buffer
			runTerminalEvaluator(t, testCase.source, &output, &errorOutput)
			if output.String() != testCase.expected {
				t.Fatalf("evaluator output differs:\n have %q\n want %q", output.String(), testCase.expected)
			}
		})
	}
}
