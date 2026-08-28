package build

import (
	"path/filepath"
	"testing"
)

func TestStructureParameterModifiersRunAsNativeCode(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        name: String
        temporary: Local Int
    )
}

example: Example := Example(id: 7, name: "first", temporary: 99)
write(example.id)
write(example.name)
example.name = "updated"
write(example.name)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" || stdout != "7\nfirst\nupdated\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestSpecConstantAndLocalStructureExampleRuns(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `Person: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
        password: Local String
    ) {
        attribute.passwordHash: Confidential String := password
    }
}

student: Student := Student(
    name: "Ada"
    age: 20
    number: 42
    password: "secret"
)
write(student.name)
write(student.number)
student.name = "Grace"
write(student.name)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	if code != 0 || stderr != "" || stdout != "Ada\n42\nGrace\n" {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestConstantStructureReferenceDeepFreezesAliases(t *testing.T) {
	directory := writeSources(t, map[string]string{"main.ahd": `JosephusNode: Class<> := {
    structure: Attributes := (
        name: String
    )
}

Holder: Class<> := {
    structure: Attributes := (
        index: Constant Int
        nodes: Constant List<JosephusNode>
    )
}

node: JosephusNode := JosephusNode(name: "one")
nodes: List<JosephusNode> := [node]
holder: Holder := Holder(index: 1, nodes: nodes)

attempt {
    clear(nodes)
}
except ConstantError {
    write("list frozen")
}

attempt {
    node.name = "changed"
}
except ConstantError {
    write("node frozen")
}

write(holder.index)
write(len(holder.nodes))
write(node.name)
`})
	stdout, stderr, code := buildAndRun(t, filepath.Join(directory, "main.ahd"), "")
	want := "list frozen\nnode frozen\n1\n1\none\n"
	if code != 0 || stderr != "" || stdout != want {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestConstantAndLocalStructureAccessDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		use  string
		code string
	}{
		{"Constant mutation", "example.id = 2", "SEM009"},
		{"Local is not an attribute", "write(example.temporary)", "SEM019"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := writeSources(t, map[string]string{"main.ahd": `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        temporary: Local Int
    )
}
example: Example := Example(id: 1, temporary: 9)
` + test.use + "\n"})
			result := Compile(filepath.Join(directory, "main.ahd"))
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
				t.Fatalf("diagnostics = %+v, want one %s", result.Diagnostics, test.code)
			}
		})
	}
}
