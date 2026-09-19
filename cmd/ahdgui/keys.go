package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Key names are the same normalized names AhdCode programs see in
// Window.onKey and Graphics' Canvas.onKey. Letters and digits name the
// physical key by its US-layout label, whatever the active keyboard layout.
// Keep this table identical to cmd/ahdgraphics/events.go.
var keyNames = map[ebiten.Key]string{
	ebiten.KeyArrowUp: "ArrowUp", ebiten.KeyArrowDown: "ArrowDown",
	ebiten.KeyArrowLeft: "ArrowLeft", ebiten.KeyArrowRight: "ArrowRight",
	ebiten.KeyEnter: "Enter", ebiten.KeyNumpadEnter: "Enter",
	ebiten.KeyEscape: "Escape", ebiten.KeySpace: "Space", ebiten.KeyTab: "Tab",
	ebiten.KeyBackspace: "Backspace", ebiten.KeyDelete: "Delete",
	ebiten.KeyHome: "Home", ebiten.KeyEnd: "End",
	ebiten.KeyPageUp: "PageUp", ebiten.KeyPageDown: "PageDown",
}

func init() {
	for offset := 0; offset < 26; offset++ {
		keyNames[ebiten.KeyA+ebiten.Key(offset)] = string(rune('A' + offset))
	}
	for offset := 0; offset < 10; offset++ {
		keyNames[ebiten.KeyDigit0+ebiten.Key(offset)] = string(rune('0' + offset))
	}
}

func keyNameKnown(name string) bool {
	for _, known := range keyNames {
		if known == name {
			return true
		}
	}
	return false
}
