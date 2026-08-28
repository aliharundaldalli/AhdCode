package semantic

import "testing"

func classNamed(result Result, name string) *Symbol {
	for _, symbol := range result.Symbols {
		if symbol != nil && symbol.Kind == ClassSymbol && symbol.Name == name {
			return symbol
		}
	}
	return nil
}

func TestStructureParameterModifiersDescribeAttributesNotLexicalScope(t *testing.T) {
	_, result := analyzeText(t, `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        name: String
        temporary: Local Int
        secret: Confidential Int
    )
}
`)
	requireSemanticClean(t, result)
	example := classNamed(result, "Example")
	if example == nil {
		t.Fatal("Example Class metadata is missing")
	}
	if id := example.Members["id"]; id == nil || !id.Constant {
		t.Fatalf("id attribute = %#v, want Constant", id)
	}
	if name := example.Members["name"]; name == nil || name.Constant {
		t.Fatalf("name attribute = %#v, want mutable", name)
	}
	if _, exists := example.Members["temporary"]; exists {
		t.Fatal("Local structure parameter became an instance attribute")
	}
	if secret := example.Members["secret"]; secret == nil || !secret.Confidential {
		t.Fatalf("secret attribute = %#v, want Confidential", secret)
	}
}

func TestConstantStructureAttributeRejectsMutation(t *testing.T) {
	_, result := analyzeText(t, `Example: Class<> := {
    structure: Attributes := (
        id: Constant Int
        name: String
        temporary: Local Int
    )
}

example: Example := Example(id: 1, name: "one", temporary: 9)
example.name = "updated"
example.id = 2
`)
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != codeConstantAssignment {
		t.Fatalf("diagnostics = %+v, want one %s", result.Diagnostics, codeConstantAssignment)
	}
}

func TestInheritedStructureParametersPreserveAttributeModifiers(t *testing.T) {
	_, result := analyzeText(t, `Parent: Class<> := {
    structure: Attributes := (
        id: Constant Int
        name: String
        temporary: Local Int
    )
}

Child: Class<Parent> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
        password: Local String
    )
}
`)
	requireSemanticClean(t, result)
	child := classNamed(result, "Child")
	if child == nil || len(child.ConstructorAttributes) != 5 {
		t.Fatalf("Child constructor attributes = %#v", child)
	}
	if child.ConstructorAttributes[0] == nil || !child.ConstructorAttributes[0].Constant {
		t.Fatal("inherited Constant attribute metadata was lost")
	}
	if child.ConstructorAttributes[1] == nil || child.ConstructorAttributes[1].Constant {
		t.Fatal("inherited mutable attribute metadata was changed")
	}
	if child.ConstructorAttributes[2] != nil || child.ConstructorAttributes[4] != nil {
		t.Fatal("inherited or declared Local parameter became an attribute")
	}
	if child.ConstructorAttributes[3] == nil || !child.ConstructorAttributes[3].Constant {
		t.Fatal("declared Constant child attribute metadata was lost")
	}
}

func TestConstantRecursiveListStructureStressAndNullSafety(t *testing.T) {
	_, clean := analyzeText(t, `JosephusNode: Class<> := {
    structure: Attributes := (
        index: Constant Int
        nodes: Constant List<JosephusNode>
    )
}
`)
	requireSemanticClean(t, clean)

	_, nullable := analyzeText(t, `JosephusNode: Class<> := {
    structure: Attributes := (
        name: String
    )
}
Holder: Class<> := {
    structure: Attributes := (
        nodes: Constant List<JosephusNode>
    )
}
node: JosephusNode := JosephusNode(name: "one")
holder: Holder := Holder(nodes: [node])
write(holder.nodes[0].name)
`)
	requireSemanticCode(t, nullable, codeNullableUse)
}
