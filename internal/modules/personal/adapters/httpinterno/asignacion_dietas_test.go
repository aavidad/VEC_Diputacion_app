package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type identidadAsignacionPrueba struct{ llamadas int }

func (x *identidadAsignacionPrueba) ResolverIdentidadAsignacionDietas(context.Context) (core.ContextoActor, error) {
	x.llamadas++
	return core.ContextoActor{}, personalports.ErrAutenticacionAsignacionDietasRequerida
}

type identidadAsignacionDenegadaPrueba struct{}

func (identidadAsignacionDenegadaPrueba) ResolverIdentidadAsignacionDietas(context.Context) (core.ContextoActor, error) {
	return core.ContextoActor{}, personalports.ErrAsignacionDietasDenegada
}

func TestAltaInicialDecodificaSujetoSinAceptarIdentidadLibre(t *testing.T) {
	cuerpo := `{"relacion_ref":"rel_aaaaaaaaaaaaaaaaaaaaaa","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","empleado_ref":"emp_aaaaaaaaaaaaaaaaaaaaaa","fecha_referencia":"2026-09-24","clave_idempotencia":"11111111-1111-4111-8111-111111111111","version_esperada":0,"centro_ref":"centro:uno","unidad_ref":"unidad:uno","administrativo_persona_ref":"per_bbbbbbbbbbbbbbbbbbbbbb","responsable_persona_ref":"per_cccccccccccccccccccccc","grupo_dieta":2,"vigente_desde":"2026-09-24","motivo_revision":"alta inicial autorizada","procedencia_acto_ref":"acto:uno"}`
	var entrada altaInicialAsignacionJSON
	if err := json.Unmarshal([]byte(cuerpo), &entrada); err != nil || entrada.RelacionRef == "" || entrada.PersonaRef == "" || entrada.EmpleadoRef == "" || entrada.UnidadRef == "" {
		t.Fatalf("alta inicial incompleta: err=%v entrada=%+v", err, entrada)
	}
	identidad := &identidadAsignacionPrueba{}
	caso := &casoAsignacionPrueba{}
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(identidad, caso, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaAsignacionesDietas, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || identidad.llamadas != 1 || caso.llamadas != 0 || auditoria.orden.Accion != "registrar_inicial" || auditoria.orden.Ruta != personalports.RutaFronteraAsignacionesDietas {
		t.Fatalf("alta inicial sin identidad: estado=%d identidad=%d caso=%d auditoria=%+v", w.Code, identidad.llamadas, caso.llamadas, auditoria.orden)
	}
}

func TestCorreccionAdministrativaExigeSujetoExplicito(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(identidad, &casoAsignacionPrueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaAsignacionesDietas + "/rel_aaaaaaaaaaaaaaaaaaaaaa"
	base := `{"fecha_referencia":"2026-09-24","clave_idempotencia":"11111111-1111-4111-8111-111111111111","version_esperada":1,"centro_ref":"centro:uno","unidad_ref":"unidad:uno","administrativo_persona_ref":"per_bbbbbbbbbbbbbbbbbbbbbb","responsable_persona_ref":"per_cccccccccccccccccccccc","grupo_dieta":2,"vigente_desde":"2026-09-24","motivo_revision":"correccion autorizada","procedencia_acto_ref":"acto:uno"`
	for _, tc := range []struct {
		nombre, cuerpo string
		estado         int
	}{
		{"sin_sujeto", base + `}`, http.StatusBadRequest},
		{"con_sujeto", base + `,"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","empleado_ref":"emp_aaaaaaaaaaaaaaaaaaaaaa"}`, http.StatusUnauthorized},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, ruta, strings.NewReader(tc.cuerpo))
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Content-Type", "application/json; charset=utf-8")
			w := httptest.NewRecorder()
			m.ServeHTTP(w, r)
			if w.Code != tc.estado {
				t.Fatalf("estado=%d esperado=%d", w.Code, tc.estado)
			}
		})
	}
	if identidad.llamadas != 1 || auditoria.llamadas != 2 || auditoria.orden.Accion != "corregir" {
		t.Fatalf("frontera administrativa: identidad=%d auditoria=%d orden=%+v", identidad.llamadas, auditoria.llamadas, auditoria.orden)
	}
}

type casoAsignacionPrueba struct{ llamadas int }

func (x *casoAsignacionPrueba) Ejecutar(context.Context, personaldomain.SolicitudAsignacionDietas) (personalports.ResultadoAsignacionDietas, error) {
	x.llamadas++
	return personalports.ResultadoAsignacionDietas{}, errors.New("no deberia llegar")
}

type auditoriaAsignacionPrueba struct {
	llamadas int
	orden    personalports.OrdenAuditoriaFronteraAsignacionDietas
	err      error
}

func (x *auditoriaAsignacionPrueba) RegistrarAuditoriaFronteraAsignacionDietas(_ context.Context, orden personalports.OrdenAuditoriaFronteraAsignacionDietas) error {
	x.llamadas++
	x.orden = orden
	return x.err
}

func TestAsignacionRechazaIdentidadDelClienteAntesDeResolver(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	caso := &casoAsignacionPrueba{}
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(identidad, caso, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaAsignacionesDietas + "/rel_aaaaaaaaaaaaaaaaaaaaaa"
	cuerpo := `{"fecha_referencia":"2026-09-24","clave_idempotencia":"11111111-1111-4111-8111-111111111111","version_esperada":1,"centro_ref":"centro:uno","unidad_ref":"unidad:uno","administrativo_persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaa","responsable_persona_ref":"per_bbbbbbbbbbbbbbbbbbbbbb","grupo_dieta":2,"vigente_desde":"2026-09-24","motivo_revision":"correccion autorizada","procedencia_acto_ref":"acto:uno","persona_ref":"per_ffffffffffffffffffffff","empleado_ref":"emp_ffffffffffffffffffffff","actor_ref":"actor:libre"}`
	r := httptest.NewRequest(http.MethodPut, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || identidad.llamadas != 0 || caso.llamadas != 0 || auditoria.llamadas != 1 || auditoria.orden.ActorRef != "" || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalPeticion || auditoria.orden.EstadoHTTP != 400 || auditoria.orden.RecursoRef != "rel_aaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("cuerpo con identidad de cliente: estado=%d identidad=%d caso=%d", w.Code, identidad.llamadas, caso.llamadas)
	}
	r = httptest.NewRequest(http.MethodGet, ruta+"?fecha_referencia=2026-09-24&unidad_ref=unidad:uno", nil)
	r.Header.Set("Accept", "application/json")
	r.Header["X-Vec-Actor"] = []string{"", "actor:ajeno"}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || identidad.llamadas != 0 || caso.llamadas != 0 || auditoria.llamadas != 2 || auditoria.orden.ActorRef != "" || auditoria.orden.EstadoHTTP != 400 || auditoria.orden.RecursoRef != "rel_aaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("cabecera de identidad libre: estado=%d identidad=%d caso=%d", w.Code, identidad.llamadas, caso.llamadas)
	}
}

func TestAsignacionAuditaConflictoConResultadoYRecurso(t *testing.T) {
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(&identidadAsignacionPrueba{}, &casoAsignacionPrueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPut, RutaAsignacionesDietas+"/rel_aaaaaaaaaaaaaaaaaaaaaa", nil)
	actor := &actorAuditoriaAsignacion{ref: "actor:administrativo"}
	if err := m.auditarRechazo(r, http.StatusConflict, actor); err != nil || auditoria.llamadas != 1 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalConflicto || auditoria.orden.EstadoHTTP != 409 || auditoria.orden.RecursoRef != "rel_aaaaaaaaaaaaaaaaaaaaaa" || auditoria.orden.ActorRef != actor.ref {
		t.Fatalf("conflicto auditado: err=%v orden=%+v", err, auditoria.orden)
	}
}

func TestAsignacionGETExigeUnidadExactaYResolucionConfiable(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	caso := &casoAsignacionPrueba{}
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(identidad, caso, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaAsignacionesDietas + "/rel_aaaaaaaaaaaaaaaaaaaaaa"
	r := httptest.NewRequest(http.MethodGet, ruta+"?fecha_referencia=2026-09-24", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || identidad.llamadas != 0 {
		t.Fatalf("consulta sin unidad: estado=%d identidad=%d", w.Code, identidad.llamadas)
	}
	r = httptest.NewRequest(http.MethodGet, ruta+"?fecha_referencia=2026-09-24&unidad_ref=unidad:uno", nil)
	r.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || identidad.llamadas != 1 || caso.llamadas != 0 || auditoria.llamadas != 2 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalAutenticacion {
		t.Fatalf("consulta sin identidad confiable: estado=%d identidad=%d caso=%d", w.Code, identidad.llamadas, caso.llamadas)
	}
}

func TestAsignacionFallaCerradaSiAuditoriaNoRegistraRechazo(t *testing.T) {
	identidad := &identidadAsignacionPrueba{}
	caso := &casoAsignacionPrueba{}
	auditoria := &auditoriaAsignacionPrueba{err: errors.New("sin auditoria")}
	m, err := NuevoManejadorAsignacionDietas(identidad, caso, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaAsignacionesDietas+"/rel_aaaaaaaaaaaaaaaaaaaaaa?fecha_referencia=2026-09-24&unidad_ref=unidad:uno", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || auditoria.llamadas != 1 || caso.llamadas != 0 {
		t.Fatalf("rechazo sin auditoria: estado=%d auditoria=%d caso=%d", w.Code, auditoria.llamadas, caso.llamadas)
	}
}

func TestAsignacionClasificaDenegacionSeparadaDeAutenticacion(t *testing.T) {
	auditoria := &auditoriaAsignacionPrueba{}
	m, err := NuevoManejadorAsignacionDietas(identidadAsignacionDenegadaPrueba{}, &casoAsignacionPrueba{}, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaAsignacionesDietas+"/rel_aaaaaaaaaaaaaaaaaaaaaa?fecha_referencia=2026-09-24&unidad_ref=unidad:uno", nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || auditoria.llamadas != 1 || auditoria.orden.Motivo != personalports.MotivoFronteraPersonalDenegado {
		t.Fatalf("denegacion: estado=%d auditoria=%+v", w.Code, auditoria.orden)
	}
}
