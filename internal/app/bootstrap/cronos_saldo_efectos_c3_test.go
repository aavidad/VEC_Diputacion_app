package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
)

func TestSaldoEfectosC3NoPublicaSinCRN25(t *testing.T) {
	rutas := map[string]http.Handler{}
	if err := montarSaldoEfectosC3(context.Background(), rutas, nil, nil, nil, cronosdomain.AcreditacionCanalMarcaje{}, nil); !errors.Is(err, ErrMontajeSaldoEfectosC3) {
		t.Fatalf("montaje incompleto: %v", err)
	}
	if len(rutas) != 0 {
		t.Fatal("ruta publicada sin lector")
	}
}
