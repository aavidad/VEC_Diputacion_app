package postgres

import (
	"context"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestRecursoVersionarRolBolsaExternaCierreUsaSoloSesionADMIN(t *testing.T) {
	base, _, _, _ := contratoV2Prueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{"organizacion:uno"}},
		{Clave: "unidad_ref", Valores: []string{"unidad:uno"}},
	}
	s := domain.SolicitudCierreVersionarRolBolsaExterna{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia, InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	e, err := materialCierreVersionarRolBolsaExterna(s)
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoVersionarRolBolsaExterna(e, s.InstantaneaAutorizacion.AsignacionPerfil)
	if err != nil || r.Referencia != s.PropuestaRef || r.Ambitos["unidad_ref"] != "unidad:uno" ||
		r.Ambitos["organizacion_ref"] != "organizacion:uno" {
		t.Fatalf("recurso: %+v %v", r, err)
	}
	if _, err := RecursoVersionarRolBolsaB103(e, s.InstantaneaAutorizacion.AsignacionPerfil); err == nil {
		t.Fatal("B103 aceptó el sobre externo")
	}
	for _, caso := range []struct {
		name    string
		alterar func(*Efecto, *domain.AsignacionPerfil)
	}{
		{"audiencia", func(e *Efecto, _ *domain.AsignacionPerfil) { e.Audiencia = AudienciaVersionarRolBolsaAprobar }},
		{"referencia", func(e *Efecto, _ *domain.AsignacionPerfil) {
			e.Referencia = "propuesta_admin:" + strings.Repeat("f", 32)
		}},
		{"ambito_candidato", func(_ *Efecto, a *domain.AsignacionPerfil) { a.Ambitos[0].Clave = "candidato_ref" }},
		{"material_actor", func(e *Efecto, _ *domain.AsignacionPerfil) {
			e.Material = []byte(strings.Replace(string(e.Material), s.Aprobador.PersonaRef, "per_"+strings.Repeat("x", 22), 1))
		}},
	} {
		t.Run(caso.name, func(t *testing.T) {
			x := e
			a := s.InstantaneaAutorizacion.AsignacionPerfil
			a.Ambitos = []domain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"organizacion:uno"}},
				{Clave: "unidad_ref", Valores: []string{"unidad:uno"}}}
			caso.alterar(&x, &a)
			if _, err := RecursoVersionarRolBolsaExterna(x, a); err == nil {
				t.Fatal("recurso ajeno admitido")
			}
		})
	}
}

func TestAutoridadVersionarRolBolsaExternaFallaCerradaSinACL(t *testing.T) {
	a := &AutoridadVersionarRolBolsaExterna{}
	if err := a.disponible(context.Background()); err == nil {
		t.Fatal("autoridad sin pool disponible")
	}
	if _, err := a.AplicarActoOrdinario(context.Background(), domain.SolicitudActoAdministracionPerfiles{}); err == nil {
		t.Fatal("acto ajeno")
	}
	if !strings.Contains(acreditarVersionarRolBolsaExternaSQL, "vec_admin_version_rol_bolsa_externa_ejecutor") ||
		!strings.Contains(acreditarVersionarRolBolsaExternaSQL, "resolver_base_version_rol_bolsa_externa_v1()") ||
		strings.Contains(acreditarVersionarRolBolsaExternaSQL, "proponer_version_rol_bolsa_v2(") {
		t.Fatal("preflight de la fachada nominal divergente")
	}
}
