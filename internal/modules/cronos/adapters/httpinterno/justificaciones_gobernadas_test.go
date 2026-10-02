package httpinterno

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type resolverJustificacionPrueba struct {
	llamadas int
	err      error
}

func (r *resolverJustificacionPrueba) ResolverOrdenJustificacion(*http.Request) (ports.OrdenJustificacion, error) {
	r.llamadas++
	return ports.OrdenJustificacion{}, r.err
}

type casoJustificacionPrueba struct {
	anexos           int
	consulta         int
	revisiones       int
	pendiente        bool
	sinDocumento     bool
	errAnexo         error
	replay           bool
	preparacion      ports.PreparacionJustificacion
	revisionRecibida ports.PeticionRevisionJustificacion
}

func (c *casoJustificacionPrueba) Consultar(context.Context, ports.OrdenJustificacion, string) (ports.PreparacionJustificacion, error) {
	c.consulta++
	return c.preparacion, nil
}

func (c *casoJustificacionPrueba) Anexar(context.Context, ports.OrdenJustificacion, ports.PeticionAnexoJustificacion) (ports.ResultadoAnexoJustificacion, error) {
	c.anexos++
	documento := &domain.DocumentoJustificacion{ID: "ref:" + strings.Repeat("a", 64), Version: 1,
		SHA256: strings.Repeat("b", 64), CustodioID: "custodia", CustodiaRef: "custodia:origen-1"}
	if c.sinDocumento {
		documento = nil
	}
	return ports.ResultadoAnexoJustificacion{Documento: documento, EnlacePendiente: c.pendiente}, c.errAnexo
}

func (c *casoJustificacionPrueba) Revisar(_ context.Context, _ ports.OrdenJustificacion, p ports.PeticionRevisionJustificacion) (ports.ReciboJustificacion, error) {
	c.revisiones++
	c.revisionRecibida = p
	return ports.ReciboJustificacion{ReciboRef: "recibo:cronos:justificacion:abcdefgh", FechaUTC: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		Justificacion: domain.Justificacion{Version: 2, Estado: domain.JustificacionAceptada}, Replay: c.replay}, nil
}

func TestJustificacionHTTPRechazaEntradaAntesDeResolver(t *testing.T) {
	resolutor := &resolverJustificacionPrueba{}
	caso := &casoJustificacionPrueba{}
	m, err := NuevoManejadorJustificaciones(caso, resolutor)
	if err != nil {
		t.Fatal(err)
	}
	base := `{"solicitud_ref":"permiso:cronos:solicitud:abcdefgh","clave_operacion":"clave_1","version_esperada":0,"documento":{"id":"ref:` + strings.Repeat("a", 64) + `","version":1,"sha256":"` + strings.Repeat("b", 64) + `","custodio_id":"custodia","custodia_ref":"custodia:origen-1"}}`
	casos := []struct {
		nombre, ruta, cuerpo string
		metodo               string
		estado               int
	}{
		{"duplicada_anidada", RutaAnexarJustificacion, strings.Replace(base, `"version":1`, `"version":1,"version":2`, 1), http.MethodPost, http.StatusBadRequest},
		{"ambigua_anidada", RutaAnexarJustificacion, strings.Replace(base, `"version":1`, `"version":1,"Version":2`, 1), http.MethodPost, http.StatusBadRequest},
		{"ambigua_raiz", RutaAnexarJustificacion, strings.Replace(base, `"solicitud_ref":`, `"Solicitud_ref":"otra","solicitud_ref":`, 1), http.MethodPost, http.StatusBadRequest},
		{"actor_libre", RutaAnexarJustificacion, strings.Replace(base, `"solicitud_ref":`, `"actor_ref":"otro","solicitud_ref":`, 1), http.MethodPost, http.StatusBadRequest},
		{"vinculo_libre", RutaRevisarJustificacion, `{"solicitud_ref":"permiso:cronos:solicitud:abcdefgh","clave_operacion":"clave_1","version_esperada":1,"decision":"aceptada","motivo_ref":"motivo:uno","vinculo":{}}`, http.MethodPost, http.StatusBadRequest},
		{"consulta_vacia", RutaConsultarJustificacion, "", http.MethodGet, http.StatusBadRequest},
		{"consulta_cruzada", RutaConsultarJustificacion + `?solicitud_ref=permiso:cronos:solicitud:abcdefgh&empleado_ref=emp_2`, "", http.MethodGet, http.StatusBadRequest},
		{"metodo", RutaAnexarJustificacion, "", http.MethodGet, http.StatusMethodNotAllowed},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			req := httptest.NewRequest(tc.metodo, tc.ruta, strings.NewReader(tc.cuerpo))
			if tc.metodo == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			m.ServeHTTP(w, req)
			if w.Code != tc.estado || resolutor.llamadas != 0 || caso.anexos != 0 || caso.consulta != 0 || caso.revisiones != 0 {
				t.Fatalf("estado=%d resolutor=%d caso=%+v", w.Code, resolutor.llamadas, caso)
			}
		})
	}
}

func TestJustificacionHTTPConservaConstanciaEnlacePendiente(t *testing.T) {
	resolutor := &resolverJustificacionPrueba{}
	caso := &casoJustificacionPrueba{pendiente: true, errAnexo: fmt.Errorf("%w: fallo Cronos", ports.ErrEnlaceJustificacionPendiente)}
	m, err := NuevoManejadorJustificaciones(caso, resolutor)
	if err != nil {
		t.Fatal(err)
	}
	contenido := `{"solicitud_ref":"permiso:cronos:solicitud:abcdefgh","clave_operacion":"clave_1","version_esperada":0,"documento":{"id":"ref:` + strings.Repeat("a", 64) + `","version":1,"sha256":"` + strings.Repeat("b", 64) + `","custodio_id":"custodia","custodia_ref":"custodia:origen-1"}}`
	req := httptest.NewRequest(http.MethodPost, RutaAnexarJustificacion, strings.NewReader(contenido))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"enlace_pendiente":true`) || caso.anexos != 1 || resolutor.llamadas != 1 {
		t.Fatalf("estado=%d cuerpo=%s caso=%+v resolutor=%d", w.Code, w.Body.String(), caso, resolutor.llamadas)
	}
	if !strings.Contains(w.Body.String(), `"documento":{"id":`) || strings.Contains(w.Body.String(), `"recibo_cronos":`) {
		t.Fatalf("recibo Cronos presentado sin enlace confirmado: %s", w.Body.String())
	}
}

func TestJustificacionHTTPNoAtribuyeConstanciaAusente(t *testing.T) {
	resolutor := &resolverJustificacionPrueba{}
	caso := &casoJustificacionPrueba{pendiente: true, sinDocumento: true, errAnexo: ports.ErrEnlaceJustificacionPendiente}
	m, _ := NuevoManejadorJustificaciones(caso, resolutor)
	contenido := `{"solicitud_ref":"permiso:cronos:solicitud:abcdefgh","clave_operacion":"clave_1","version_esperada":0,"documento":{"id":"ref:` + strings.Repeat("a", 64) + `","version":1,"sha256":"` + strings.Repeat("b", 64) + `","custodio_id":"custodia","custodia_ref":"custodia:origen-1"}}`
	req := httptest.NewRequest(http.MethodPost, RutaAnexarJustificacion, strings.NewReader(contenido))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), `"documento":`) {
		t.Fatalf("constancia ausente presentada: estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
}

func TestJustificacionHTTPResolverFallaCerrado(t *testing.T) {
	resolutor := &resolverJustificacionPrueba{err: errors.New("fallo interno")}
	caso := &casoJustificacionPrueba{}
	m, _ := NuevoManejadorJustificaciones(caso, resolutor)
	ref := "permiso:cronos:solicitud:abcdefgh"
	req := httptest.NewRequest(http.MethodGet, RutaConsultarJustificacion+"?solicitud_ref="+ref, nil)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable || caso.consulta != 0 {
		t.Fatalf("estado=%d caso=%+v", w.Code, caso)
	}
}

func TestJustificacionHTTPRevisionUsaVinculoDeLecturaAutorizada(t *testing.T) {
	ref := "permiso:cronos:solicitud:abcdefgh"
	vinculo := domain.VinculoJustificacion{SolicitudRef: ref, EmpleadoRef: "emp_autorizado",
		Documento: domain.DocumentoJustificacion{ID: "ref:" + strings.Repeat("a", 64), Version: 1}}
	resolutor := &resolverJustificacionPrueba{}
	caso := &casoJustificacionPrueba{preparacion: ports.PreparacionJustificacion{
		Actual: &domain.Justificacion{Version: 1, Estado: domain.JustificacionPendiente, Vinculo: vinculo}}}
	m, _ := NuevoManejadorJustificaciones(caso, resolutor)
	cuerpo := `{"solicitud_ref":"` + ref + `","clave_operacion":"clave_1","version_esperada":1,"decision":"aceptada","motivo_ref":"motivo:uno"}`
	req := httptest.NewRequest(http.MethodPost, RutaRevisarJustificacion, strings.NewReader(cuerpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || caso.consulta != 1 || caso.revisiones != 1 || caso.revisionRecibida.Vinculo != vinculo {
		t.Fatalf("estado=%d consulta=%d revision=%d vinculo=%+v", w.Code, caso.consulta, caso.revisiones, caso.revisionRecibida.Vinculo)
	}
	if strings.Contains(w.Body.String(), "emp_autorizado") || strings.Contains(w.Body.String(), vinculo.Documento.ID) {
		t.Fatalf("respuesta de revisión expone vínculo interno: %s", w.Body.String())
	}
}

func TestJustificacionHTTPReplayReautorizaLecturaSinExigirVersionActual(t *testing.T) {
	ref := "permiso:cronos:solicitud:abcdefgh"
	resolutor := &resolverJustificacionPrueba{}
	caso := &casoJustificacionPrueba{replay: true, preparacion: ports.PreparacionJustificacion{
		Actual: &domain.Justificacion{Version: 2, Vinculo: domain.VinculoJustificacion{SolicitudRef: ref}}}}
	m, _ := NuevoManejadorJustificaciones(caso, resolutor)
	cuerpo := `{"solicitud_ref":"` + ref + `","clave_operacion":"clave_1","version_esperada":1,"decision":"aceptada","motivo_ref":"motivo:uno"}`
	req := httptest.NewRequest(http.MethodPost, RutaRevisarJustificacion, strings.NewReader(cuerpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	if w.Code != http.StatusOK || caso.consulta != 1 || caso.revisiones != 1 || caso.revisionRecibida.VersionEsperada != 1 {
		t.Fatalf("replay: estado=%d consulta=%d revision=%d", w.Code, caso.consulta, caso.revisiones)
	}
}

var _ CasoUsoJustificacionHTTP = (*casoJustificacionPrueba)(nil)
var _ ResolverOrdenJustificacion = (*resolverJustificacionPrueba)(nil)
