package httpinterno

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
	core "vec-diputacion-granada/internal/vec/domain"
)

type identidadRelacionesDietasPrueba struct{ llamadas int }

func (i *identidadRelacionesDietasPrueba) ResolverIdentidadRelacionesDietas(context.Context) (core.ContextoActor, personaldomain.FechaCivil, error) {
	i.llamadas++
	return core.ContextoActor{}, "", ErrManejadorRelacionesDietasNoDisponible
}

type identidadSinEmpleadoRelacionesDietasPrueba struct{ actor core.ContextoActor }

func (i identidadSinEmpleadoRelacionesDietasPrueba) ResolverIdentidadRelacionesDietas(context.Context) (core.ContextoActor, personaldomain.FechaCivil, error) {
	return i.actor, personaldomain.FechaCivil("2026-09-24"), nil
}

func TestRelacionesDietasDistingueDenegacionV3DeCaida(t *testing.T) {
	if estadoErrorRelacionesDietas(personalports.ErrRelacionEmpleadoDenegada) != http.StatusForbidden ||
		estadoErrorRelacionesDietas(personalports.ErrRelacionEmpleadoNoDisponible) != http.StatusServiceUnavailable ||
		estadoErrorRelacionesDietas(errors.New("infraestructura")) != http.StatusServiceUnavailable {
		t.Fatal("denegación V3 e indisponibilidad se confundieron")
	}
}

func TestRelacionesDietasAuditaDependenciaSinEmpleadoAcreditado(t *testing.T) {
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	instantanea := core.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []core.VinculoReferenciaContextoActor{}}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaRelacionesDietasPrueba{}
	auditoria := &auditoriaRelacionesDietasPrueba{}
	m, err := NuevoManejadorRelacionesDietas(identidadSinEmpleadoRelacionesDietasPrueba{actor}, consulta, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaRelacionesDietas, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || consulta.llamadas != 0 || auditoria.llamadas != 1 || auditoria.orden.ActorRef != actor.Principal.ID || auditoria.orden.RecursoRef != "" || auditoria.orden.EstadoHTTP != http.StatusServiceUnavailable || auditoria.orden.Validar() != nil {
		t.Fatalf("sin empleado: estado=%d orden=%+v", w.Code, auditoria.orden)
	}
}

type consultaRelacionesDietasPrueba struct{ llamadas int }

func (c *consultaRelacionesDietasPrueba) ConsultarPropiasParaDietas(context.Context, personaldomain.SolicitudConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	c.llamadas++
	return personalports.ResultadoConsultaRelacionPropia{}, ErrManejadorRelacionesDietasNoDisponible
}

type auditoriaRelacionesDietasPrueba struct {
	llamadas int
	orden    personalports.OrdenAuditoriaFronteraAsignacionDietas
	fallar   bool
}

func (a *auditoriaRelacionesDietasPrueba) RegistrarAuditoriaFronteraAsignacionDietas(_ context.Context, orden personalports.OrdenAuditoriaFronteraAsignacionDietas) error {
	a.llamadas++
	a.orden = orden
	if a.fallar {
		return ErrManejadorRelacionesDietasNoDisponible
	}
	return nil
}

func TestRelacionesDietasNoAceptaSelectorDeClienteYAudaDenegacion(t *testing.T) {
	identidad := &identidadRelacionesDietasPrueba{}
	consulta := &consultaRelacionesDietasPrueba{}
	auditoria := &auditoriaRelacionesDietasPrueba{}
	manejador, err := NuevoManejadorRelacionesDietas(identidad, consulta, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaRelacionesDietas+"?relacion_ref=rel_aaaaaaaaaaaaaaaaaaaaaa", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	manejador.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || identidad.llamadas != 0 || consulta.llamadas != 0 || auditoria.llamadas != 1 || auditoria.orden.Ruta != personalports.RutaFronteraRelacionesDietas || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalNoEncontrada || auditoria.orden.EstadoHTTP != http.StatusNotFound || auditoria.orden.Validar() != nil {
		t.Fatalf("selector libre: estado=%d identidad=%d consulta=%d auditoria=%+v", w.Code, identidad.llamadas, consulta.llamadas, auditoria.orden)
	}
	r = httptest.NewRequest(http.MethodGet, RutaRelacionesDietas, nil)
	r.Header.Set("Accept", "application/json")
	r.Header.Add("Authorization", "")
	r.Header.Add("Authorization", "Bearer libre")
	w = httptest.NewRecorder()
	manejador.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || identidad.llamadas != 0 || consulta.llamadas != 0 || auditoria.llamadas != 2 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalPeticion || auditoria.orden.EstadoHTTP != http.StatusBadRequest {
		t.Fatal("segunda cabecera libre atravesó lectura Personal")
	}
	r = httptest.NewRequest(http.MethodGet, RutaRelacionesDietas, nil)
	r.Header.Add("Accept", "application/json")
	r.Header.Add("Accept", "text/html")
	w = httptest.NewRecorder()
	manejador.ServeHTTP(w, r)
	if w.Code != http.StatusNotAcceptable || identidad.llamadas != 0 || consulta.llamadas != 0 || auditoria.llamadas != 3 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalRepresentacion || auditoria.orden.EstadoHTTP != http.StatusNotAcceptable {
		t.Fatal("segunda representación atravesó lectura Personal")
	}
	auditoria.fallar = true
	w = httptest.NewRecorder()
	manejador.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || identidad.llamadas != 0 || consulta.llamadas != 0 {
		t.Fatal("fallo de auditoría no cerró la lectura")
	}
	auditoria.fallar = false
	r = httptest.NewRequest(http.MethodGet, RutaRelacionesDietas, nil)
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	manejador.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || identidad.llamadas != 1 || consulta.llamadas != 0 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalDependencia || auditoria.orden.ActorRef != "" {
		t.Fatal("caída de identidad clasificada como permiso o actor no verificado")
	}
}
