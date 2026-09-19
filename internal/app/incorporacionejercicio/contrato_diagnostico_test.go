package incorporacionejercicio

import (
	"context"
	"errors"
	"strings"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFalloComposicionConservaCausaSinPublicarSuTexto(t *testing.T) {
	privada := errors.New("postgres://privado:secreto@dependencia")
	err := fallo(context.Background(), privada)
	if !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) || !errors.Is(err, privada) {
		t.Fatalf("el fallo no conserva centinela y causa: %v", err)
	}
	if strings.Contains(err.Error(), privada.Error()) {
		t.Fatalf("el texto del error expone la causa: %q", err.Error())
	}
}
