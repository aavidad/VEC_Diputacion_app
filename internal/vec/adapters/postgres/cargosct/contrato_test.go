package cargosct

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
	core "vec-diputacion-granada/internal/vec/domain"
)

func planPrueba(t *testing.T) Plan {
	t.Helper()
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	a := core.AsignacionPerfil{AsignacionID: "asignacion_sintetica", Version: 1, PerfilActivoRef: "prf_sintetico", PrincipalID: "per_sintetica", VersionRolRef: "rol:ct_cargo_jefatura_servicio_rrhh:v1", Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion_sintetica"}}, {Clave: "unidad_ref", Valores: []string{"unidad_sintetica"}}}, VigenteDesde: now, VigenteHasta: now.Add(24 * time.Hour), EmitidaPor: "per_operador_sintetico", EmitidaEn: now}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	return Plan{Version: 1, Clave: strings.Repeat("a", 32), RolID: "ct_cargo_jefatura_servicio_rrhh", VersionRolRef: a.VersionRolRef, VersionRolSHA256: strings.Repeat("b", 64), ControlRolSHA256: strings.Repeat("c", 64), PreimagenSHA256: strings.Repeat("d", 64), RegistroDestinoRef: "rca_sintetico", DestinoSHA256: strings.Repeat("e", 64), OrganizacionRef: "organizacion_sintetica", UnidadRef: "unidad_sintetica", AsignacionCanonica: b, CaducaEn: now.Add(time.Hour)}
}
func TestPlanLigadoATodosLosCampos(t *testing.T) {
	p := planPrueba(t)
	h, err := p.Huella()
	if err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*Plan){func(x *Plan) { x.Clave = strings.Repeat("f", 32) }, func(x *Plan) { x.PreimagenSHA256 = strings.Repeat("f", 64) }, func(x *Plan) { x.DestinoSHA256 = strings.Repeat("f", 64) }, func(x *Plan) { x.CaducaEn = x.CaducaEn.Add(time.Second) }} {
		q := p
		mutar(&q)
		got, err := q.Huella()
		if err != nil || got == h {
			t.Fatal("cambio no ligado a aprobación")
		}
	}
}
func TestDestinoDebeCubrirAmbitoExacto(t *testing.T) {
	p := planPrueba(t)
	p.UnidadRef = "unidad_otra"
	if _, err := p.Canonica(); err == nil {
		t.Fatal("aceptó un ámbito ajeno")
	}
}
func TestNoAdmiteAsignacionRevocadaONoCanonica(t *testing.T) {
	p := planPrueba(t)
	p.AsignacionCanonica = append([]byte(" "), p.AsignacionCanonica...)
	if _, err := p.Canonica(); err == nil {
		t.Fatal("aceptó bytes no canónicos")
	}
	p = planPrueba(t)
	p.AsignacionCanonica = bytes.Replace(p.AsignacionCanonica, []byte(`"estado":"activa"`), []byte(`"estado":"revocada"`), 1)
	if _, err := p.Canonica(); err == nil {
		t.Fatal("aceptó una revocación como alta")
	}
}
func TestClavesDuplicadasNoCambianPlanRevisado(t *testing.T) {
	p := planPrueba(t)
	s := Solicitud{Plan: p, Operador: MaterialOperador{Decision: []byte(`{}`), Motivo: []byte(`{}`), PersonaVersion: 1, PerfilVersion: 1}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LeerSolicitud(b); err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte(`"version":1`), []byte(`"version":2,"version":1`), 1)
	if _, err := LeerSolicitud(b); err == nil {
		t.Fatal("aceptó una clave duplicada")
	}
}
func TestRecursoNoIntercambiaPermisos(t *testing.T) {
	p := planPrueba(t)
	h, _ := p.Huella()
	for op, tipo := range map[string]string{"preparar": "perfil", "aprobar": "propuesta_perfil", "aplicar": "perfil", "recuperar": "recibo_perfil"} {
		r, err := p.Recurso(op)
		if err != nil || r.Tipo != tipo || r.Atributos["plan_sha256"] != h || r.Referencia != "cargo_ct:"+p.Clave {
			t.Fatalf("recurso inválido %s", op)
		}
		if _, err = r.HuellaContextoAutorizacionSHA256(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.Recurso("publicar"); err == nil {
		t.Fatal("aceptó operación desconocida")
	}
}

func TestAdjuntoCertificadoLigadoALaAprobacion(t *testing.T) {
	p := planPrueba(t)
	h, _ := p.Huella()
	p.VinculoCertificadoCanonico = `{"version":1,"certificado_der_sha256":"sintetico"}`
	got, err := p.Huella()
	if err != nil || got == h {
		t.Fatal("adjunto no cambia la aprobación del plan")
	}
	p.VinculoCertificadoCanonico = `{"version":2,"version":1}`
	if _, err := p.Huella(); err == nil {
		t.Fatal("aceptó clave duplicada en descriptor")
	}
}
