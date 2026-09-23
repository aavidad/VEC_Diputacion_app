package httppersonal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestMiBolsaSerializaUltimoEstadoSinMotivoLibre(t *testing.T) {
	desde := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	i := puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: desde.Add(time.Hour), Participaciones: []puertosbolsa.ParticipacionMiBolsa{{
		Bolsa: "bolsa:01", Categoria: "Auxiliar", Version: 3, OrdenInicial: 2, TotalInstantanea: 4,
		EstadoBolsa: "vigente", VigenteDesde: desde.Add(-time.Hour),
		SituacionActual:   &puertosbolsa.SituacionActualMiBolsa{Estado: "no_disponible", Desde: desde},
		UltimoLlamamiento: &puertosbolsa.UltimoLlamamientoMiBolsa{EmitidoEn: desde, Canal: "correo", Resultado: "enviado"},
	}}}
	contenido, err := json.Marshal(nuevaRespuesta(i))
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := os.ReadFile("testdata/mi_bolsa_situacion.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contenido, bytes.TrimSpace(esperado)) {
		t.Fatalf("cambió el payload HTTP propio: %s", contenido)
	}
	for _, requerido := range []string{`"situacion_actual"`, `"estado":"no_disponible"`, `"desde":"2026-09-20T10:00:00.000000Z"`, `"hasta":null`, `"ultimo_llamamiento"`, `"resultado":"enviado"`} {
		if !strings.Contains(string(contenido), requerido) {
			t.Fatalf("falta %s: %s", requerido, contenido)
		}
	}
	for _, prohibido := range []string{"motivo", "actor", "recibo_ref", "participacion_ref"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("se expone %s: %s", prohibido, contenido)
		}
	}
}

type preparadorPrueba struct {
	llamadas int
	err      error
}

func (p *preparadorPrueba) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	p.llamadas++
	return mibolsa.Orden{}, p.err
}

type consultorPrueba struct {
	llamadas int
	err      error
}

func (c *consultorPrueba) Consultar(context.Context, mibolsa.Orden) (puertosbolsa.InstantaneaMiBolsa, error) {
	c.llamadas++
	return puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC), Participaciones: []puertosbolsa.ParticipacionMiBolsa{{Bolsa: "bolsa:auxiliar", Categoria: "categoria:auxiliar", Version: 2, OrdenInicial: 3, TotalInstantanea: 20, EstadoBolsa: "vigente", VigenteDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}, c.err
}

func TestMiBolsaContratoCerradoSinSelector(t *testing.T) {
	p, c := new(preparadorPrueba), new(consultorPrueba)
	h, e := Nuevo(p, c)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa, nil))
	if w.Code != 200 || p.llamadas != 1 || c.llamadas != 1 {
		t.Fatalf("status=%d p=%d c=%d", w.Code, p.llamadas, c.llamadas)
	}
	s := w.Body.String()
	for _, prohibido := range []string{"candidato", "persona", "participacion_ref", "puntuacion", "disponibilidad", "can_"} {
		if strings.Contains(s, prohibido) {
			t.Fatalf("filtra %q: %s", prohibido, s)
		}
	}
	for _, requerido := range []string{"vec.bolsa.mi-bolsa.v1", "orden_inicial", "total_instantanea", "vigente_hasta", "certificado sintético", "Cl@ve", "FNMT", "DNIe"} {
		if !strings.Contains(s, requerido) {
			t.Fatalf("falta %q: %s", requerido, s)
		}
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("cache o cookie no seguros")
	}
}

func TestMiBolsaRechazaEntradaAntesDeLaAutoridad(t *testing.T) {
	casos := []struct {
		nombre, metodo, ruta string
		cookie               bool
	}{{"query", http.MethodGet, RutaMiBolsa + "?candidato=can_x", false}, {"cuerpo", http.MethodGet, RutaMiBolsa, false}, {"cookie", http.MethodGet, RutaMiBolsa, true}, {"metodo", http.MethodPost, RutaMiBolsa, false}}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			p, c := new(preparadorPrueba), new(consultorPrueba)
			h, _ := Nuevo(p, c)
			var cuerpo *strings.Reader
			if caso.nombre == "cuerpo" {
				cuerpo = strings.NewReader("{}")
			} else {
				cuerpo = strings.NewReader("")
			}
			r := httptest.NewRequest(caso.metodo, caso.ruta, cuerpo)
			if caso.cookie {
				r.Header.Set("Cookie", "x=y")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 && w.Code != 405 {
				t.Fatalf("status=%d", w.Code)
			}
			if !strings.Contains(w.Body.String(), `{"error":{"codigo":`) {
				t.Fatalf("error sin sobre: %s", w.Body.String())
			}
			if p.llamadas != 0 || c.llamadas != 0 {
				t.Fatalf("se invoco frontera: p=%d c=%d", p.llamadas, c.llamadas)
			}
		})
	}
}

func TestMiBolsaClasificaDenegacionYDependencia(t *testing.T) {
	for _, caso := range []struct {
		err    error
		status int
	}{{errors.Join(dominiovec.ErrAutorizacionDenegada, errors.New("detalle")), 403}, {ErrDependenciaNoDisponible, 503}, {errors.Join(dominiovec.ErrAutorizacionDenegada, puertosvec.ErrRegistroDecisionNoDisponible), 503}} {
		p := &preparadorPrueba{err: caso.err}
		c := new(consultorPrueba)
		h, _ := Nuevo(p, c)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa, nil))
		if w.Code != caso.status || c.llamadas != 0 {
			t.Fatalf("status=%d llamadas=%d", w.Code, c.llamadas)
		}
		if !strings.Contains(w.Body.String(), `{"error":{"codigo":`) {
			t.Fatalf("error sin sobre: %s", w.Body.String())
		}
	}
}

func TestMiBolsaErroresAplicacionSinFiltrarDetalle(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		err    error
		status int
		codigo string
	}{
		{"autenticacion", ErrAutenticacionAusente, 401, "autenticacion_requerida"},
		{"denegada", dominiovec.ErrAutorizacionDenegada, 403, "acceso_denegado"},
		{"concesion V3", errors.Join(dominiovec.ErrAutorizacionDenegada, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible), 503, "servicio_no_disponible"},
		{"denegacion V3", puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, 503, "servicio_no_disponible"},
		{"material", puertosbolsa.ErrMaterialMiBolsaNoDisponible, 503, "servicio_no_disponible"},
		{"fallo inesperado", errors.New("detalle privado"), 500, "error_interno"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p := new(preparadorPrueba)
			c := &consultorPrueba{err: errors.Join(caso.err, errors.New("detalle privado"))}
			h, _ := Nuevo(p, c)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsa, nil))
			esperado := `{"error":{"codigo":"` + caso.codigo + `"}}`
			if w.Code != caso.status || w.Body.String() != esperado || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
				t.Fatalf("status=%d cuerpo=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMiBolsaRechazaCabecerasYVariantesDeRuta(t *testing.T) {
	for _, cabecera := range []string{"Cookie", "Proxy-Authorization", "X-Vec-Candidato", "X-Auth-User", "X-Forwarded-User", "X-Remote-User", "Remote-User"} {
		t.Run(cabecera, func(t *testing.T) {
			p, c := new(preparadorPrueba), new(consultorPrueba)
			h, _ := Nuevo(p, c)
			r := httptest.NewRequest(http.MethodGet, RutaMiBolsa, nil)
			r.Header[cabecera] = []string{"valor"}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 || p.llamadas != 0 || c.llamadas != 0 {
				t.Fatal("cabecera cruzó frontera")
			}
		})
	}
	for _, ruta := range []string{RutaMiBolsa + "/", RutaMiBolsa + "/otro", "/api/vec/bolsa/%6di-bolsa"} {
		t.Run(ruta, func(t *testing.T) {
			p, c := new(preparadorPrueba), new(consultorPrueba)
			h, _ := Nuevo(p, c)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
			if w.Code != 404 || p.llamadas != 0 || c.llamadas != 0 {
				t.Fatal("ruta no canónica cruzó frontera")
			}
		})
	}
}
