package build

import "testing"

// TestCollectionMutationProgramsRunAsNativeExecutables covers the v0.1 List and
// Pair mutation surface through the whole compile chain.
func TestCollectionMutationProgramsRunAsNativeExecutables(t *testing.T) {
	cases := []program{
		{
			name: "List add appends to the end",
			sources: map[string]string{"main.ahd": `values: List<Int> := [
    10
    20
]

values.add(30)
write(str(values))
write(len(values))
`},
			expected: "[10, 20, 30]\n3\n",
		},
		{
			name: "List eject removes by index",
			sources: map[string]string{"main.ahd": `values: List<Int> := [
    10
    20
    30
]

values.eject(1)
write(str(values))
values.eject(-1)
write(str(values))
`},
			expected: "[10, 30]\n[10]\n",
		},
		{
			name: "an empty List accepts add and eject",
			sources: map[string]string{"main.ahd": `values: List<Int> := []
values.add(10)
values.add(20)
write(str(values))
values.eject(0)
write(str(values))
write(len(values))
`},
			expected: "[10, 20]\n[20]\n1\n",
		},
		{
			name: "Pair insert appends and update keeps position",
			sources: map[string]string{"main.ahd": `scores: Pair<String, Int> := {}
scores["Ali"] = 85
scores["Ayşe"] = 92
write(str(scores))
scores["Ali"] = 100
write(str(scores))
`},
			expected: "{\"Ali\": 85, \"Ayşe\": 92}\n{\"Ali\": 100, \"Ayşe\": 92}\n",
		},
		{
			name: "Pair eject and re-add moves the key to the end",
			sources: map[string]string{"main.ahd": `scores: Pair<String, Int> := {}
scores["Ali"] = 85
scores["Ayşe"] = 92
scores.eject("Ali")
write(str(scores))
scores["Ali"] = 100
write(str(scores))
for key in scores {
    write(key)
}
`},
			expected: "{\"Ayşe\": 92}\n{\"Ayşe\": 92, \"Ali\": 100}\nAyşe\nAli\n",
		},
		{
			name: "mutation through an alias is shared",
			sources: map[string]string{"main.ahd": `first: List<Int> := [1, 2]
second: List<Int> := first
second.add(3)
write(str(first))
first.eject(0)
write(str(second))

left: Pair<String, Int> := {
    "x": 1
}

right: Pair<String, Int> := left
right["y"] = 2
write(str(left))
right.eject("x")
write(str(left))
`},
			expected: "[1, 2, 3]\n[2, 3]\n{\"x\": 1, \"y\": 2}\n{\"y\": 2}\n",
		},
		{
			name: "an out-of-range eject raises a catchable IndexError",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1]
attempt {
    values.eject(5)
}
except IndexError as error {
    write("index rejected")
}

write(len(values))
`},
			expected: "index rejected\n1\n",
		},
		{
			name: "a missing key eject raises a catchable KeyError",
			sources: map[string]string{"main.ahd": `scores: Pair<String, Int> := {
    "a": 1
}

attempt {
    scores.eject("missing")
}
except KeyError as error {
    write("key rejected")
}

write(len(scores))
`},
			expected: "key rejected\n1\n",
		},
		{
			name: "a frozen graph rejects every mutation through an alias",
			sources: map[string]string{"main.ahd": `values: List<Int> := [1, 2]
alias: List<Int> := values
frozen: Constant List<Int> := values

attempt {
    alias.add(3)
}
except ConstantError as error {
    write("add rejected")
}

attempt {
    alias.eject(0)
}
except ConstantError as error {
    write("eject rejected")
}

scores: Pair<String, Int> := {
    "a": 1
}

shared: Pair<String, Int> := scores
frozenPair: Constant Pair<String, Int> := scores

attempt {
    shared["b"] = 2
}
except ConstantError as error {
    write("insert rejected")
}

attempt {
    shared.eject("a")
}
except ConstantError as error {
    write("pair eject rejected")
}

write(len(values))
write(len(scores))
`},
			expected: "add rejected\neject rejected\ninsert rejected\npair eject rejected\n2\n1\n",
		},
		{
			name: "mutation evaluates its receiver once",
			sources: map[string]string{"main.ahd": `calls: Int := 0

pick: Function := (
    values: List<Int>
) -> List<Int> {
    calls: Global Int
    calls = calls + 1
    return values
}

values: List<Int> := [1, 2]
pick(values).add(3)
pick(values).eject(0)
write(str(values))
write(calls)
`},
			expected: "[2, 3]\n2\n",
		},
		{
			name: "a Class method may still be named add or eject",
			sources: map[string]string{"main.ahd": `Counter: Class<> := {
    structure: Attributes := (
        value: Int
    )

    add: Function := (
        amount: Int
    ) -> Nothing {
        attribute.value = attribute.value + amount
    }

    eject: Function := (
    ) -> Int {
        return attribute.value
    }
}

counter: Counter := Counter(value: 1)
counter.add(2)
write(counter.eject())
`},
			expected: "3\n",
		},
	}
	runAcceptance(t, cases)
}
