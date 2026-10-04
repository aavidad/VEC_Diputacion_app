package temas

import (
	"math"
	"testing"
)

func TestContrasteSRGB(t *testing.T) {
	if got := Contraste("#000000", "#ffffff"); got != 21 {
		t.Fatalf("black/white: %v", got)
	}
	if got := Contraste("#ffffff", "#FFFFFF"); got != 1 {
		t.Fatalf("same: %v", got)
	}
	if Contraste("#687588", "#ffffff") < 3 {
		t.Fatal("control border")
	}
	if Contraste("#dde5ef", "#ffffff") >= 3 {
		t.Fatal("decorative border")
	}
	if !math.IsNaN(Contraste("url(x)", "#ffffff")) {
		t.Fatal("invalid color")
	}
}
