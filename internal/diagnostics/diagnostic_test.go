package diagnostics

import (
	"testing"

	"ahdcode/internal/source"
)

func TestBagReturnsDefensiveCopy(t *testing.T) {
	var bag Bag
	bag.Error("LEX999", "problem", source.Span{}, "hint")
	items := bag.Items()
	items[0].Code = "changed"
	if bag.Items()[0].Code != "LEX999" {
		t.Fatal("Items exposed Bag internals")
	}
	if !bag.HasErrors() || bag.Len() != 1 {
		t.Fatal("Bag error accounting is incorrect")
	}
}
