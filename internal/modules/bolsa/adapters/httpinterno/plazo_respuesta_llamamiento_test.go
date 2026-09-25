package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type consultorPlazoPrueba struct {
	plazo    puertosbolsa.PlazoRespuestaLlamamiento
	err      error
	llamadas int
}

func (c *consultorPlazoPrueba) ConsultarPlazoRespuesta(context.Context) (puertosbolsa.PlazoRespuestaLlamamiento, error) {
	c.llamadas++
	return c.plazo, c.err
}

func manejadorPlazoPrueba(t *testing.T, c *consultorPlazoPrueba) http.Handler {
	t.Helper()
	h, err := NuevoHandlerPlazoRespuestaLlamamiento(c)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestPlazoRespuestaHTTPConCatalogo(t *testing.T) {
	calculado := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	c := &consultorPlazoPrueba{plazo: puertosbolsa.PlazoRespuestaLlamamiento{
		Configurada: true,
		Regla: puertosbolsa.ReglaPlazoRespuesta{Etiqueta: "Plazo", Texto: "Un día hábil.", Referencia: "vec.bolsa.reglas:1:b05.plazo_respuesta",
			Origen: "reglamento", Articulo: "art. 9", Ejemplo: false},
		CalculadoEn: calculado, UltimoDia: "2026-09-29",
		VenceEn:      time.Date(2026, 9, 29, 21, 59, 59, 0, time.UTC),
		VenceAntesDe: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC),
	}}
	w := httptest.NewRecorder()
	manejadorPlazoPrueba(t, c).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPlazoRespuestaLlamamiento, nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("estado=%d cabeceras=%v", w.Code, w.Header())
	}
	var sobre struct {
		Data struct {
			Esquema     string `json:"esquema"`
			Configurada bool   `json:"configurada"`
			Regla       struct {
				Texto, Referencia, Origen, Articulo string
				Ejemplo                             bool
			} `json:"regla"`
			Vencimiento struct {
				UltimoDia string    `json:"ultimo_dia"`
				VenceEn   time.Time `json:"vence_en"`
			} `json:"vencimiento"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil {
		t.Fatal(err)
	}
	d := sobre.Data
	if d.Esquema != puertosbolsa.EsquemaPlazoRespuestaLlamamiento || !d.Configurada || d.Regla.Texto != "Un día hábil." ||
		d.Regla.Referencia != "vec.bolsa.reglas:1:b05.plazo_respuesta" || d.Regla.Origen != "reglamento" ||
		d.Regla.Articulo != "art. 9" || d.Regla.Ejemplo || d.Vencimiento.UltimoDia != "2026-09-29" ||
		!d.Vencimiento.VenceEn.Equal(c.plazo.VenceEn) {
		t.Fatalf("contrato inesperado: %s", w.Body.String())
	}
}

func TestPlazoRespuestaHTTPSinCatalogo(t *testing.T) {
	w := httptest.NewRecorder()
	manejadorPlazoPrueba(t, &consultorPlazoPrueba{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPlazoRespuestaLlamamiento, nil))
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"data":{"esquema":"vec.bolsa.llamamiento.plazo_respuesta.v1","configurada":false}}` {
		t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
}

func TestPlazoRespuestaHTTPRechazaEntradasYErrores(t *testing.T) {
	casos := []struct {
		nombre string
		metodo string
		ruta   string
		cab    map[string]string
		err    error
		estado int
	}{
		{"metodo", http.MethodPost, RutaPlazoRespuestaLlamamiento, nil, nil, http.StatusMethodNotAllowed},
		{"consulta", http.MethodGet, RutaPlazoRespuestaLlamamiento + "?plazo=3", nil, nil, http.StatusBadRequest},
		{"cookie", http.MethodGet, RutaPlazoRespuestaLlamamiento, map[string]string{"Cookie": "a=b"}, nil, http.StatusBadRequest},
		{"identidad heredada", http.MethodGet, RutaPlazoRespuestaLlamamiento, map[string]string{"X-VEC-Actor": "rrhh"}, nil, http.StatusBadRequest},
		{"otra ruta", http.MethodGet, RutaPlazoRespuestaLlamamiento + "/x", nil, nil, http.StatusNotFound},
		{"denegada", http.MethodGet, RutaPlazoRespuestaLlamamiento, nil, dominiovec.ErrAutorizacionDenegada, http.StatusForbidden},
		{"no disponible", http.MethodGet, RutaPlazoRespuestaLlamamiento, nil, puertosbolsa.ErrPlazoRespuestaNoDisponible, http.StatusServiceUnavailable},
	}
	for _, caso := range casos {
		c := &consultorPlazoPrueba{err: caso.err}
		r := httptest.NewRequest(caso.metodo, caso.ruta, nil)
		for k, v := range caso.cab {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		manejadorPlazoPrueba(t, c).ServeHTTP(w, r)
		if w.Code != caso.estado {
			t.Errorf("%s: estado=%d cuerpo=%s", caso.nombre, w.Code, w.Body.String())
		}
		if caso.err == nil && c.llamadas != 0 {
			t.Errorf("%s: la entrada rechazada llegó al caso de uso", caso.nombre)
		}
		if strings.Contains(w.Body.String(), "regla") {
			t.Errorf("%s: un error publicó la regla: %s", caso.nombre, w.Body.String())
		}
	}
	if _, err := NuevoHandlerPlazoRespuestaLlamamiento(nil); err == nil {
		t.Fatal("un handler sin consultor debe rechazarse")
	}
}
