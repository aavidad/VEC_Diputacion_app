package httppersonal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// El indicador solo sale cuando la vista ya aplica la elección de modalidades.
func TestRespuestaIndicaOfertasSegunModalidades(t *testing.T) {
	for _, aplicada := range []bool{false, true} {
		cuerpo, err := json.Marshal(nuevaRespuesta(puertosbolsa.InstantaneaMiBolsa{OfertasSegunModalidades: aplicada}))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(cuerpo), `"ofertas_segun_modalidades":true`) != aplicada ||
			strings.Contains(string(cuerpo), `"ofertas_segun_modalidades":false`) {
			t.Fatalf("aplicada=%v cuerpo=%s", aplicada, cuerpo)
		}
	}
}

type consultorFiltradoPrueba struct {
	consultorPrueba
	todas int
}

func (c *consultorFiltradoPrueba) ConsultarTodasOfertas(ctx context.Context, o mibolsa.Orden) (puertosbolsa.InstantaneaMiBolsa, error) {
	c.todas++
	return c.consultorPrueba.Consultar(ctx, o)
}

// «Ver todas» usa la consulta sin filtro solo con la consulta exacta; las
// variantes se rechazan antes de la autoridad.
func TestMiBolsaVerTodasLasOfertas(t *testing.T) {
	casos := []struct {
		ruta                    string
		codigo, filtrada, todas int
	}{
		{RutaMiBolsa, 200, 1, 0},
		{RutaMiBolsa + "?ofertas=todas", 200, 0, 1},
		{RutaMiBolsa + "?ofertas=todas&x=1", 400, 0, 0},
		{RutaMiBolsa + "?ofertas=algunas", 400, 0, 0},
		{RutaMiBolsa + "?ofertas=Todas", 400, 0, 0},
		{RutaMiBolsa + "?", 400, 0, 0},
	}
	for _, caso := range casos {
		p, c := new(preparadorPrueba), new(consultorFiltradoPrueba)
		h, _ := Nuevo(p, c)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
		if w.Code != caso.codigo || c.llamadas-c.todas != caso.filtrada || c.todas != caso.todas {
			t.Fatalf("%s: status=%d filtrada=%d todas=%d", caso.ruta, w.Code, c.llamadas-c.todas, c.todas)
		}
	}
	// Sin filtro compuesto, «Ver todas» es la consulta normal.
	c := new(consultorPrueba)
	h, _ := Nuevo(new(preparadorPrueba), c)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa+"?ofertas=todas", nil))
	if w.Code != 200 || c.llamadas != 1 {
		t.Fatalf("status=%d llamadas=%d", w.Code, c.llamadas)
	}
}
