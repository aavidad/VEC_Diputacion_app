package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorPoliticaPrueba struct{ denegar, denegarConsulta, puedePublicar bool }

func (p preparadorPoliticaPrueba) PrepararConsultaPoliticaOfertas(_ context.Context, bolsa string) (ConsultaPoliticaOfertasPreparada, error) {
	if p.denegar || p.denegarConsulta {
		return ConsultaPoliticaOfertasPreparada{}, dominiovec.ErrAutorizacionDenegada
	}
	return ConsultaPoliticaOfertasPreparada{BolsaRef: bolsa, PuedePublicar: p.puedePublicar,
		Material: ports.ConsultaPoliticaOfertasAutorizada{BolsaRef: bolsa}}, nil
}
func (p preparadorPoliticaPrueba) PrepararPublicacionPoliticaOfertas(_ context.Context, e EntradaPublicarPoliticaOfertas) (ports.ComandoPublicarPoliticaOfertas, error) {
	if p.denegar {
		return ports.ComandoPublicarPoliticaOfertas{}, dominiovec.ErrAutorizacionDenegada
	}
	return ports.ComandoPublicarPoliticaOfertas{BolsaRef: e.BolsaRef, VersionEsperada: e.VersionEsperada, ClaveIdempotencia: e.ClaveIdempotencia, Politica: e.Politica, ActorRef: "per_abcdefghijklmnopqrstuv"}, nil
}

type operadorPoliticaPrueba struct {
	publicada   bool
	consultadas int
}

func (o *operadorPoliticaPrueba) ConsultarAutorizada(_ context.Context, c ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error) {
	o.consultadas++
	return ports.VersionPoliticaOfertas{BolsaRef: c.BolsaRef, Ejemplo: true}, nil
}
func (o *operadorPoliticaPrueba) Publicar(_ context.Context, c ports.ComandoPublicarPoliticaOfertas) (ports.VersionPoliticaOfertas, error) {
	o.publicada = true
	return ports.VersionPoliticaOfertas{BolsaRef: c.BolsaRef, Version: 1, HuellaSHA256: strings.Repeat("a", 64), Ejemplo: true, Configurada: true, Politica: &c.Politica, ReciboRef: "recibo:politica-ofertas:" + strings.Repeat("b", 64)}, nil
}

func cuerpoPoliticaPrueba(t *testing.T) string {
	t.Helper()
	p := domain.PoliticaOfertas{Plazo: domain.PlazoPoliticaOfertas{Inicio: "notificacion", Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087"}, Adjudicacion: domain.AdjudicacionPoliticaOfertas{Criterio: "orden_vigente", Elegibilidad: "disposicion_en_plazo"}, NoCubierta: domain.NoCubiertaPoliticaOfertas{Accion: "llamamiento_directo", Condicion: "sin_disposiciones_elegibles"}}
	b, e := json.Marshal(map[string]any{"bolsa_ref": "bolsa:prueba", "version_esperada": 0, "clave_idempotencia": "clave-0001", "politica": p})
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func TestPoliticaOfertasGETyPOSTAutorizados(t *testing.T) {
	o := &operadorPoliticaPrueba{}
	h, e := NuevoHandlerPoliticaOfertas(preparadorPoliticaPrueba{puedePublicar: true}, o)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPoliticaOfertas+"?bolsa_ref=bolsa:prueba", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"configurada":false`) || !strings.Contains(w.Body.String(), `"puede_publicar":true`) || !strings.Contains(w.Body.String(), ports.EsquemaPoliticaOfertas) {
		t.Fatalf("GET %d %s", w.Code, w.Body.String())
	}
	r := httptest.NewRequest(http.MethodPost, RutaPoliticaOfertas, strings.NewReader(cuerpoPoliticaPrueba(t)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || !o.publicada || !strings.Contains(w.Body.String(), `"recibo_ref":"recibo:politica-ofertas:`) {
		t.Fatalf("POST %d %s", w.Code, w.Body.String())
	}
}

func TestPoliticaOfertasGETNoSuponePermisoDeEdicion(t *testing.T) {
	h, e := NuevoHandlerPoliticaOfertas(preparadorPoliticaPrueba{}, &operadorPoliticaPrueba{})
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPoliticaOfertas+"?bolsa_ref=bolsa:prueba", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"puede_publicar":false`) {
		t.Fatalf("GET %d %s", w.Code, w.Body.String())
	}
}

func TestPoliticaOfertasDeniegaAntesDeConsultarOEscribir(t *testing.T) {
	o := &operadorPoliticaPrueba{}
	h, _ := NuevoHandlerPoliticaOfertas(preparadorPoliticaPrueba{denegar: true}, o)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPoliticaOfertas+"?bolsa_ref=bolsa:prueba", nil))
	if w.Code != 403 || o.publicada {
		t.Fatalf("GET denegado %d", w.Code)
	}
	r := httptest.NewRequest(http.MethodPost, RutaPoliticaOfertas, strings.NewReader(cuerpoPoliticaPrueba(t)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 || o.publicada {
		t.Fatalf("POST denegado %d", w.Code)
	}
}

func TestPoliticaOfertasGETDenegadoAunquePublique(t *testing.T) {
	o := &operadorPoliticaPrueba{}
	h, err := NuevoHandlerPoliticaOfertas(preparadorPoliticaPrueba{denegarConsulta: true, puedePublicar: true}, o)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaPoliticaOfertas+"?bolsa_ref=bolsa:prueba", nil))
	if w.Code != http.StatusForbidden || o.consultadas != 0 || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatalf("GET sin permiso propio: estado=%d llamadas=%d cuerpo=%s", w.Code, o.consultadas, w.Body.String())
	}
}
