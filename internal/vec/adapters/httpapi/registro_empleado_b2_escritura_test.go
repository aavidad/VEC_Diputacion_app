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
)

type operadorActosRegistroEmpleadoB2Prueba struct {
	alta     personaldomain.SolicitudAltaEmpleadoB2
	hecho    personaldomain.SolicitudHechoEmpleadoB2
	recibo   personalports.ReciboActoRegistroEmpleadoB2
	acceso   personalports.AccesoActualRegistroEmpleadoB2
	err      error
	llamadas int
}

func (o *operadorActosRegistroEmpleadoB2Prueba) RegistrarEmpleado(_ context.Context, s personaldomain.SolicitudAltaEmpleadoB2) (personalports.ResultadoAltaEmpleadoB2, error) {
	o.llamadas++
	o.alta = s
	return personalports.ResultadoAltaEmpleadoB2{Recibo: o.recibo, AccesoActual: o.acceso}, o.err
}
func (o *operadorActosRegistroEmpleadoB2Prueba) RegistrarHecho(_ context.Context, s personaldomain.SolicitudHechoEmpleadoB2) (personalports.ResultadoHechoEmpleadoB2, error) {
	o.llamadas++
	o.hecho = s
	return personalports.ResultadoHechoEmpleadoB2{Recibo: o.recibo, AccesoActual: o.acceso}, o.err
}

const claveRegistroEmpleadoB2Prueba = "12345678-1234-4234-8234-123456789abc"
const altaJSONRegistroEmpleadoB2Prueba = `{"persona_ref":"per_bbbbbbbbbbbbbbbbbbbbbb","organismo_ref":"org_prueba","unidad_ref":"uni_prueba","regimen":{"ref":"reg_prueba","version":1},"modalidad":{"ref":"mod_prueba","version":1},"vigente_desde":"2026-09-25","acto_ref":"acto_prueba","fuente_ref":"fuente_prueba","fuente_version":1,"fuente_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
const hechoJSONRegistroEmpleadoB2Prueba = `{"tipo":"relacion","empleado_ref":"emp_aaaaaaaaaaaaaaaaaaaaaa","revision_esperada":1,"unidad_ref":"uni_prueba","regimen":{"ref":"reg_prueba","version":1},"modalidad":{"ref":"mod_prueba","version":1},"estado":"vigente","vigente_desde":"2026-09-25","acto_ref":"acto_prueba","fuente_ref":"fuente_prueba","fuente_version":1,"fuente_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`

func peticionPostRegistroEmpleadoB2(ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRegistroEmpleadoB2Prueba)
	return r
}
func reciboRegistroEmpleadoB2Prueba(tipo string) personalports.ReciboActoRegistroEmpleadoB2 {
	r := personalports.ReciboActoRegistroEmpleadoB2{ReciboRef: "perrec_" + strings.Repeat("a", 32), EmpleadoRef: empRefRegistroEmpleadoB2Prueba, RelacionRef: "rel_" + strings.Repeat("b", 22), Tipo: tipo, Version: 1, RegistradoEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), DecisionRef: "decision_1", EfectoRef: "efecto_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit_1"}
	if tipo == "alta" {
		r.ProyeccionRef = "proy_prueba"
	} else {
		r.HechoRef = "hecho_prueba"
	}
	return r
}

func accesoRegistroEmpleadoB2Prueba() personalports.AccesoActualRegistroEmpleadoB2 {
	return personalports.AccesoActualRegistroEmpleadoB2{DecisionRef: "decision_1", EfectoRef: "efecto_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "audit_1", ConsultadaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), EstadoReplay: "registrado"}
}

func TestRegistroEmpleadoB2HTTPPostAltaYReplay(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o := &operadorActosRegistroEmpleadoB2Prueba{recibo: reciboRegistroEmpleadoB2Prueba("alta"), acceso: accesoRegistroEmpleadoB2Prueba()}
	h, err := NewHandlerAltaEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 201 || o.llamadas != 1 || o.alta.Actor.Principal.ID != a.actor.Principal.ID || o.alta.Procedencia.IdempotenciaRef != claveRegistroEmpleadoB2Prueba || !strings.Contains(w.Body.String(), `"recibo"`) || !strings.Contains(w.Body.String(), `"eficacia_administrativa":false`) || !strings.Contains(w.Body.String(), `"firma_oficial":false`) {
		t.Fatalf("alta estado=%d solicitud=%+v cuerpo=%s", w.Code, o.alta, w.Body.String())
	}
	o.acceso.EstadoReplay = "replay"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 200 || o.llamadas != 2 {
		t.Fatalf("replay=%d llamadas=%d", w.Code, o.llamadas)
	}
}

func TestRegistroEmpleadoB2HTTPPostHechoYConflicto(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o := &operadorActosRegistroEmpleadoB2Prueba{recibo: reciboRegistroEmpleadoB2Prueba("relacion"), acceso: accesoRegistroEmpleadoB2Prueba()}
	h, err := NewHandlerHechosEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaHechosEmpleadoB2, hechoJSONRegistroEmpleadoB2Prueba))
	if w.Code != 201 || o.hecho.Tipo != "relacion" || o.hecho.RelacionRef != "" || o.hecho.RevisionEsperada != 1 || o.hecho.OrganismoRef != "org_prueba" || o.hecho.Actor.Principal.ID != a.actor.Principal.ID {
		t.Fatalf("hecho estado=%d solicitud=%+v cuerpo=%s", w.Code, o.hecho, w.Body.String())
	}
	o.err = personaldomain.ErrRegistroEmpleadoB2Conflicto
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaHechosEmpleadoB2, hechoJSONRegistroEmpleadoB2Prueba))
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"conflicto"`) {
		t.Fatalf("conflicto=%d %s", w.Code, w.Body.String())
	}
}

func TestRegistroEmpleadoB2HTTPPostRechazaIDsYClaveAntesDeAutoridad(t *testing.T) {
	for _, caso := range []struct {
		cuerpo    string
		modificar func(*http.Request)
	}{
		{altaJSONRegistroEmpleadoB2Prueba[:len(altaJSONRegistroEmpleadoB2Prueba)-1] + `,"empleado_ref":"emp_inventado"}`, nil},
		{altaJSONRegistroEmpleadoB2Prueba[:len(altaJSONRegistroEmpleadoB2Prueba)-1] + `,"actor_ref":"otro"}`, nil},
		{altaJSONRegistroEmpleadoB2Prueba[:len(altaJSONRegistroEmpleadoB2Prueba)-1] + `,"persona_ref":"per_otra"}`, nil},
		{altaJSONRegistroEmpleadoB2Prueba, func(r *http.Request) { r.Header.Del("Idempotency-Key") }},
		{altaJSONRegistroEmpleadoB2Prueba, func(r *http.Request) { r.Header.Set("Idempotency-Key", "invalida") }},
	} {
		a, o, audit := &autoridadRegistroEmpleadoB2Prueba{}, &operadorActosRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
		h, _ := NewHandlerAltaEmpleadoB2(a, o, audit)
		r := peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, caso.cuerpo)
		if caso.modificar != nil {
			caso.modificar(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 || a.llamadas != 0 || o.llamadas != 0 {
			t.Fatalf("estado=%d autoridad=%d operador=%d cuerpo=%s", w.Code, a.llamadas, o.llamadas, w.Body.String())
		}
	}
}

func TestRegistroEmpleadoB2HTTPPostDeniegaYAudita(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
	}{{ErrAutenticacionRutaExactaRequerida, 401}, {ErrAccesoRutaExactaDenegado, 403}} {
		a := &autoridadRegistroEmpleadoB2Prueba{err: caso.err}
		o, audit := &operadorActosRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
		h, _ := NewHandlerAltaEmpleadoB2(a, o, audit)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
		if w.Code != caso.estado || o.llamadas != 0 || len(audit.ordenes) != 1 {
			t.Fatalf("estado=%d operador=%d auditoria=%+v", w.Code, o.llamadas, audit.ordenes)
		}
	}
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_ajeno"}
	o, audit := &operadorActosRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerAltaEmpleadoB2(a, o, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 403 || o.llamadas != 0 || len(audit.ordenes) != 1 {
		t.Fatalf("ambito=%d operador=%d auditoria=%+v", w.Code, o.llamadas, audit.ordenes)
	}
	audit.err = errors.New("sin auditoria")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 503 {
		t.Fatalf("fallo auditoria=%d", w.Code)
	}
}

func TestRegistroEmpleadoB2HTTPHechoDeniegaOrganismoAusenteYAjeno(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t)}
	o, audit := &operadorActosRegistroEmpleadoB2Prueba{}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerHechosEmpleadoB2(a, o, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaHechosEmpleadoB2, hechoJSONRegistroEmpleadoB2Prueba))
	if w.Code != 403 || o.llamadas != 0 || len(audit.ordenes) != 1 || strings.Contains(w.Body.String(), empRefRegistroEmpleadoB2Prueba) {
		t.Fatalf("sin organismo=%d llamadas=%d auditoria=%+v", w.Code, o.llamadas, audit.ordenes)
	}
	a.organismo = "org_ajeno"
	o.err = personaldomain.ErrRegistroEmpleadoB2Denegado
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaHechosEmpleadoB2, hechoJSONRegistroEmpleadoB2Prueba))
	if w.Code != 403 || o.llamadas != 1 || o.hecho.OrganismoRef != "org_ajeno" || len(audit.ordenes) != 2 || strings.Contains(w.Body.String(), empRefRegistroEmpleadoB2Prueba) {
		t.Fatalf("ajeno=%d orden=%+v auditoria=%+v", w.Code, o.hecho, audit.ordenes)
	}
}

func TestRegistroEmpleadoB2HTTPAltaPersonaObjetivoNoAcreditadaEsOpaca(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o, audit := &operadorActosRegistroEmpleadoB2Prueba{err: personaldomain.ErrRegistroEmpleadoB2Denegado}, &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerAltaEmpleadoB2(a, o, audit)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 403 || o.llamadas != 1 || len(audit.ordenes) != 1 || strings.Contains(w.Body.String(), "per_bbbbbbbbbbbbbbbbbbbbbb") || strings.Contains(w.Body.String(), "B1") {
		t.Fatalf("objetivo no acreditado=%d cuerpo=%s auditoria=%+v", w.Code, w.Body.String(), audit.ordenes)
	}
}

func TestRegistroEmpleadoB2HTTPCatalogosExigenVersionYRechazanTextoLibre(t *testing.T) {
	for _, cuerpo := range []string{
		strings.Replace(altaJSONRegistroEmpleadoB2Prueba, `"regimen":{"ref":"reg_prueba","version":1}`, `"regimen":{"ref":"reg_prueba"}`, 1),
		strings.Replace(altaJSONRegistroEmpleadoB2Prueba, `"modalidad":{"ref":"mod_prueba","version":1}`, `"modalidad":{"ref":"mod_prueba","version":0}`, 1),
		strings.Replace(altaJSONRegistroEmpleadoB2Prueba, `"regimen":{"ref":"reg_prueba","version":1}`, `"regimen":"reg_prueba"`, 1),
		strings.Replace(altaJSONRegistroEmpleadoB2Prueba, `"regimen":{"ref":"reg_prueba","version":1}`, `"regimen":{"ref":"reg_prueba","version":1,"estado":"vigente"}`, 1),
		strings.Replace(altaJSONRegistroEmpleadoB2Prueba, `"regimen":{"ref":"reg_prueba","version":1}`, `"regimen_ref":"reg_prueba"`, 1),
	} {
		a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
		o := &operadorActosRegistroEmpleadoB2Prueba{}
		h, _ := NewHandlerAltaEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, cuerpo))
		if w.Code != 400 || o.llamadas != 0 {
			t.Fatalf("catálogo inválido=%d operador=%d cuerpo=%s", w.Code, o.llamadas, w.Body.String())
		}
	}
}

func TestRegistroEmpleadoB2HTTPHechosCatalogadosExactos(t *testing.T) {
	const prefijo = `{"empleado_ref":"emp_aaaaaaaaaaaaaaaaaaaaaa","relacion_ref":"rel_bbbbbbbbbbbbbbbbbbbbbb","revision_esperada":1,"relacion_version_esperada":1,"vigente_desde":"2026-09-25","acto_ref":"acto_prueba","fuente_ref":"fuente_prueba","fuente_version":1,"fuente_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",`
	for _, caso := range []struct {
		tipo, cuerpo string
		valida       func(personaldomain.SolicitudHechoEmpleadoB2) bool
	}{
		{"ocupacion", prefijo + `"tipo":"ocupacion","plaza_ref":"plaza_prueba","unidad_ref":"uni_prueba","version_plaza_ref":"ver_prueba","modalidad":{"ref":"mod_prueba","version":3},"clase_ocupacion":"titular"}`, func(s personaldomain.SolicitudHechoEmpleadoB2) bool {
			return s.Modalidad.Ref == "mod_prueba" && s.Modalidad.Version == 3 && s.ClaseOcupacion == "titular"
		}},
		{"servicio", prefijo + `"tipo":"servicio","clase_servicio":{"ref":"cls_prueba","version":4},"periodo_desde":"2025-01-01","periodo_hasta":"2025-02-01","dias_reconocidos":31,"estado":"reconocido"}`, func(s personaldomain.SolicitudHechoEmpleadoB2) bool {
			return s.ClaseServicio.Ref == "cls_prueba" && s.ClaseServicio.Version == 4
		}},
		{"situacion", prefijo + `"tipo":"situacion","situacion":{"ref":"sit_prueba","version":2},"estado":"vigente"}`, func(s personaldomain.SolicitudHechoEmpleadoB2) bool {
			return s.Situacion.Ref == "sit_prueba" && s.Situacion.Version == 2
		}},
	} {
		t.Run(caso.tipo, func(t *testing.T) {
			a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
			o := &operadorActosRegistroEmpleadoB2Prueba{err: personaldomain.ErrRegistroEmpleadoB2Conflicto}
			h, _ := NewHandlerHechosEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaHechosEmpleadoB2, caso.cuerpo))
			if w.Code != 409 || o.llamadas != 1 || o.hecho.Tipo != caso.tipo || !caso.valida(o.hecho) {
				t.Fatalf("hecho=%d llamadas=%d orden=%+v", w.Code, o.llamadas, o.hecho)
			}
		})
	}
}

func TestRegistroEmpleadoB2HTTPCatalogoRetiradoDevuelveConflictoNeutro(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o := &operadorActosRegistroEmpleadoB2Prueba{err: personaldomain.ErrRegistroEmpleadoB2Conflicto}
	h, _ := NewHandlerAltaEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPostRegistroEmpleadoB2(RutaAltaEmpleadoB2, altaJSONRegistroEmpleadoB2Prueba))
	if w.Code != 409 || o.llamadas != 1 || !strings.Contains(w.Body.String(), `"codigo":"conflicto"`) || strings.Contains(w.Body.String(), "reg_prueba") {
		t.Fatalf("conflicto catálogo=%d cuerpo=%s", w.Code, w.Body.String())
	}
}
