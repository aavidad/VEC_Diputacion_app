package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type identidadCircuitoPrueba struct {
	llamadas  int
	solicitud dietasports.SolicitudOperacionCircuito
	err       error
}

func (r *identidadCircuitoPrueba) ResolverIdentidadEfectivaCircuito(_ context.Context, s dietasports.SolicitudOperacionCircuito) (dietasports.IdentidadEfectivaCircuito, error) {
	r.llamadas++
	r.solicitud = s
	if r.err != nil {
		return dietasports.IdentidadEfectivaCircuito{}, r.err
	}
	return dietasports.IdentidadEfectivaCircuito{UnidadCompetenciaRef: "unidad:acreditada"}, nil
}

func (r *identidadCircuitoPrueba) EstadoCompetenciasCircuito(context.Context) (dietasports.EstadoCompetenciasCircuito, error) {
	if errors.Is(r.err, dietasports.ErrCompetenciaCircuitoSinFuente) {
		return dietasports.EstadoCompetenciasCircuito{Fuente: dietasports.FuenteCompetenciaSinFuente}, nil
	}
	return dietasports.EstadoCompetenciasCircuito{Fuente: dietasports.FuenteCompetenciaAcreditada, Etapas: []domain.EtapaCircuito{domain.EtapaRevision}}, nil
}

type usoCircuitoPrueba struct {
	decisiones, listas, documentos int
	documento                      dietasports.SolicitudDocumentoCircuito
	decision                       dietasports.SolicitudDecisionCircuito
	consulta                       dietasports.ConsultaBandejaCircuito
	replay                         bool
	devolucion                     *domain.DevolucionComision
}

func (u *usoCircuitoPrueba) Decidir(_ context.Context, _ dietasports.IdentidadEfectivaCircuito, s dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	u.decisiones++
	u.decision = s
	return dietasports.ResultadoCircuitoComision{Comision: dietasports.VistaComisionCircuito{Referencia: s.Referencia, Estado: domain.EstadoPendienteAutorizacion, Version: 2}, Recibo: dietasports.ReciboBorradorComision{Referencia: "recibo:prueba", Version: 2, RegistradoEn: time.Date(2026, 9, 24, 12, 0, 0, 123456000, time.UTC), Repeticion: u.replay}}, nil
}
func (u *usoCircuitoPrueba) ListarPendientes(_ context.Context, _ dietasports.IdentidadEfectivaCircuito, q dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	u.listas++
	u.consulta = q
	return dietasports.PaginaBandejaCircuito{Items: []dietasports.VistaComisionCircuito{}}, nil
}

func (u *usoCircuitoPrueba) ConsultarDocumento(_ context.Context, _ dietasports.IdentidadEfectivaCircuito, s dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error) {
	u.documentos++
	u.documento = s
	return dietasports.DocumentoCircuito{Referencia: s.Referencia, Estado: s.Etapa.EstadoPendiente(), Version: 2, Calculo: json.RawMessage(`{}`), Documento: json.RawMessage(`{"lineas":[]}`), Devolucion: u.devolucion}, nil
}

func peticionCircuito(m http.Handler, metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	var lector io.Reader
	if cuerpo != "" {
		lector = strings.NewReader(cuerpo)
	}
	p := httptest.NewRequest(metodo, ruta, lector)
	p.Header.Set("Accept", "application/json")
	if cuerpo != "" {
		p.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, p)
	return w
}

func TestCircuitoSinFuenteDeCompetenciaLoDiceYNoActua(t *testing.T) {
	r, u := &identidadCircuitoPrueba{err: dietasports.ErrCompetenciaCircuitoSinFuente}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	w := peticionCircuito(m, http.MethodGet, RutaCircuitoCompetencias, "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"fuente":"sin_fuente","etapas":[]}` {
		t.Fatalf("competencias sin fuente: %d %s", w.Code, w.Body.String())
	}
	w = peticionCircuito(m, http.MethodGet, RutaCircuito+"?etapa=revision", "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"items":[],"competencia":"sin_fuente"}` || u.listas != 0 {
		t.Fatalf("bandeja sin fuente: %d %s", w.Code, w.Body.String())
	}
	ref := "dco_" + strings.Repeat("a", 22)
	w = peticionCircuito(m, http.MethodPost, RutaCircuito+"/"+ref+"/decisiones", `{"etapa":"revision","decision":"aprobar","motivo":"","clave_idempotencia":"clave_0123456789abcdef","version_esperada":2}`)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "dietas.error.competencia_sin_fuente") || u.decisiones != 0 {
		t.Fatalf("decisión sin fuente: %d %s", w.Code, w.Body.String())
	}
	w = peticionCircuito(m, http.MethodGet, RutaCircuito+"/"+ref+"?etapa=revision", "")
	if w.Code != http.StatusForbidden || u.documentos != 0 {
		t.Fatalf("documento sin fuente: %d %s", w.Code, w.Body.String())
	}
}

func TestDocumentoCircuitoExigeEtapaYNoAdmiteUnidad(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	ref := "dco_" + strings.Repeat("c", 22)
	for _, consulta := range []string{"", "?etapa=otra", "?etapa=revision&unidad_ref=unidad:libre", "?etapa=revision&etapa=autorizacion"} {
		if w := peticionCircuito(m, http.MethodGet, RutaCircuito+"/"+ref+consulta, ""); w.Code != http.StatusBadRequest {
			t.Fatalf("consulta %q: %d", consulta, w.Code)
		}
	}
	if r.llamadas != 0 {
		t.Fatal("se resolvió identidad para una petición inválida")
	}
	w := peticionCircuito(m, http.MethodGet, RutaCircuito+"/"+ref+"?etapa=autorizacion", "")
	if w.Code != http.StatusOK || u.documento.UnidadRef != "unidad:acreditada" || u.documento.Etapa != domain.EtapaAutorizacion || !strings.Contains(w.Body.String(), `"codigos_ruta":[]`) {
		t.Fatalf("documento: %d %#v %s", w.Code, u.documento, w.Body.String())
	}
	if w := peticionCircuito(m, http.MethodPost, RutaCircuito+"/"+ref, `{}`); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("método: %d", w.Code)
	}
}

func TestCircuitoHTTPDerivaUnidadYRechazaSuplantacion(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	ref := "dco_" + strings.Repeat("a", 22)
	cuerpo := `{"etapa":"revision","decision":"aprobar","motivo":"","clave_idempotencia":"clave_0123456789abcdef","version_esperada":1}`
	enviar := func(body string) *httptest.ResponseRecorder {
		p := httptest.NewRequest(http.MethodPost, RutaCircuito+"/"+ref+"/decisiones", strings.NewReader(body))
		p.Header.Set("Accept", "application/json")
		p.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, p)
		return w
	}
	w := enviar(strings.TrimSuffix(cuerpo, "}") + `,"unidad_ref":"unidad:inventada"}`)
	if w.Code != http.StatusBadRequest || r.llamadas != 0 {
		t.Fatalf("suplantación: %d, %d", w.Code, r.llamadas)
	}
	w = enviar(cuerpo)
	if w.Code != http.StatusCreated || u.decisiones != 1 || r.solicitud.Decision.UnidadRef != "" || u.decision.UnidadRef != "unidad:acreditada" {
		t.Fatalf("unidad frontera: %d, %#v, %#v", w.Code, r.solicitud.Decision, u.decision)
	}
	if !strings.Contains(w.Body.String(), `"registrado_en":"2026-09-24T12:00:00.123456Z"`) {
		t.Fatalf("recibo: %s", w.Body.String())
	}
	u.replay = true
	if w = enviar(cuerpo); w.Code != http.StatusOK {
		t.Fatalf("replay: %d", w.Code)
	}
}

func TestBandejaHTTPFiltraFechasSinUnidadLibre(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	get := func(query string) int {
		p := httptest.NewRequest(http.MethodGet, RutaCircuito+query, nil)
		p.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, p)
		return w.Code
	}
	if got := get("?etapa=revision&fecha_desde=2026-09-01&fecha_hasta=2026-09-30&limit=20"); got != http.StatusOK || u.listas != 1 || u.consulta.UnidadRef != "unidad:acreditada" || u.consulta.FechaDesde != "2026-09-01" {
		t.Fatalf("bandeja: %d %#v", got, u.consulta)
	}
	if got := get("?etapa=revision&unidad_ref=unidad:inventada"); got != http.StatusBadRequest || u.listas != 1 {
		t.Fatalf("unidad libre: %d", got)
	}
	if got := get("?etapa=revision&fecha_desde=2026-10-01&fecha_hasta=2026-09-30"); got != http.StatusBadRequest || u.listas != 1 {
		t.Fatalf("fechas invertidas: %d", got)
	}
}

// Reenvío: el documento que ve quien revisa lleva la devolución anterior con
// etapa, motivo, versión y fecha, sin actor ni persona que la hizo.
func TestDocumentoCircuitoProyectaDevolucionAnteriorSinActor(t *testing.T) {
	u := &usoCircuitoPrueba{devolucion: &domain.DevolucionComision{Etapa: domain.EtapaAutorizacion, Motivo: "Falta el justificante del taxi", Version: 3, DevueltaEn: "2026-09-23T09:00:00.123456Z"}}
	m, err := NuevoManejadorCircuito(&identidadCircuitoPrueba{}, u)
	if err != nil {
		t.Fatal(err)
	}
	w := peticionCircuito(m, http.MethodGet, RutaCircuito+"/dco_"+strings.Repeat("c", 22)+"?etapa=revision", "")
	var cuerpo struct {
		Comision struct {
			Devolucion map[string]any `json:"devolucion"`
		} `json:"comision"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &cuerpo) != nil || len(cuerpo.Comision.Devolucion) != 4 ||
		cuerpo.Comision.Devolucion["motivo"] != "Falta el justificante del taxi" || cuerpo.Comision.Devolucion["etapa"] != "autorizacion" ||
		strings.Contains(w.Body.String(), "act_") || strings.Contains(w.Body.String(), "actor") {
		t.Fatalf("devolución anterior: %d %s", w.Code, w.Body.String())
	}
}

// Go y el navegador recortan conjuntos distintos; la frontera rechaza ambos.
func TestDecisionHTTPRechazaBordesQueRecortaElNavegador(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	for _, motivo := range []string{`\ufeffFalta justificante`, `Falta justificante\ufeff`, `Falta justificante\u0085`} {
		p := httptest.NewRequest(http.MethodPost, RutaCircuito+"/dco_"+strings.Repeat("a", 22)+"/decisiones",
			strings.NewReader(`{"etapa":"revision","decision":"devolver","motivo":"`+motivo+`","clave_idempotencia":"clave_0123456789abcdef","version_esperada":1}`))
		p.Header.Set("Accept", "application/json")
		p.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, p)
		if w.Code != http.StatusBadRequest || r.llamadas != 0 || u.decisiones != 0 {
			t.Fatalf("%s: %d %d %d", motivo, w.Code, r.llamadas, u.decisiones)
		}
	}
}
