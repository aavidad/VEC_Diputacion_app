package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type operadorHistorialHTTPPrueba struct {
	operadorOperacionesHTTPPrueba
	historial ports.HistorialParticipacion
}

func (o operadorHistorialHTTPPrueba) ListarHistorial(context.Context, ports.SolicitudCambiarSituacionParticipacion) (ports.HistorialParticipacion, error) {
	return o.historial, nil
}

func TestOperacionesSituacionIncluyeTrazaDeValores(t *testing.T) {
	anterior, nuevo, version := "disponible", "no_disponible", "version:2"
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	o := operadorHistorialHTTPPrueba{historial: ports.HistorialParticipacion{Operaciones: []ports.RegistroOperacionSituacion{}, Cambios: []ports.CambioValorParticipacion{
		{Instante: ahora, ReciboRef: "recibo:01", Campo: "situacion", ValorAnterior: &anterior, ValorNuevo: &nuevo, Actor: "per_01"},
		{Instante: ahora, ReciboRef: "recibo:02", Campo: "telefono_1", ValorNuevo: &version, Actor: "per_01"},
	}}}
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, o)
	r := httptest.NewRequest(http.MethodGet, RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/operaciones", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var cuerpo struct {
		Data struct {
			Esquema string           `json:"esquema"`
			Cambios []map[string]any `json:"cambios"`
		} `json:"data"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || cuerpo.Data.Esquema != "vec.bolsa.rrhh.operaciones_situacion.v1" || len(cuerpo.Data.Cambios) != 2 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	c := cuerpo.Data.Cambios
	if c[0]["valor_anterior"] != "disponible" || c[0]["valor_nuevo"] != "no_disponible" || c[1]["valor_anterior"] != nil || c[1]["valor_nuevo"] != "version:2" || c[1]["campo"] != "telefono_1" {
		t.Fatalf("cambios=%v", c)
	}
}
