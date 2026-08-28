package formatter

import (
	"testing"

	"ahdcode/internal/source"
)

func FuzzFormattingIsPanicFreeAndIdempotent(f *testing.F) {
	for _, seed := range []string{
		"x:Int:=5\n", "write(\"hello {name}\")\n", "// comment\nvalues:List<Int>:=[1,2]\n",
		"f:Function:=(x:Int)->Int{return x}\n", "text:String:=\"\"\"a\nb\"\"\"\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		first := Format(source.NewFile(1, "fuzz.ahd", text))
		if first.HasErrors() || first.Text == "" {
			return
		}
		second := Format(source.NewFile(1, "fuzz.ahd", first.Text))
		if second.HasErrors() {
			t.Fatalf("formatted output became invalid: %+v\n%s", second.Diagnostics, first.Text)
		}
		if second.Text != first.Text {
			t.Fatalf("formatter is not idempotent:\n%s\n---\n%s", first.Text, second.Text)
		}
	})
}
