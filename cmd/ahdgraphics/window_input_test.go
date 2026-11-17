package main

import "testing"

func TestWindowSamplesPointerAtDisplayRate(t *testing.T) {
	if windowTPS != 60 {
		t.Fatalf("window tick rate is %d, want 60", windowTPS)
	}
}

func TestDragContinuesOutsideOnlyWhenThePressBeganInside(t *testing.T) {
	const width, height = 200, 100
	var tracker pointerTracker

	if _, _, _, ok := tracker.event(pointerSample{x: 100, y: 50, pressed: true, justPressed: true}, width, height, 1); ok {
		t.Fatal("the press itself was reported as a drag")
	}
	kind, x, y, ok := tracker.event(pointerSample{x: 110, y: 40, pressed: true}, width, height, 1)
	if !ok || kind != "drag" || x != 10 || y != 10 {
		t.Fatalf("first movement = %s %v %v ok=%v", kind, x, y, ok)
	}
	kind, x, y, ok = tracker.event(pointerSample{x: -20, y: 40, pressed: true}, width, height, 1)
	if !ok || kind != "drag" || x != -120 || y != 10 {
		t.Fatalf("drag outside = %s %v %v ok=%v", kind, x, y, ok)
	}
	if _, _, _, ok := tracker.event(pointerSample{x: -20, y: 40, pressed: false, justReleased: true}, width, height, 1); ok {
		t.Fatal("releasing without moving reported another drag")
	}
	kind, _, _, ok = tracker.event(pointerSample{x: 20, y: 40, pressed: false}, width, height, 1)
	if !ok || kind != "move" {
		t.Fatalf("after release = %s ok=%v", kind, ok)
	}

	var outside pointerTracker
	if _, _, _, ok := outside.event(pointerSample{x: -5, y: 10, pressed: true, justPressed: true}, width, height, 1); ok {
		t.Fatal("an outside press was reported")
	}
	if _, _, _, ok := outside.event(pointerSample{x: 20, y: 10, pressed: true}, width, height, 1); ok {
		t.Fatal("a press that started outside became a drag inside the Canvas")
	}
}
