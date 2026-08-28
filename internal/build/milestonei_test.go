package build

import "testing"

func TestMilestoneIClarificationsRunAsNativeExecutable(t *testing.T) {
	cases := []program{
		{
			name: "truncated dividend-signed remainder",
			sources: map[string]string{"main.ahd": `write(-5 % 2)
write(5 % -2)
write(-5 % -2)

attempt {
    write(5 % 0)
}
except DivisionByZeroError as error {
    write("division by zero")
}
`},
			expected: "-1\n1\n-1\ndivision by zero\n",
		},
		{
			name: "List equality is deep and same is identity",
			sources: map[string]string{"main.ahd": `a: List<Int> := [1, 2]
b: List<Int> := [1, 2]
c: List<Int> := a

write(str(a == b))
write(str(a same b))
write(str(a same c))
`},
			expected: "true\nfalse\ntrue\n",
		},
		{
			name: "Pair equality is deep and same is identity",
			sources: map[string]string{"main.ahd": `a: Pair<String, Int> := {
    "x": 1
    "y": 2
}
b: Pair<String, Int> := {
    "x": 1
    "y": 2
}
c: Pair<String, Int> := a

write(str(a == b))
write(str(a same b))
write(str(a same c))
`},
			expected: "true\nfalse\ntrue\n",
		},
	}
	runAcceptance(t, cases)
}
