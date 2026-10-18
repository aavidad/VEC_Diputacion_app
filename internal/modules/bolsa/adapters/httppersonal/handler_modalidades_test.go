package httppersonal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type consultorModalidadesHTTPPrueba struct {
	consultorPrueba
	todas int
}

func (c *consultorModalidadesHTTPPrueba) ConsultarTodasLasOfertas(context.Context, mibolsa.Orden) (bolsa.InstantaneaMiBolsa, error) {
	c.todas++
	return bolsa.InstantaneaMiBolsa{}, nil
}

func TestMiBolsaModalidadesSelectorSoloDeVista(t *testing.T) {
	for _, caso := range []struct {
		query  string
		codigo int
		todas  int
	}{
		{"", 200, 0},
		{"?ofertas=todas", 200, 1},
		{"?ofertas=todas&candidato=can_ajeno", 400, 0},
		{"?ofertas=todas&ofertas=todas", 400, 0},
		{"?ofertas=%74odas", 400, 0},
		{"?ofertas=otras", 400, 0},
		{"?", 400, 0},
	} {
		t.Run(caso.query, func(t *testing.T) {
			p, c := new(preparadorPrueba), new(consultorModalidadesHTTPPrueba)
			h, err := Nuevo(p, c)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa+caso.query, nil))
			if w.Code != caso.codigo || c.todas != caso.todas {
				t.Fatalf("código=%d todas=%d", w.Code, c.todas)
			}
			if caso.codigo == 400 && (p.llamadas != 0 || c.llamadas != 0) ||
				caso.todas == 1 && (p.llamadas != 1 || c.llamadas != 0) {
				t.Fatal("el selector alteró la frontera de identidad o eligió otra consulta")
			}
		})
	}
}

func TestMiBolsaModalidadesTodasExigeCapacidadYAutenticacion(t *testing.T) {
	p, c := new(preparadorPrueba), new(consultorPrueba)
	h, err := Nuevo(p, c)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa+"?ofertas=todas", nil))
	if w.Code != 400 || p.llamadas != 0 || c.llamadas != 0 {
		t.Fatal("una composición sin filtro aceptó un selector que no implementa")
	}
	p.err = ErrAutenticacionAusente
	completo := new(consultorModalidadesHTTPPrueba)
	h, err = Nuevo(p, completo)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa+"?ofertas=todas", nil))
	if w.Code != 401 || completo.todas != 0 || completo.llamadas != 0 {
		t.Fatal("Ver todas omitió la autenticación de la misma candidatura")
	}
}
