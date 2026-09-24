package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type autoridadRegistroEmpleadoB2Prueba struct {
	actor     vecdomain.ContextoActor
	organismo string
	err       error
	llamadas  int
}

func (a *autoridadRegistroEmpleadoB2Prueba) ResolverContextoRegistroEmpleadoB2(context.Context) (vecdomain.ContextoActor, string, error) {
	a.llamadas++
	return a.actor, a.organismo, a.err
}

type consultorRegistroEmpleadoB2Prueba struct {
	resultadoFicha    personalports.ResultadoFichaEmpleadoB2
	resultadoVacantes personalports.ResultadoVacantesB2
	ficha             personaldomain.SolicitudFichaEmpleadoB2
	vacantes          personaldomain.SolicitudVacantesB2
	err               error
	llamadas          int
}

func (c *consultorRegistroEmpleadoB2Prueba) ConsultarFicha(_ context.Context, s personaldomain.SolicitudFichaEmpleadoB2) (personalports.ResultadoFichaEmpleadoB2, error) {
	c.llamadas++
	c.ficha = s
	return c.resultadoFicha, c.err
}
func (c *consultorRegistroEmpleadoB2Prueba) ConsultarVacantes(_ context.Context, s personaldomain.SolicitudVacantesB2) (personalports.ResultadoVacantesB2, error) {
	c.llamadas++
	c.vacantes = s
	return c.resultadoVacantes, c.err
}

type auditorRegistroEmpleadoB2Prueba struct {
	ordenes []DenegacionRegistroEmpleadoB2
	err     error
}

func (a *auditorRegistroEmpleadoB2Prueba) RegistrarDenegacionRegistroEmpleadoB2(_ context.Context, orden DenegacionRegistroEmpleadoB2) error {
	a.ordenes = append(a.ordenes, orden)
	return a.err
}

const queryRegistroEmpleadoB2Prueba = "vigente_en=2026-09-25&conocido_en=2026-09-25T10%3A00%3A00.000000Z"
const empRefRegistroEmpleadoB2Prueba = "emp_aaaaaaaaaaaaaaaaaaaaaa"

func TestRegistroEmpleadoB2HTTPDeniegaDependenciasAusentes(t *testing.T) {
	a, c, audit := &autoridadRegistroEmpleadoB2Prueba{}, &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	for _, caso := range []struct {
		nombre string
		a      AutoridadContextoRegistroEmpleadoB2
		c      ConsultorRegistroEmpleadoB2
		audit  AuditorDenegacionRegistroEmpleadoB2
	}{
		{"autoridad", nil, c, audit}, {"consulta", a, nil, audit}, {"auditoria", a, c, nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			h, err := NewHandlerFichaEmpleadoB2(caso.a, caso.c, caso.audit)
			if h != nil || !errors.Is(err, ErrHandlerRegistroEmpleadoB2Invalido) {
				t.Fatalf("montaje=%v error=%v", h, err)
			}
		})
	}
}

func TestRegistroEmpleadoB2HTTPRechazaEntradaAntesDeAutoridad(t *testing.T) {
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{PrefijoFichaEmpleadoB2 + empRefRegistroEmpleadoB2Prueba + "?" + queryRegistroEmpleadoB2Prueba + "&persona_ref=otra", 400},
		{PrefijoFichaEmpleadoB2 + empRefRegistroEmpleadoB2Prueba + "?vigente_en=2026-09-25", 400},
		{PrefijoFichaEmpleadoB2 + "../otro?" + queryRegistroEmpleadoB2Prueba, 404},
		{PrefijoFichaEmpleadoB2 + "emp_corta?" + queryRegistroEmpleadoB2Prueba, 404},
		{RutaVacantesEmpleadoB2 + "?" + queryRegistroEmpleadoB2Prueba + "&limite=101", 400},
		{RutaVacantesEmpleadoB2 + "?" + queryRegistroEmpleadoB2Prueba + "&organismo_ref=otro", 400},
	} {
		a, c, audit := &autoridadRegistroEmpleadoB2Prueba{}, &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
		var h http.Handler
		if strings.HasPrefix(caso.ruta, RutaVacantesEmpleadoB2) {
			h, _ = NewHandlerVacantesEmpleadoB2(a, c, audit)
		} else {
			h, _ = NewHandlerFichaEmpleadoB2(a, c, audit)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
		if w.Code != caso.estado || a.llamadas != 0 || c.llamadas != 0 {
			t.Fatalf("ruta=%q estado=%d autoridad=%d consulta=%d", caso.ruta, w.Code, a.llamadas, c.llamadas)
		}
	}
}

func TestRegistroEmpleadoB2HTTPAuditaAutenticacionYPermiso(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
		motivo string
	}{
		{ErrAutenticacionRutaExactaRequerida, 401, "autenticacion_requerida"},
		{ErrAccesoRutaExactaDenegado, 403, "acceso_denegado"},
	} {
		a := &autoridadRegistroEmpleadoB2Prueba{err: caso.err}
		c, audit := &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
		h, _ := NewHandlerFichaEmpleadoB2(a, c, audit)
		r := httptest.NewRequest(http.MethodGet, PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba+"?"+queryRegistroEmpleadoB2Prueba, nil)
		r.Header.Set("X-VEC-Subject", "otro_actor")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || c.llamadas != 0 || len(audit.ordenes) != 1 || audit.ordenes[0].Motivo != caso.motivo || audit.ordenes[0].ActorRef != "" || strings.Contains(audit.ordenes[0].Ruta, empRefRegistroEmpleadoB2Prueba) {
			t.Fatalf("estado=%d auditoria=%+v", w.Code, audit.ordenes)
		}
	}
	a := &autoridadRegistroEmpleadoB2Prueba{err: ErrAccesoRutaExactaDenegado}
	c, audit := &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{err: errors.New("fallo auditoria")}
	h, _ := NewHandlerFichaEmpleadoB2(a, c, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 503 {
		t.Fatalf("denegacion sin auditoria=%d", w.Code)
	}
}

func TestRegistroEmpleadoB2HTTPPropagaSoloContextoServidor(t *testing.T) {
	actor := actorOrganizacionHistoricaPrueba(t)
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actor, organismo: "org_prueba"}
	c, audit := &consultorRegistroEmpleadoB2Prueba{err: personaldomain.ErrRegistroEmpleadoB2NoEncontrado}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerFichaEmpleadoB2(a, c, audit)
	r := httptest.NewRequest(http.MethodGet, PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba+"?"+queryRegistroEmpleadoB2Prueba, nil)
	r.Header.Set("X-VEC-Subject", "otro_actor")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 404 || c.llamadas != 1 || c.ficha.Actor.Principal.ID != actor.Principal.ID || c.ficha.EmpleadoRef != empRefRegistroEmpleadoB2Prueba || c.ficha.OrganismoRef != "org_prueba" {
		t.Fatalf("estado=%d solicitud=%+v", w.Code, c.ficha)
	}
	c.err = personaldomain.ErrRegistroEmpleadoB2Denegado
	h, _ = NewHandlerVacantesEmpleadoB2(a, c, audit)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaVacantesEmpleadoB2+"?"+queryRegistroEmpleadoB2Prueba+"&limite=12", nil))
	if w.Code != 403 || c.vacantes.OrganismoRef != "org_prueba" || c.vacantes.Limite != 12 || len(audit.ordenes) != 1 {
		t.Fatalf("estado=%d solicitud=%+v auditoria=%+v", w.Code, c.vacantes, audit.ordenes)
	}
}

func TestRegistroEmpleadoB2HTTPNoPublicaResultadoSinAcreditar(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	c, audit := &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	for _, caso := range []struct {
		vacantes bool
		ruta     string
	}{
		{false, PrefijoFichaEmpleadoB2 + empRefRegistroEmpleadoB2Prueba + "?" + queryRegistroEmpleadoB2Prueba},
		{true, RutaVacantesEmpleadoB2 + "?" + queryRegistroEmpleadoB2Prueba},
	} {
		var h http.Handler
		if caso.vacantes {
			h, _ = NewHandlerVacantesEmpleadoB2(a, c, audit)
		} else {
			h, _ = NewHandlerFichaEmpleadoB2(a, c, audit)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
		if w.Code != 503 || strings.Contains(w.Body.String(), "ficha") || strings.Contains(w.Body.String(), "pagina") || w.Header().Get("Cache-Control") == "" || w.Header().Get("Set-Cookie") != "" {
			t.Fatalf("resultado sin acreditar: %d %s %v", w.Code, w.Body.String(), w.Header())
		}
	}
}

func TestRegistroEmpleadoB2HTTPPublicaFichaMinimaAcreditada(t *testing.T) {
	actor := actorOrganizacionHistoricaPrueba(t)
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actor, organismo: "org_prueba"}
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	c := &consultorRegistroEmpleadoB2Prueba{resultadoFicha: personalports.ResultadoFichaEmpleadoB2{
		Ficha: personaldomain.FichaEmpleadoB2{
			EmpleadoRef: empRefRegistroEmpleadoB2Prueba, OrganismoRef: "org_prueba", PersonaRef: "per_bbbbbbbbbbbbbbbbbbbbbb",
			Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: instante}, Version: 1,
			Relaciones: []personaldomain.RelacionRegistroEmpleadoB2{}, Ocupaciones: []personaldomain.OcupacionEmpleadoB2{},
			Situaciones: []personaldomain.SituacionEmpleadoB2{}, Servicios: []personaldomain.ServicioReconocidoB2{},
		},
		Evidencia: personalports.EvidenciaRegistroEmpleadoB2{
			ReciboRef: "recibo_1", DecisionRef: "decision_1", EfectoRef: "efecto_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit_1", ConsultadaEn: instante,
		},
	}}
	h, err := NewHandlerFichaEmpleadoB2(a, c, &auditorRegistroEmpleadoB2Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ficha"`) || !strings.Contains(w.Body.String(), `"empleado_ref"`) || !strings.Contains(w.Body.String(), `"organismo_ref":"org_prueba"`) || strings.Contains(w.Body.String(), `"actor_ref"`) {
		t.Fatalf("respuesta=%d cuerpo=%s", w.Code, w.Body.String())
	}
	c.resultadoFicha.Ficha.OrganismoRef = "org_ajeno"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, PrefijoFichaEmpleadoB2+empRefRegistroEmpleadoB2Prueba+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "org_ajeno") {
		t.Fatalf("respuesta de otro organismo=%d %s", w.Code, w.Body.String())
	}
}

func TestRegistroEmpleadoB2HTTPCoberturaNoAcreditadaEsDistintaDeCaida(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	c := &consultorRegistroEmpleadoB2Prueba{err: personaldomain.ErrCoberturaVacantesB2NoAcreditada}
	h, _ := NewHandlerVacantesEmpleadoB2(a, c, &auditorRegistroEmpleadoB2Prueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaVacantesEmpleadoB2+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"codigo":"cobertura_no_acreditada"`) {
		t.Fatalf("cobertura=%d %s", w.Code, w.Body.String())
	}
	c.err = personaldomain.ErrRegistroEmpleadoB2NoDisponible
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaVacantesEmpleadoB2+"?"+queryRegistroEmpleadoB2Prueba, nil))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"codigo":"servicio_no_disponible"`) {
		t.Fatalf("dependencia=%d %s", w.Code, w.Body.String())
	}
}

func TestRegistroEmpleadoB2HTTPFichaDeniegaOrganismoAusenteYAjeno(t *testing.T) {
	actor := actorOrganizacionHistoricaPrueba(t)
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actor}
	c, audit := &consultorRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerFichaEmpleadoB2(a, c, audit)
	ruta := PrefijoFichaEmpleadoB2 + empRefRegistroEmpleadoB2Prueba + "?" + queryRegistroEmpleadoB2Prueba
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != 403 || c.llamadas != 0 || len(audit.ordenes) != 1 || strings.Contains(w.Body.String(), empRefRegistroEmpleadoB2Prueba) {
		t.Fatalf("sin organismo=%d consultas=%d auditoria=%+v", w.Code, c.llamadas, audit.ordenes)
	}
	a.organismo = "org_ajeno"
	c.err = personaldomain.ErrRegistroEmpleadoB2Denegado
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
	if w.Code != 403 || c.llamadas != 1 || c.ficha.OrganismoRef != "org_ajeno" || len(audit.ordenes) != 2 || strings.Contains(w.Body.String(), empRefRegistroEmpleadoB2Prueba) {
		t.Fatalf("ajeno=%d consulta=%+v auditoria=%+v", w.Code, c.ficha, audit.ordenes)
	}
}
