package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type actorFichaPropiaHTTP struct {
	actor core.ContextoActor
	err   error
}

func (a actorFichaPropiaHTTP) ResolverActorFichaPropia(context.Context) (core.ContextoActor, error) {
	return a.actor, a.err
}

type consultaFichaPropiaHTTP struct {
	err       error
	solicitud personaldomain.SolicitudFichaPropia
	llamadas  int
}

func (c *consultaFichaPropiaHTTP) Consultar(_ context.Context, s personaldomain.SolicitudFichaPropia) (personalports.ResultadoFichaPropia, error) {
	c.llamadas++
	c.solicitud = s
	if c.err != nil {
		return personalports.ResultadoFichaPropia{}, c.err
	}
	return personalports.ResultadoFichaPropia{
		Ficha:     personaldomain.FichaPropia{Corte: s.Corte, Relaciones: []personaldomain.RelacionFichaPropia{{Inicio: "2026-01-01", Estado: "vigente", Regimen: "Laboral fijo"}}, Servicios: []personaldomain.ServicioFichaPropia{}},
		Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: "fichapropia:abc", DecisionRef: "dec_x", EfectoRef: "emp_" + strings.Repeat("a", 24), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:x", ConsultadaEn: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)},
	}, nil
}

type registroFichaPropiaHTTP struct {
	denegaciones []personalports.DenegacionFichaPropia
	err          error
}

func (r *registroFichaPropiaHTTP) RegistrarDenegacionFichaPropia(_ context.Context, d personalports.DenegacionFichaPropia) error {
	r.denegaciones = append(r.denegaciones, d)
	return r.err
}

func actorFichaPropiaPruebaHTTP(t *testing.T) core.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func manejadorFichaPropiaPrueba(t *testing.T, consulta *consultaFichaPropiaHTTP, registro *registroFichaPropiaHTTP) *ManejadorFichaPropia {
	t.Helper()
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// 23:30 UTC del 24 es ya día 25 en Madrid.
	ahora := func() time.Time { return time.Date(2026, 9, 24, 23, 30, 0, 500, time.UTC) }
	m, err := NuevoManejadorFichaPropia(actorFichaPropiaHTTP{actor: actorFichaPropiaPruebaHTTP(t)}, consulta, registro, ahora, madrid)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFichaPropiaHTTPSirveSoloDatosVisibles(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{}
	m := manejadorFichaPropiaPrueba(t, consulta, &registroFichaPropiaHTTP{})
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("respuesta %d %v", w.Code, w.Header())
	}
	cuerpo := w.Body.String()
	for _, prohibido := range []string{"emp_", "per_", "dec_x", "auditoria", "consumo_huella"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Fatalf("la respuesta expone %q: %s", prohibido, cuerpo)
		}
	}
	var sobre struct {
		Data struct {
			Ficha        personaldomain.FichaPropia `json:"ficha"`
			ReciboRef    string                     `json:"recibo_ref"`
			ConsultadaEn string                     `json:"consultada_en"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil || sobre.Data.ReciboRef != "fichapropia:abc" || sobre.Data.ConsultadaEn != "2026-09-25T08:00:00.000000Z" || sobre.Data.Ficha.Relaciones[0].Regimen != "Laboral fijo" {
		t.Fatalf("sobre inesperado: %v %s", err, cuerpo)
	}
	if consulta.solicitud.Corte.VigenteEn != "2026-09-25" || !consulta.solicitud.Corte.ConocidoEn.Equal(time.Date(2026, 9, 24, 23, 29, 59, 0, time.UTC)) {
		t.Fatalf("corte inesperado: %+v", consulta.solicitud.Corte)
	}
}

func TestFichaPropiaHTTPDenegacionesAuditadas(t *testing.T) {
	casos := map[string]struct {
		peticion func() *http.Request
		err      error
		estado   int
		codigo   string
		consulta bool
	}{
		"metodo": {func() *http.Request { return httptest.NewRequest(http.MethodPost, RutaFichaPropia, nil) }, nil, 405, "metodo_no_permitido", false},
		"parametros": {func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaFichaPropia+"?empleado=emp_x", nil)
		}, nil, 404, "no_encontrada", false},
		"cookie": {func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil)
			r.Header.Set("Cookie", "a=b")
			return r
		}, nil, 400, "peticion_invalida", false},
		"sin_empleado": {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaSinEmpleado, 403, "sin_empleado", true},
		"ambiguo":      {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaAmbigua, 403, "empleado_ambiguo", true},
		"denegado":     {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, personaldomain.ErrFichaPropiaDenegada, 403, "acceso_denegado", true},
		"caido":        {func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil) }, errors.New("detalle"), 503, "no_disponible", true},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			consulta := &consultaFichaPropiaHTTP{err: caso.err}
			registro := &registroFichaPropiaHTTP{}
			w := httptest.NewRecorder()
			manejadorFichaPropiaPrueba(t, consulta, registro).ServeHTTP(w, caso.peticion())
			if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"error":"`+caso.codigo+`"`) || len(registro.denegaciones) != 1 || registro.denegaciones[0].EstadoHTTP != caso.estado || (consulta.llamadas == 1) != caso.consulta {
				t.Fatalf("estado %d cuerpo %s denegaciones %+v", w.Code, w.Body.String(), registro.denegaciones)
			}
			if caso.consulta && registro.denegaciones[0].ActorRef == "" {
				t.Fatal("denegación con identidad acreditada sin actor")
			}
		})
	}
}

func TestFichaPropiaHTTPSinAuditoriaNoRevelaMotivo(t *testing.T) {
	consulta := &consultaFichaPropiaHTTP{err: personaldomain.ErrFichaPropiaDenegada}
	w := httptest.NewRecorder()
	manejadorFichaPropiaPrueba(t, consulta, &registroFichaPropiaHTTP{err: errors.New("caída")}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaFichaPropia, nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "acceso_denegado") {
		t.Fatalf("sin auditoría se reveló el motivo: %d %s", w.Code, w.Body.String())
	}
}
