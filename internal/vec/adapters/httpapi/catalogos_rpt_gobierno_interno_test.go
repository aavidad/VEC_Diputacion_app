package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

type autoridadRPTHTTPPrueba struct {
	err      error
	llamadas int
}

func (a *autoridadRPTHTTPPrueba) ResolverCredenciales(context.Context) (application.CredencialesGobiernoCategoriaRPT, error) {
	a.llamadas++
	return application.CredencialesGobiernoCategoriaRPT{}, a.err
}

type operadorRPTHTTPPrueba struct {
	llamadas int
	accion   string
	recibo   string
	revision int64
	err      error
}

func (o *operadorRPTHTTPPrueba) salida(accion, recibo string, rev int64) (ports.ResultadoGobiernoCategoriaRPT, error) {
	o.llamadas++
	o.accion = accion
	o.recibo = recibo
	o.revision = rev
	return ports.ResultadoGobiernoCategoriaRPT{PropuestaRef: "propuesta:ejemplo", ReciboRef: recibo, Evidencia: ports.EvidenciaGobiernoCategoriaRPT{DecisionRef: "decision:ejemplo", AuditoriaRef: "auditoria:ejemplo", ConsumoHuellaSHA256: strings.Repeat("a", 64)}}, o.err
}
func (o *operadorRPTHTTPPrueba) Proponer(_ context.Context, p application.OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return o.salida("proponer", p.Borrador.ReciboRef, 0)
}
func (o *operadorRPTHTTPPrueba) Aprobar(_ context.Context, p application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return o.salida("aprobar", p.Material.ReciboRef, p.Material.RevisionEsperada)
}
func (o *operadorRPTHTTPPrueba) Confirmar(_ context.Context, p application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return o.salida("confirmar", p.Material.ReciboRef, p.Material.RevisionEsperada)
}

type auditorRPTHTTPPrueba struct {
	llamadas int
	err      error
	ultimo   RechazoGobiernoCategoriaRPTInterno
}

func (a *auditorRPTHTTPPrueba) RegistrarRechazoGobiernoCategoriaRPT(_ context.Context, r RechazoGobiernoCategoriaRPTInterno) error {
	a.llamadas++
	a.ultimo = r
	return a.err
}

const claveRPTHTTPPrueba = "123e4567-e89b-42d3-a456-426614174000"

func peticionRPTHTTPPrueba(ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRPTHTTPPrueba)
	return r
}
func TestGobiernoCategoriaRPTInternoAccionesYRecibo(t *testing.T) {
	avance := `{"propuesta_ref":"propuesta:ejemplo","catalogo_id":"catalogo.ejemplo","modulo_id":"bolsa","huella_sha256":"` + strings.Repeat("a", 64) + `","revision_esperada":`
	casos := []struct {
		ruta, cuerpo, accion string
		rev                  int64
	}{
		{RutaProponerGobiernoCategoriaRPT, `{"propuesta_ref":"propuesta:ejemplo","accion":"publicar","catalogo_id":"catalogo.ejemplo","modulo_id":"bolsa","version":1,"documento_canonico":"{}","preimagenes_control":{},"fuente_ref":"fuente:ejemplo"}`, "proponer", 0},
		{RutaAprobarGobiernoCategoriaRPT, avance + `1}`, "aprobar", 1},
		{RutaConfirmarGobiernoCategoriaRPT, avance + `2}`, "confirmar", 2},
	}
	for _, tc := range casos {
		t.Run(tc.accion, func(t *testing.T) {
			a := &autoridadRPTHTTPPrueba{}
			o := &operadorRPTHTTPPrueba{}
			audit := &auditorRPTHTTPPrueba{}
			h, err := NuevoHandlerGobiernoCategoriaRPTInterno(a, o, audit)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionRPTHTTPPrueba(tc.ruta, tc.cuerpo))
			if w.Code != http.StatusOK || o.accion != tc.accion || o.revision != tc.rev || o.recibo != reciboGobiernoCategoriaRPTHTTP(tc.ruta, claveRPTHTTPPrueba) || audit.llamadas != 0 {
				t.Fatalf("estado=%d operador=%+v auditor=%+v", w.Code, o, audit)
			}
			if !strings.Contains(w.Body.String(), `"decision_ref":"decision:ejemplo"`) {
				t.Fatalf("falta evidencia: %s", w.Body.String())
			}
			w2 := httptest.NewRecorder()
			h.ServeHTTP(w2, peticionRPTHTTPPrueba(tc.ruta, tc.cuerpo))
			if w2.Code != http.StatusOK || o.recibo != reciboGobiernoCategoriaRPTHTTP(tc.ruta, claveRPTHTTPPrueba) {
				t.Fatal("recibo no estable")
			}
		})
	}
}
func TestGobiernoCategoriaRPTInternoRechazos(t *testing.T) {
	base := `{"propuesta_ref":"propuesta:ejemplo","catalogo_id":"catalogo.ejemplo","modulo_id":"bolsa","huella_sha256":"` + strings.Repeat("a", 64) + `","revision_esperada":1}`
	casos := []struct {
		nombre, ruta, cuerpo string
		mod                  func(*http.Request)
		sesion               error
		servicio             error
		estado               int
		audita               bool
	}{
		{nombre: "sin sesion", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, sesion: ErrSesionGobiernoCategoriaRPTRequerida, estado: 401, audita: true},
		{nombre: "sesion denegada", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, sesion: ErrSesionGobiernoCategoriaRPTDenegada, estado: 403, audita: true},
		{nombre: "actor en JSON", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: strings.Replace(base, `"propuesta_ref"`, `"actor":"otro","propuesta_ref"`, 1), estado: 400, audita: true},
		{nombre: "clave duplicada", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: strings.Replace(base, `"revision_esperada":1`, `"revision_esperada":1,"revision_esperada":1`, 1), estado: 400, audita: true},
		{nombre: "revision incorrecta", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: strings.Replace(base, `"revision_esperada":1`, `"revision_esperada":2`, 1), estado: 400, audita: true},
		{nombre: "header duplicada", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, mod: func(r *http.Request) { r.Header.Add("Idempotency-Key", claveRPTHTTPPrueba) }, estado: 400, audita: true},
		{nombre: "cookie", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, mod: func(r *http.Request) { r.Header.Set("Cookie", "session=evil") }, estado: 400, audita: true},
		{nombre: "conflicto", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, servicio: ports.ErrGobiernoCategoriaRPTConflicto, estado: 409, audita: true},
		{nombre: "error privado", ruta: RutaAprobarGobiernoCategoriaRPT, cuerpo: base, servicio: errors.New("dsn privado"), estado: 503, audita: true},
		{nombre: "query", ruta: RutaAprobarGobiernoCategoriaRPT + "?actor=otro", cuerpo: base, estado: 404},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			a := &autoridadRPTHTTPPrueba{err: tc.sesion}
			o := &operadorRPTHTTPPrueba{err: tc.servicio}
			audit := &auditorRPTHTTPPrueba{}
			h, err := NuevoHandlerGobiernoCategoriaRPTInterno(a, o, audit)
			if err != nil {
				t.Fatal(err)
			}
			r := peticionRPTHTTPPrueba(tc.ruta, tc.cuerpo)
			if tc.mod != nil {
				tc.mod(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.estado {
				t.Fatalf("estado %d, esperado %d: %s", w.Code, tc.estado, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "dsn privado") {
				t.Fatal("filtracion")
			}
			if tc.audita && (audit.llamadas != 1 || audit.ultimo.CorrelacionRef == "") {
				t.Fatalf("auditoria %+v", audit)
			}
			if !tc.audita && audit.llamadas != 0 {
				t.Fatal("auditoria inesperada")
			}
		})
	}
}
func TestGobiernoCategoriaRPTInternoAuditoriaFallaCerrada(t *testing.T) {
	a := &autoridadRPTHTTPPrueba{err: ErrSesionGobiernoCategoriaRPTRequerida}
	o := &operadorRPTHTTPPrueba{}
	audit := &auditorRPTHTTPPrueba{err: errors.New("falla")}
	h, _ := NuevoHandlerGobiernoCategoriaRPTInterno(a, o, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRPTHTTPPrueba(RutaAprobarGobiernoCategoriaRPT, `{}`))
	if w.Code != 503 || o.llamadas != 0 {
		t.Fatalf("estado %d", w.Code)
	}
}
