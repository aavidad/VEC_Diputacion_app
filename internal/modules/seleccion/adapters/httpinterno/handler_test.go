package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/httpcomun"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

type preparadorPrueba struct{ err error }

func (p preparadorPrueba) PrepararOrdenRRHH(*http.Request) (application.Orden, error) {
	return application.Orden{}, p.err
}

type casosPrueba struct {
	consulta application.ConsultaListado
	detalle  string
	llamadas int
}

func (c *casosPrueba) Convocatorias(context.Context) ([]domain.ConvocatoriaPublicada, error) {
	c.llamadas++
	return nil, nil
}

func (c *casosPrueba) Listar(_ context.Context, _ application.Orden, q application.ConsultaListado) (application.PaginaRRHH, error) {
	c.llamadas++
	c.consulta = q
	return application.PaginaRRHH{Filas: []application.FilaVisible{{SolicitudRef: "sol_x", NumeroJustificante: "2026/SOL-000001",
		NombreVisible: "Reyes Álvarez, Antonio", DocumentoParcial: "***5678*", Estado: domain.EstadoPresentada, Turno: "libre"}}, CursorSiguiente: "9"}, nil
}

func (c *casosPrueba) Detalle(_ context.Context, _ application.Orden, s string) (application.FichaRRHH, error) {
	c.llamadas++
	c.detalle = s
	return application.FichaRRHH{Fila: ports.FilaSolicitudRRHH{SolicitudRef: s, PresentadaEn: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)},
		Datos:    domain.DatosPersonales{Nombre: "Antonio", DocumentoIdentidad: "12345678Z"},
		Meritos:  []domain.PuntosMerito{{MeritoDeclarado: domain.MeritoDeclarado{ClaveGrupo: "g", ClaveMerito: "m", Cantidad: "14"}, Titulo: "Meses", Unidad: "mes"}},
		Historia: []ports.EventoHistoria{{Tipo: "presentada", Version: 1, En: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)}}}, nil
}

func post(ruta, cuerpo string, cabeceras map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	return r
}

func TestConsultasRRHH(t *testing.T) {
	casos := &casosPrueba{}
	h, err := Nuevo(preparadorPrueba{}, casos)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, post(RutaConsultas, `{"convocatoria_ref":"bolsa-operario-diputacion-2026","cursor":"","limite":50}`, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"nombre_visible":"Reyes Álvarez, Antonio"`) || !strings.Contains(w.Body.String(), `"cursor_siguiente":"9"`) ||
		strings.Contains(w.Body.String(), "12345678Z") || casos.consulta.Limite != 50 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, post(RutaDetalleConsultas, `{"solicitud_ref":"sol_x"}`, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"documento_identidad":"12345678Z"`) || !strings.Contains(w.Body.String(), `"titulo":"Meses"`) || casos.detalle != "sol_x" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	llamadas := casos.llamadas
	for _, r := range []*http.Request{
		httptest.NewRequest(http.MethodGet, RutaConsultas, nil),
		post(RutaConsultas, `{"convocatoria_ref":"x","limite":1,"estado":"borrador"}`, nil),
		post(RutaDetalleConsultas, `{"solicitud_ref":"sol_x"}`, map[string]string{"Cookie": "a=b"}),
		post(RutaDetalleConsultas+"?x=1", `{"solicitud_ref":"sol_x"}`, nil),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 || w.Code >= 500 {
			t.Fatalf("%s %s aceptada: %d", r.Method, r.URL, w.Code)
		}
	}
	if casos.llamadas != llamadas {
		t.Fatal("una consulta rechazada llegó al caso de uso")
	}
	sin, _ := Nuevo(preparadorPrueba{err: httpcomun.ErrAutenticacionAusente}, casos)
	w = httptest.NewRecorder()
	sin.ServeHTTP(w, post(RutaConsultas, `{"convocatoria_ref":"x","limite":1}`, nil))
	if w.Code != 401 || casos.llamadas != llamadas {
		t.Fatalf("sin identidad: %d", w.Code)
	}
}
