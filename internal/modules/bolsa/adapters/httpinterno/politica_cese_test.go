package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorPoliticaCesePrueba struct{ llamadas int }

func (p *preparadorPoliticaCesePrueba) PrepararConsultaPoliticaCese(*http.Request) (ports.OrdenConsultaPoliticaCese, error) {
	p.llamadas++
	return ports.OrdenConsultaPoliticaCese{}, nil
}

type consultorPoliticaCesePrueba struct{ llamadas int }

func (c *consultorPoliticaCesePrueba) Consultar(context.Context, ports.OrdenConsultaPoliticaCese) (domain.PoliticaCese, error) {
	c.llamadas++
	return domain.PoliticaCese{Version: 1, CatalogoRef: "catalogo:bolsa:cese:ejemplo-sintetico:v1", CatalogoSHA256: strings.Repeat("a", 64),
		Mapeo: map[string]string{"interinidad": "general"}, MesesGeneral: 5, MesesAcumulacion: 9,
		Computo: "fecha_cese_meses_calendario_ajuste_fin_mes", Estado: "ejemplo_sintetico",
		PublicadaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}, nil
}

func TestHandlerPoliticaCeseRechazaEntradaYMinimizaRespuesta(t *testing.T) {
	p, c := &preparadorPoliticaCesePrueba{}, &consultorPoliticaCesePrueba{}
	h, err := NuevoHandlerPoliticaCese(p, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{RutaPoliticaCese + "?participacion_ref=ajena", RutaPoliticaCese + "/ajena"} {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusOK || p.llamadas != 0 || c.llamadas != 0 {
			t.Fatalf("entrada no autorizada aceptada: %s: %d", ruta, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPoliticaCese, nil))
	if w.Code != http.StatusOK || p.llamadas != 1 || c.llamadas != 1 ||
		!strings.Contains(w.Body.String(), `"estado":"ejemplo_sintetico"`) ||
		!strings.Contains(w.Body.String(), `"meses_acumulacion":9`) ||
		strings.Contains(w.Body.String(), "participacion_ref") || strings.Contains(w.Body.String(), "recibo_ref") {
		t.Fatalf("respuesta de política no minimizada: %d %s", w.Code, w.Body.String())
	}
}
