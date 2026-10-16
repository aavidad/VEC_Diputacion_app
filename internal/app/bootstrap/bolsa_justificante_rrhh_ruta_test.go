package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type auditorJustificanteRRHHPrueba struct {
	llamada func(vecports.OrdenAuditoriaFronteraRutaExacta) error
}

func (a auditorJustificanteRRHHPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	return a.llamada(o)
}

func TestJustificanteRRHHAuditaRechazoAntesDeEntregarlo(t *testing.T) {
	for _, falla := range []bool{false, true} {
		t.Run(map[bool]string{false: "asentado", true: "sin_asiento"}[falla], func(t *testing.T) {
			w := httptest.NewRecorder()
			asientos := 0
			a := auditorJustificanteRRHHPrueba{func(o vecports.OrdenAuditoriaFronteraRutaExacta) error {
				asientos++
				if w.Code != http.StatusOK || w.Body.Len() != 0 {
					t.Fatal("entregó el error antes de asentar")
				}
				if o.Validar() != nil || o.ActorRef != "" || o.Ruta != "/api/vec/bolsa/justificantes-renuncia/rrhh" {
					t.Fatal("asiento inválido o con identidad no acreditada")
				}
				if falla {
					return errors.New("sin asiento")
				}
				return nil
			}}
			ruta := &vechttp.RutaExacta{Ruta: "/api/vec/bolsa/justificantes-renuncia/rrhh", Manejador: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })}
			h := componerRaizConJustificanteRenunciaRRHH(http.NotFoundHandler(), ruta, a)
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
			esperado := http.StatusForbidden
			if falla {
				esperado = http.StatusServiceUnavailable
			}
			if falla && !strings.Contains(w.Body.String(), "servicio_no_disponible") {
				t.Fatal("cuerpo 503 incoherente", w.Body.String())
			}
			if asientos != 1 || w.Code != esperado {
				t.Fatalf("asientos=%d estado=%d", asientos, w.Code)
			}
		})
	}
}

func TestJustificanteRRHHNoInterceptaRutasAjenas(t *testing.T) {
	h := componerRaizConJustificanteRenunciaRRHH(http.NotFoundHandler(), &vechttp.RutaExacta{
		Ruta: "/api/vec/bolsa/justificantes-renuncia/rrhh", Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("interceptó ruta ajena") })}, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/justificantes-renuncia/rrhh-ajena", nil))
	if w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
}
