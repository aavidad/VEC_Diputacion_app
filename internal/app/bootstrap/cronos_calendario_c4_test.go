package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
)

func TestMontajeCalendarioC4NoPublicaRutaConFuenteIncompleta(t *testing.T) {
	rutas := map[string]http.Handler{}
	if err := montarCalendarioLaboralC4(context.Background(), rutas, nil, nil, nil, nil, nil, nil, cronosdomain.AcreditacionCanalMarcaje{}); !errors.Is(err, ErrMontajeCalendarioLaboralC4) {
		t.Fatalf("montaje incompleto: %v", err)
	}
	if _, ok := rutas[cronoshttp.RutaCalendarioLaboralPropioC4]; ok {
		t.Fatal("ruta publicada sin fuente")
	}
}
