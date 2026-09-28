package httppersonal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorHistorialPrueba struct{}

func (preparadorHistorialPrueba) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	return mibolsa.Orden{}, nil
}

type consultorHistorialPrueba struct {
	pagina   int
	llamadas int
}

func (c *consultorHistorialPrueba) ConsultarHistorial(_ context.Context, _ mibolsa.Orden, pagina int) (puertosbolsa.PaginaHistorialMiBolsa, error) {
	c.pagina, c.llamadas = pagina, c.llamadas+1
	ahora := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	return puertosbolsa.PaginaHistorialMiBolsa{ConsultadaEn: ahora, Pagina: pagina, Tamano: 20, Items: []puertosbolsa.HechoHistorialMiBolsa{{
		Clase: "contrato_bolsa", Bolsa: "bolsa:auxiliar", Categoria: "Auxiliar", OcurridoEn: ahora,
		Tipo: "incorporacion", Procedencia: "evento_ct_recibido",
	}}}, nil
}

func TestHistorialPropioGETPaginaYMinimizacion(t *testing.T) {
	consultor := new(consultorHistorialPrueba)
	h, err := NuevoHistorial(preparadorHistorialPrueba{}, consultor)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsaHistorial+"?pagina=2", nil))
	if w.Code != 200 || consultor.pagina != 2 || consultor.llamadas != 1 {
		t.Fatalf("respuesta/página: %d %+v", w.Code, consultor)
	}
	var cuerpo struct {
		Data struct {
			Esquema   string   `json:"esquema"`
			Campos    []string `json:"campos_visibles"`
			Historial struct {
				Items []map[string]any `json:"items"`
			} `json:"historial"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if cuerpo.Data.Esquema != puertosbolsa.EsquemaHistorialMiBolsa || len(cuerpo.Data.Campos) != 3 || len(cuerpo.Data.Historial.Items) != 1 {
		t.Fatalf("contrato de salida: %s", w.Body.String())
	}
	item := cuerpo.Data.Historial.Items[0]
	if len(item) != 9 || item["inicio"] != nil || item["fin_previsto"] != nil || item["modalidad_clave"] != nil || item["procedencia"] != "evento_ct_recibido" {
		t.Fatalf("proyección no minimizada: %#v", item)
	}
	for _, prohibido := range []string{"candidato_ref", "participacion_ref", "expediente_ref", "actor", "justificante_ref"} {
		if _, ok := item[prohibido]; ok {
			t.Fatalf("dato interno expuesto: %s", prohibido)
		}
	}
}

func TestHistorialPropioRechazaSelectorYPaginaNoCanonica(t *testing.T) {
	consultor := new(consultorHistorialPrueba)
	h, err := NuevoHistorial(preparadorHistorialPrueba{}, consultor)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"?pagina=0", "?pagina=01", "?pagina=10001", "?pagina=1&candidato=can_ajeno", "?bolsa=bolsa:ajena", "?pagina=%31"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsaHistorial+query, nil))
		if w.Code != 400 {
			t.Fatalf("%q: HTTP %d", query, w.Code)
		}
	}
	if consultor.llamadas != 0 {
		t.Fatalf("consulta ejecutada pese a selector: %d", consultor.llamadas)
	}
}
