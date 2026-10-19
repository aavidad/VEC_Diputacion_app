package httppersonal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preferenciasModalidadesHTTPPrueba struct {
	seleccion         []string
	guardadas, leidas int
	reutilizado       bool
}

func (s *preferenciasModalidadesHTTPPrueba) GuardarModalidades(_ context.Context, _ mibolsa.Orden, seleccion []string, version uint64, _ string) (bolsa.ReciboPreferenciasModalidades, error) {
	s.guardadas++
	s.seleccion = seleccion
	return bolsa.ReciboPreferenciasModalidades{ReciboRef: "recibo:modalidades", Version: version + 1,
		Modalidades: seleccion, RegistradoEn: time.Date(2026, 10, 18, 8, 0, 0, 1000, time.UTC), Reutilizado: s.reutilizado}, nil
}

func (s *preferenciasModalidadesHTTPPrueba) ConsultarPreferenciasModalidades(context.Context, mibolsa.Orden) (bolsa.EstadoPreferenciasModalidades, error) {
	s.leidas++
	return bolsa.EstadoPreferenciasModalidades{ModalidadesGuardadas: s.seleccion}, nil
}

func TestPreferenciasModalidadesHTTPDistingueTodasYNinguna(t *testing.T) {
	for _, contenido := range []string{"null", "[]"} {
		t.Run(contenido, func(t *testing.T) {
			p, s := new(preparadorPreferenciasPrueba), new(preferenciasModalidadesHTTPPrueba)
			h, err := NuevoHandlerPreferenciasModalidades(p, s, s)
			if err != nil {
				t.Fatal(err)
			}
			for _, codigo := range []int{201, 200} {
				s.reutilizado = codigo == 200
				r := peticionPreferenciasPrueba(`{"modalidades":` + contenido + `,"version_esperada":0,"clave_idempotencia":"modalidades-001"}`)
				r.URL.RawQuery = consultaPreferenciasModalidades
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				var respuesta map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
					t.Fatal(err)
				}
				if w.Code != codigo || string(respuesta["modalidades"]) != contenido || (s.seleccion == nil) != (contenido == "null") {
					t.Fatalf("código=%d selección=%s", w.Code, respuesta["modalidades"])
				}
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", RutaPreferenciasBolsas+"?tipo=modalidades", nil))
			var respuesta map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || string(respuesta["modalidades_guardadas"]) != contenido || string(respuesta["modalidades_disponibles"]) != "[]" || s.leidas != 1 {
				t.Fatalf("lectura=%d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestPreferenciasModalidadesHTTPFronteraAntesDeIdentidad(t *testing.T) {
	p, s := new(preparadorPreferenciasPrueba), new(preferenciasModalidadesHTTPPrueba)
	modalidades, err := NuevoHandlerPreferenciasModalidades(p, s, s)
	if err != nil {
		t.Fatal(err)
	}
	bolsas, err := NuevoHandlerPreferenciasBolsasConLectura(p, new(ordenadorPreferenciasPrueba), new(lectorPreferenciasPrueba))
	if err != nil {
		t.Fatal(err)
	}
	h, err := NuevoSelectorPreferenciasModalidades(bolsas, modalidades)
	if err != nil {
		t.Fatal(err)
	}
	for _, consulta := range []string{"tipo=otras", "tipo=%6dodalidades", "tipo=modalidades&tipo=modalidades", "tipo=modalidades&candidato_ref=ajeno"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", RutaPreferenciasBolsas+"?"+consulta, nil))
		if w.Code != 400 || p.llamadas != 0 || s.leidas != 0 {
			t.Fatalf("consulta=%s código=%d", consulta, w.Code)
		}
	}
	for _, cuerpo := range []string{
		`{"modalidades":[],"modalidades":null,"version_esperada":0,"clave_idempotencia":"modalidades-001"}`,
		`{"modalidades":[],"version_esperada":null,"clave_idempotencia":"modalidades-001"}`,
		`{"modalidades":[],"version_esperada":0,"clave_idempotencia":null}`,
		`{"modalidades":[],"version_esperada":0,"clave_idempotencia":"modalidades-001","fuente":{}}`,
	} {
		r := httptest.NewRequest(http.MethodPut, RutaPreferenciasBolsas+"?tipo=modalidades", strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || p.llamadas != 0 || s.guardadas != 0 {
			t.Fatalf("cuerpo rechazado=%d", w.Code)
		}
	}
	// La consulta sin selector conserva el servicio de bolsas.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", RutaPreferenciasBolsas, nil))
	if w.Code != 200 || p.llamadas != 1 || s.leidas != 0 {
		t.Fatalf("bolsas=%d", w.Code)
	}
}

type preferenciasModalidadesHTTPFallo struct{ err error }

func (f preferenciasModalidadesHTTPFallo) GuardarModalidades(context.Context, mibolsa.Orden, []string, uint64, string) (bolsa.ReciboPreferenciasModalidades, error) {
	return bolsa.ReciboPreferenciasModalidades{}, f.err
}

func (f preferenciasModalidadesHTTPFallo) ConsultarPreferenciasModalidades(context.Context, mibolsa.Orden) (bolsa.EstadoPreferenciasModalidades, error) {
	return bolsa.EstadoPreferenciasModalidades{}, f.err
}

// Los errores del catálogo y del almacén llegan unidos a causas internas; la
// respuesta solo lleva el código público y nunca el texto del error.
func TestPreferenciasModalidadesHTTPNoMuestraErroresInternos(t *testing.T) {
	interno := errors.New("catalogo c12 ruta /privada dsn=postgres://secreto")
	casos := []struct {
		err    error
		codigo int
		cuerpo string
	}{
		{errors.Join(bolsa.ErrPreferenciasNoDisponibles, interno), 503, "servicio_no_disponible"},
		{errors.Join(bolsa.ErrOrdenPreferenciasInvalido, interno), 422, "orden_invalido"},
		{errors.Join(bolsa.ErrPreferenciasDesactualizadas, interno), 409, "preferencias_desactualizadas"},
		{interno, 500, "error_interno"},
	}
	for _, c := range casos {
		h, err := NuevoHandlerPreferenciasModalidades(new(preparadorPreferenciasPrueba),
			preferenciasModalidadesHTTPFallo{c.err}, preferenciasModalidadesHTTPFallo{c.err})
		if err != nil {
			t.Fatal(err)
		}
		lectura := httptest.NewRequest(http.MethodGet, RutaPreferenciasBolsas+"?tipo=modalidades", nil)
		escritura := peticionPreferenciasPrueba(`{"modalidades":["a"],"version_esperada":0,"clave_idempotencia":"modalidades-001"}`)
		escritura.URL.RawQuery = consultaPreferenciasModalidades
		for _, r := range []*http.Request{lectura, escritura} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != c.codigo || !strings.Contains(w.Body.String(), c.cuerpo) ||
				strings.Contains(w.Body.String(), "secreto") || strings.Contains(w.Body.String(), "privada") {
				t.Fatalf("%s %v: código=%d cuerpo=%s", r.Method, c.err, w.Code, w.Body.String())
			}
		}
	}
}
