package referencias

import (
	"context"
	"regexp"
	"testing"
)

func TestReferenciasOpacasDistintas(t *testing.T) {
	patron := regexp.MustCompile(`^sol_[A-Za-z0-9_-]{22,64}$`)
	a, err1 := Aleatorio{}.NuevaReferenciaSolicitud(context.Background())
	b, err2 := Aleatorio{}.NuevaReferenciaSolicitud(context.Background())
	if err1 != nil || err2 != nil || a == b || !patron.MatchString(a) || !patron.MatchString(b) {
		t.Fatalf("referencias inválidas: %q %q", a, b)
	}
}
