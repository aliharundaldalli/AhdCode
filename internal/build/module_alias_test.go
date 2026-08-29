package build

import "testing"

func TestModuleAliasesRunThroughLoweringAndBackend(t *testing.T) {
	runProgramCases(t, []program{
		{
			name: "user module alias",
			sources: map[string]string{
				"Engine.ahd": "tick: Function := () -> Int { return 7 }\n",
				"main.ahd":   "bring Engine as E\nwrite(E.tick())\n",
			},
			expected: "7\n",
		},
		{
			name:     "Math and Time aliases",
			sources:  map[string]string{"main.ahd": "bring Math as M\nbring Time as T\nwrite(M.sqrt(25))\nwrite(T.Calendar.isLeapYear(2028))\n"},
			expected: "5.0\ntrue\n",
		},
		{
			name: "Latex alias",
			sources: map[string]string{"main.ahd": `bring Latex as L
write(L.escape("A&B"))
`},
			expected: "A\\&B\n",
		},
	})
}
