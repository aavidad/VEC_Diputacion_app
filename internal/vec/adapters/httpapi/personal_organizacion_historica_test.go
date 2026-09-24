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

type autoridadOrganizacionHistoricaPrueba struct {
	actor             vecdomain.ContextoActor
	organismo, unidad string
	err               error
	llamadas          int
}

func (a *autoridadOrganizacionHistoricaPrueba) ResolverContextoOrganizacionHistorica(context.Context) (vecdomain.ContextoActor, string, string, error) {
	a.llamadas++
	return a.actor, a.organismo, a.unidad, a.err
}

type consultorOrganizacionHistoricaPrueba struct {
	llamadas         int
	solicitud        personaldomain.SolicitudConsultaOrganizacionHistorica
	err              error
	alterarResultado func(*personalports.ResultadoConsultaOrganizacionHistorica)
}

func (c *consultorOrganizacionHistoricaPrueba) Consultar(_ context.Context, s personaldomain.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	c.llamadas++
	c.solicitud = s
	if c.err != nil {
		return personalports.ResultadoConsultaOrganizacionHistorica{}, c.err
	}
	resultado := personalports.ResultadoConsultaOrganizacionHistorica{
		Pagina: personalports.PaginaOrganizacionHistorica{Selector: s.Selector},
		Evidencia: personalports.EvidenciaConsultaOrganizacionHistorica{
			ReciboRef: "recibo_prueba", DecisionRef: "decision_prueba", EfectoRef: "efecto_prueba",
			ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria_prueba",
			ConsultadaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
		},
	}
	if c.alterarResultado != nil {
		c.alterarResultado(&resultado)
	}
	return resultado, nil
}

type auditorOrganizacionHistoricaPrueba struct {
	ordenes []DenegacionOrganizacionHistorica
	err     error
}

func (a *auditorOrganizacionHistoricaPrueba) RegistrarDenegacionOrganizacionHistorica(_ context.Context, o DenegacionOrganizacionHistorica) error {
	a.ordenes = append(a.ordenes, o)
	return a.err
}

const queryOrganizacionHistoricaPrueba = "vigente_en=2026-09-25&conocido_en=2026-09-25T10%3A00%3A00.000000Z"

func actorOrganizacionHistoricaPrueba(t *testing.T) vecdomain.ContextoActor {
	t.Helper()
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + strings.Repeat("a", 22), Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	i := vecdomain.InstantaneaContextoActor{
		VinculoRef: "vca_" + strings.Repeat("b", 22), VinculoVersion: 1,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_" + strings.Repeat("c", 22), PersonaVersion: 1,
		PerfilActivoRef: "prf_" + strings.Repeat("d", 22), PerfilVersion: 1,
		Estado:       vecdomain.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(time.Hour),
	}
	actor, err := vecdomain.NuevoContextoActor(cuenta, i, instante)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func TestOrganizacionHistoricaHTTPDeniegaSinDependencias(t *testing.T) {
	a, c, audit := &autoridadOrganizacionHistoricaPrueba{}, &consultorOrganizacionHistoricaPrueba{}, &auditorOrganizacionHistoricaPrueba{}
	for _, caso := range []struct {
		name  string
		a     AutoridadContextoOrganizacionHistorica
		c     ConsultorOrganizacionHistorica
		audit AuditorDenegacionOrganizacionHistorica
	}{
		{"autoridad", nil, c, audit}, {"consulta", a, nil, audit}, {"auditoria", a, c, nil},
	} {
		t.Run(caso.name, func(t *testing.T) {
			if h, err := NewHandlerOrganizacionHistoricaPersonal(caso.a, caso.c, caso.audit); !errors.Is(err, ErrHandlerOrganizacionHistoricaInvalido) || h != nil {
				t.Fatalf("montaje inesperado: %v", err)
			}
		})
	}
}

func TestOrganizacionHistoricaHTTPConsultaYAmbitoServidor(t *testing.T) {
	a := &autoridadOrganizacionHistoricaPrueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba", unidad: "uni_prueba"}
	c, audit := &consultorOrganizacionHistoricaPrueba{}, &auditorOrganizacionHistoricaPrueba{}
	h, err := NewHandlerOrganizacionHistoricaPersonal(a, c, audit)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal+"?"+queryOrganizacionHistoricaPrueba, nil)
	r.Header.Set("X-VEC-Subject", "persona_falsa")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || c.llamadas != 1 || c.solicitud.Selector.OrganismoRef != "org_prueba" || c.solicitud.Selector.UnidadClave != "uni_prueba" || c.solicitud.Actor.Principal.ID != a.actor.Principal.ID || len(audit.ordenes) != 0 {
		t.Fatalf("respuesta=%d cuerpo=%s solicitud=%+v", w.Code, w.Body.String(), c.solicitud)
	}
	if w.Header().Get("Cache-Control") == "" || w.Header().Get("Set-Cookie") != "" || !strings.Contains(w.Body.String(), `"data"`) || !strings.Contains(w.Body.String(), `"evidencia"`) {
		t.Fatalf("cabeceras o payload: %v %s", w.Header(), w.Body.String())
	}
}

func TestOrganizacionHistoricaHTTPRechazaEntradaAntesDeAutoridad(t *testing.T) {
	for _, q := range []string{
		"", "vigente_en=2026-09-25", queryOrganizacionHistoricaPrueba + "&organismo_ref=otro",
		queryOrganizacionHistoricaPrueba + "&limite=101", queryOrganizacionHistoricaPrueba + "&limite=1&limite=2",
		queryOrganizacionHistoricaPrueba + "&unidad_clave=", queryOrganizacionHistoricaPrueba + "&cursor=%0A",
		queryOrganizacionHistoricaPrueba + "&unidad_clave=../../otra", queryOrganizacionHistoricaPrueba + "&version_rpt_ref=%3Cscript%3E",
		"vigente_en=2026-02-30&conocido_en=2026-09-25T10%3A00%3A00.000000Z",
	} {
		t.Run(q, func(t *testing.T) {
			a, c, audit := &autoridadOrganizacionHistoricaPrueba{}, &consultorOrganizacionHistoricaPrueba{}, &auditorOrganizacionHistoricaPrueba{}
			h, _ := NewHandlerOrganizacionHistoricaPersonal(a, c, audit)
			ruta := RutaOrganizacionHistoricaPersonal
			if q != "" {
				ruta += "?" + q
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
			if w.Code != 400 || a.llamadas != 0 || c.llamadas != 0 {
				t.Fatalf("q=%q respuesta=%d", q, w.Code)
			}
		})
	}
}

func TestOrganizacionHistoricaHTTPAuditaDenegacionesYFallaCerrado(t *testing.T) {
	actor := actorOrganizacionHistoricaPrueba(t)
	for _, caso := range []struct {
		name   string
		err    error
		unidad string
		query  string
		status int
		motivo string
	}{
		{"sin identidad", ErrAutenticacionRutaExactaRequerida, "", queryOrganizacionHistoricaPrueba, 401, "autenticacion_requerida"},
		{"sin permiso", ErrAccesoRutaExactaDenegado, "", queryOrganizacionHistoricaPrueba, 403, "acceso_denegado"},
		{"otra unidad", nil, "uni_prueba", queryOrganizacionHistoricaPrueba + "&unidad_clave=uni_ajena", 403, "acceso_denegado"},
	} {
		t.Run(caso.name, func(t *testing.T) {
			a := &autoridadOrganizacionHistoricaPrueba{actor: actor, organismo: "org_prueba", unidad: caso.unidad, err: caso.err}
			c, audit := &consultorOrganizacionHistoricaPrueba{}, &auditorOrganizacionHistoricaPrueba{}
			h, _ := NewHandlerOrganizacionHistoricaPersonal(a, c, audit)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal+"?"+caso.query, nil))
			if w.Code != caso.status || len(audit.ordenes) != 1 || audit.ordenes[0].Motivo != caso.motivo || c.llamadas != 0 {
				t.Fatalf("respuesta=%d auditoria=%+v", w.Code, audit.ordenes)
			}
		})
	}
	a := &autoridadOrganizacionHistoricaPrueba{err: ErrAutenticacionRutaExactaRequerida}
	c, audit := &consultorOrganizacionHistoricaPrueba{}, &auditorOrganizacionHistoricaPrueba{err: errors.New("sin registro")}
	h, _ := NewHandlerOrganizacionHistoricaPersonal(a, c, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal+"?"+queryOrganizacionHistoricaPrueba, nil))
	if w.Code != 503 {
		t.Fatalf("auditoria fallida debe cerrar: %d", w.Code)
	}
}

func TestOrganizacionHistoricaHTTPClasificaDenegacionDelServicio(t *testing.T) {
	for _, caso := range []struct {
		nombre     string
		err        error
		estado     int
		auditorias int
	}{
		{"concesion denegada", personaldomain.ErrConsultaOrganizacionHistoricaDenegada, 403, 1},
		{"dependencia no disponible", personaldomain.ErrOrganizacionHistoricaNoDisponible, 503, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			a := &autoridadOrganizacionHistoricaPrueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
			c, audit := &consultorOrganizacionHistoricaPrueba{err: caso.err}, &auditorOrganizacionHistoricaPrueba{}
			h, err := NewHandlerOrganizacionHistoricaPersonal(a, c, audit)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal+"?"+queryOrganizacionHistoricaPrueba, nil))
			if w.Code != caso.estado || c.llamadas != 1 || len(audit.ordenes) != caso.auditorias || strings.Contains(w.Body.String(), caso.err.Error()) {
				t.Fatalf("respuesta=%d cuerpo=%s auditorias=%+v", w.Code, w.Body.String(), audit.ordenes)
			}
		})
	}
}

func TestOrganizacionHistoricaHTTPNoPublicaResultadoConOtroAmbito(t *testing.T) {
	a := &autoridadOrganizacionHistoricaPrueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	c := &consultorOrganizacionHistoricaPrueba{alterarResultado: func(r *personalports.ResultadoConsultaOrganizacionHistorica) {
		r.Pagina.Selector.OrganismoRef = "org_ajeno"
	}}
	h, err := NewHandlerOrganizacionHistoricaPersonal(a, c, &auditorOrganizacionHistoricaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal+"?"+queryOrganizacionHistoricaPrueba, nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "org_ajeno") {
		t.Fatalf("resultado ajeno publicado: %d %s", w.Code, w.Body.String())
	}
}
