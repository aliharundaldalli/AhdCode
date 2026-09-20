package main

import "testing"

// Surface category labels (v2.2). They are presentation only: the geometry
// is untouched, and an axis without them shows its numbers exactly as it
// did before v2.2.

func TestAxisTicksWithoutCategoriesShowsTheNumbers(t *testing.T) {
	ticks := axisTicks([]float64{1, 2, 3, 4, 5}, nil)
	if len(ticks) != 2 {
		t.Fatalf("a numeric axis has %d labels, want 2", len(ticks))
	}
	if ticks[0].text != "1" || ticks[1].text != "5" {
		t.Fatalf("numeric labels = %q and %q", ticks[0].text, ticks[1].text)
	}
	if ticks[0].at != -1 || ticks[1].at != 1 {
		t.Fatalf("numeric labels sit at %v and %v, want the two ends", ticks[0].at, ticks[1].at)
	}
}

func TestAxisTicksNamesEveryCategoryWhenThereAreFew(t *testing.T) {
	categories := []string{"Analysis", "Algebra", "Geometry", "Statistics", "Programming"}
	ticks := axisTicks([]float64{1, 2, 3, 4, 5}, categories)
	if len(ticks) != len(categories) {
		t.Fatalf("%d labels for %d categories", len(ticks), len(categories))
	}
	for index, tick := range ticks {
		if tick.text != categories[index] {
			t.Fatalf("label %d = %q, want %q", index, tick.text, categories[index])
		}
	}
	// In order, left to right, and inside the floor's edge so the two axes
	// do not collide at the corner they share.
	for index := 1; index < len(ticks); index++ {
		if ticks[index].at <= ticks[index-1].at {
			t.Fatalf("labels are not in order: %v", ticks)
		}
	}
	if ticks[0].at <= -1 || ticks[len(ticks)-1].at >= 1 {
		t.Fatalf("the end labels reach the corner: %v and %v", ticks[0].at, ticks[len(ticks)-1].at)
	}
}

// A long list of categories would overlap, so only the two ends are named
// -- which is still what a numeric axis shows, but in words.
func TestAxisTicksNamesOnlyTheEndsWhenThereAreMany(t *testing.T) {
	coordinates := make([]float64, maxDrawnCategories+1)
	categories := make([]string, maxDrawnCategories+1)
	for index := range coordinates {
		coordinates[index] = float64(index)
		categories[index] = string(rune('A' + index))
	}
	ticks := axisTicks(coordinates, categories)
	if len(ticks) != 2 {
		t.Fatalf("%d labels for %d categories, want the two ends", len(ticks), len(categories))
	}
	if ticks[0].text != categories[0] || ticks[1].text != categories[len(categories)-1] {
		t.Fatalf("end labels = %q and %q", ticks[0].text, ticks[1].text)
	}
}

// A category list that does not line up with the coordinates is refused;
// it would silently mislabel the axis.
func TestSurfaceSpecChecksCategoryCounts(t *testing.T) {
	base := func() surfaceSpec {
		return surfaceSpec{
			X: []float64{1, 2, 3}, Y: []float64{1, 2},
			Z:     [][]float64{{1, 2, 3}, {4, 5, 6}},
			Width: 800, Height: 600,
		}
	}
	valid := base()
	valid.XCategories = []string{"Y1", "Y2", "Y3"}
	valid.YCategories = []string{"A", "B"}
	if err := valid.validate(); err != nil {
		t.Fatalf("a Surface with one label per coordinate: %v", err)
	}
	// No categories at all is still valid: that is every Surface before v2.2.
	if err := base().validate(); err != nil {
		t.Fatalf("a Surface with no categories: %v", err)
	}
	for name, spec := range map[string]surfaceSpec{
		"too few x": func() surfaceSpec { s := base(); s.XCategories = []string{"one"}; return s }(),
		"too many y": func() surfaceSpec {
			s := base()
			s.YCategories = []string{"a", "b", "c"}
			return s
		}(),
	} {
		if err := spec.validate(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}
