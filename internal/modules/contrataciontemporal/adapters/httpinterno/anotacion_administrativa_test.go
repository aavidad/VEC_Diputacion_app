package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadAnotacionPrueba struct {
	c   ContextoCanalAnotacionAdministrativa
	err error
}

func (a autoridadAnotacionPrueba) ResolverContextoCanalAnotacionAdministrativa(context.Context) (ContextoCanalAnotacionAdministrativa, error) {
	return a.c, a.err
}

type ejecutorAnotacionPrueba struct {
	registros, recuperaciones int
	r                         ports.ReciboAnotacionAdministrativa
	err                       error
	solicitud                 application.SolicitudRegistrarAnotacionAdministrativa
}

func (e *ejecutorAnotacionPrueba) RegistrarAnotacionAdministrativa(_ context.Context, s application.SolicitudRegistrarAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error) {
	e.registros++
	e.solicitud = s
	return e.r, e.err
}
func (e *ejecutorAnotacionPrueba) RecuperarAnotacionAdministrativa(context.Context, string, string, ContextoCanalAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error) {
	e.recuperaciones++
	return e.r, e.err
}
func TestManejadorAnotacionPOSTYRecuperacion(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	e := &ejecutorAnotacionPrueba{r: reciboAnotacionPrueba(c, refA(), 8)}
	h, err := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
	if err != nil {
		t.Fatal(err)
	}
	p := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(`{"expediente_ref":"ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`))
	p.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, p)
	if w.Code != 201 || e.registros != 1 {
		t.Fatalf("POST=%d registros=%d", w.Code, e.registros)
	}
	comprobarReciboAnotacionJSON(t, w, e.r)
	g := httptest.NewRequest(http.MethodGet, RutaRecuperacionAnotacionesAdministrativas+"?expediente_ref="+refA()+"&clave_idempotencia=11111111-2222-4333-8444-555555555555", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, g)
	if w.Code != 200 || e.recuperaciones != 1 {
		t.Fatalf("GET=%d recuperaciones=%d", w.Code, e.recuperaciones)
	}
	comprobarReciboAnotacionJSON(t, w, e.r)
}
func TestManejadorAnotacionDeniegaAntesDeEjecutor(t *testing.T) {
	e := &ejecutorAnotacionPrueba{}
	h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{err: errors.New("denegado")}, e)
	r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || e.registros != 0 {
		t.Fatalf("codigo=%d llamadas=%d", w.Code, e.registros)
	}
}
func reciboAnotacionPrueba(c ContextoCanalAnotacionAdministrativa, expediente string, version uint64) ports.ReciboAnotacionAdministrativa {
	return ports.ReciboAnotacionAdministrativa{Operacion: ports.OperacionRegistrarAnotacionAdministrativa, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: expediente, VersionAnterior: version, VersionResultante: version + 1, FaseResultante: domain.ClaveFase("nombramiento"), EstadoResultante: domain.EstadoEnCurso, SeguimientoOriginal: domain.VinculoSeguimientoOriginal{SeguimientoRef: refB(), VersionSeguimiento: 1, HuellaRaizSeguimientoSHA256: strings.Repeat("a", 64)}, ReciboRef: refB(), AuditoriaRef: refC(), EventoRef: refD(), ActorRef: refB(), RegistradaEn: time.Date(2026, 9, 10, 12, 0, 0, 123456000, time.UTC)}
}

func refA() string { return "ref:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }
func refB() string { return "ref:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" }
func refC() string { return "ref:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" }
func refD() string { return "ref:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" }

func TestManejadorAnotacionRechazaCamposProhibidosYRecuperacionInvalida(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	e := &ejecutorAnotacionPrueba{}
	h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
	r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(`{"expediente_ref":"`+refA()+`","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota","seguimiento_ref":"prohibido"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity || e.registros != 0 {
		t.Fatalf("POST=%d registros=%d", w.Code, e.registros)
	}
	r = httptest.NewRequest(http.MethodGet, RutaRecuperacionAnotacionesAdministrativas+"?expediente_ref="+refA(), nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || e.recuperaciones != 0 {
		t.Fatalf("GET=%d recuperaciones=%d", w.Code, e.recuperaciones)
	}
}

func TestManejadorAnotacionRechazaDuplicadosYReciboNoCotejado(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	for _, tc := range []struct {
		nombre, metodo, ruta, cuerpo string
		estado                       int
	}{
		{"POST duplicado", http.MethodPost, RutaAnotacionesAdministrativas, `{"expediente_ref":"` + refA() + `","expediente_ref":"` + refA() + `","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`, 422},
		{"GET duplicado", http.MethodGet, RutaRecuperacionAnotacionesAdministrativas + "?expediente_ref=" + refA() + "&expediente_ref=" + refA() + "&clave_idempotencia=11111111-2222-4333-8444-555555555555", "", 400},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			e := &ejecutorAnotacionPrueba{}
			h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
			r := httptest.NewRequest(tc.metodo, tc.ruta, strings.NewReader(tc.cuerpo))
			if tc.metodo == http.MethodPost {
				r.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.estado || e.registros+e.recuperaciones != 0 {
				t.Fatalf("estado=%d llamadas=%d", w.Code, e.registros+e.recuperaciones)
			}
		})
	}
	e := &ejecutorAnotacionPrueba{r: ports.ReciboAnotacionAdministrativa{}}
	h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
	r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(`{"expediente_ref":"`+refA()+`","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatalf("recibo no cotejado=%d", w.Code)
	}
}

func comprobarReciboAnotacionJSON(t *testing.T, w *httptest.ResponseRecorder, esperado ports.ReciboAnotacionAdministrativa) {
	t.Helper()
	var envoltorio map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &envoltorio); err != nil || len(envoltorio) != 1 || envoltorio["data"] == nil {
		t.Fatalf("envoltorio de recibo inválido: %v", err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(envoltorio["data"], &campos); err != nil || len(campos) != 13 {
		t.Fatalf("forma pública debe contener13campos: %v", err)
	}
	var r ports.ReciboAnotacionAdministrativa
	if err := json.Unmarshal(envoltorio["data"], &r); err != nil || !reflect.DeepEqual(r, esperado) {
		t.Fatalf("recibo público distinto: %v", err)
	}
	if !r.FaseResultante.Valida() || !r.EstadoResultante.Valido() {
		t.Fatal("fixture positiva sin fase/estado")
	}
}

// Un Reader puede devolver simultáneamente todos los bytes y un fallo de
// transporte. El JSON válido no elimina ese error ni autoriza la operación.
type cuerpoAnotacionConError struct{ contenido []byte }

func (c *cuerpoAnotacionConError) Read(p []byte) (int, error) {
	n := copy(p, c.contenido)
	c.contenido = c.contenido[n:]
	if len(c.contenido) == 0 {
		return n, io.ErrUnexpectedEOF
	}
	return n, nil
}
func (c *cuerpoAnotacionConError) Close() error { return nil }
func TestManejadorAnotacionRechazaJSONCompletoConErrorDeLectura(t *testing.T) {
	c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
	e := &ejecutorAnotacionPrueba{r: reciboAnotacionPrueba(c, refA(), 8)}
	h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
	cuerpo := `{"expediente_ref":"` + refA() + `","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`
	r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(cuerpo))
	r.Body = &cuerpoAnotacionConError{[]byte(cuerpo)}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity || e.registros != 0 || e.recuperaciones != 0 {
		t.Fatalf("error de lectura descartado: estado%d registros%d", w.Code, e.registros)
	}
}
func TestManejadorAnotacionPOSTRechazaConsultaEnURL(t *testing.T) {
	for _, sufijo := range []string{"?version_esperada=8", "?", "?actor_ref=prohibido"} {
		t.Run(sufijo, func(t *testing.T) {
			c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
			e := &ejecutorAnotacionPrueba{r: reciboAnotacionPrueba(c, refA(), 8)}
			h, _ := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
			r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas+sufijo, strings.NewReader(`{"expediente_ref":"`+refA()+`","version_esperada":8,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest || e.registros+e.recuperaciones != 0 {
				t.Fatalf("POST aceptó consulta: %d", w.Code)
			}
		})
	}
}
func TestEntradaAnotacionAcotaVersionAntesDeIncrementar(t *testing.T) {
	for _, v := range []uint64{0, ports.MaximoEnteroSeguroOperacionAnalisis, ^uint64(0)} {
		b := []byte(fmt.Sprintf(`{"expediente_ref":"%s","version_esperada":%d,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`, refA(), v))
		if _, err := decodificarEntradaAnotacionAdministrativa(b); err == nil {
			t.Fatalf("versión no incremental admitida: %d", v)
		}
	}
	b := []byte(fmt.Sprintf(`{"expediente_ref":"%s","version_esperada":%d,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"nota"}`, refA(), ports.MaximoEnteroSeguroOperacionAnalisis-1))
	if _, err := decodificarEntradaAnotacionAdministrativa(b); err != nil {
		t.Fatalf("máxima versión incremental rechazada: %v", err)
	}
}

func TestManejadorAnotacionValidaObservacionesAntesDelEjecutor(t *testing.T) {
	for _, caso := range []struct {
		nombre, observaciones string
		valida                bool
	}{
		{"2000 caracteres ASCII", strings.Repeat("a", 2000), true},
		{"2000 caracteres NFC multibyte", strings.Repeat("á", 2000), true},
		{"NFC con salto y tabulador interiores", "Revisión del expediente\nCentro:\tGranada", true},
		{"2001 caracteres ASCII", strings.Repeat("a", 2001), false},
		{"2001 caracteres NFC multibyte", strings.Repeat("á", 2001), false},
		{"texto descompuesto NFD", "Revisio\u0301n", false},
		{"control nulo", "nota\x00interna", false},
		{"retorno de carro", "nota\rinterior", false},
		{"espacio inicial", " nota", false},
		{"espacio final", "nota ", false},
		{"solo espacios", "   ", false},
		{"texto vacío", "", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := ContextoCanalAnotacionAdministrativa{AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa", SesionRef: "ses_bbbbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccccc", OrganizacionRef: refD()}
			e := &ejecutorAnotacionPrueba{r: reciboAnotacionPrueba(c, refA(), 8)}
			h, err := NuevoManejadorAnotacionAdministrativa(autoridadAnotacionPrueba{c: c}, e)
			if err != nil {
				t.Fatal(err)
			}
			contenido, err := json.Marshal(entradaAnotacionAdministrativaJSON{ExpedienteRef: refA(), VersionEsperada: 8, ClaveIdempotencia: "11111111-2222-4333-8444-555555555555", Observaciones: caso.observaciones})
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaAnotacionesAdministrativas, strings.NewReader(string(contenido)))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if !caso.valida {
				if w.Code != http.StatusUnprocessableEntity || e.registros != 0 || e.recuperaciones != 0 {
					t.Fatalf("observaciones inválidas: HTTP%d registros%d recuperaciones%d", w.Code, e.registros, e.recuperaciones)
				}
				var respuesta envoltorioErrorCobertura
				if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil || respuesta.Error.Codigo != "contenido_no_valido" {
					t.Fatalf("error de contenido no conservado: %v", err)
				}
				return
			}
			if w.Code != http.StatusCreated || e.registros != 1 || e.recuperaciones != 0 || e.solicitud.Observaciones != caso.observaciones {
				t.Fatalf("observaciones válidas alteradas o rechazadas: HTTP%d registros%d recuperaciones%d", w.Code, e.registros, e.recuperaciones)
			}
		})
	}
}
