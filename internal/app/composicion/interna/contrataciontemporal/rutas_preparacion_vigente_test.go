package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

type consultorPreparacionVigenteComposicionPrueba struct{ llamadas int }

func (c *consultorPreparacionVigenteComposicionPrueba) ConsultarParaAdaptador(
	context.Context, application.SolicitudConsultarPreparacionCoberturaVigente,
) (application.ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador, error) {
	c.llamadas++
	return application.ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{},
		errors.New("no debe consultar sin autoridad")
}

func TestRutasPreparacionVigenteSoloSeMontanConConsultorYAutoridad(t *testing.T) {
	base := dependenciasRutasPrueba()
	rutas, err := NuevasRutas(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range rutas {
		if ruta.Ruta == httpinterno.RutaPreparacionCoberturaVigente {
			t.Fatal("la ruta GET se montó sin consultor")
		}
	}
	consultor := &consultorPreparacionVigenteComposicionPrueba{}
	base.ConsultorPreparacionVigente = consultor
	rutas, err = NuevasRutas(base)
	if err != nil {
		t.Fatal(err)
	}
	var manejador http.Handler
	for _, ruta := range rutas {
		if ruta.Ruta == httpinterno.RutaPreparacionCoberturaVigente {
			if manejador != nil {
				t.Fatal("la ruta GET se montó más de una vez")
			}
			manejador = ruta.Manejador
		}
	}
	if manejador == nil {
		t.Fatal("falta ruta GET")
	}
	w := httptest.NewRecorder()
	manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		httpinterno.RutaPreparacionCoberturaVigente, nil))
	if w.Code == http.StatusOK || consultor.llamadas != 0 {
		t.Fatalf("la autoridad no cerró la lectura: HTTP %d, consultas %d", w.Code, consultor.llamadas)
	}
	base.AutoridadCobertura = nil
	if rutas, err := NuevasRutas(base); err == nil || len(rutas) != 0 {
		t.Fatal("la composición aceptó un GET sin autoridad")
	}
}
