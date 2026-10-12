package postgres

import (
	"context"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestAutoridadVersionarRolBolsaB103DeniegaActosAjenos(t *testing.T) {
	a := &AutoridadVersionarRolBolsaB103{}
	if _, err := a.AplicarActoOrdinario(context.Background(), domain.SolicitudActoAdministracionPerfiles{}); err == nil {
		t.Fatal("acto ordinario aceptado")
	}
	if _, err := a.ProponerActoSensible(context.Background(), domain.SolicitudActoAdministracionPerfiles{}); err == nil {
		t.Fatal("propuesta sensible ajena aceptada")
	}
	if _, err := a.CerrarPropuestaSensible(context.Background(), domain.SolicitudCierrePropuestaAdministracionPerfiles{}); err == nil {
		t.Fatal("cierre sensible ajeno aceptado")
	}
	if err := a.disponible(context.Background()); err == nil {
		t.Fatal("adaptador sin dependencias disponible")
	}
}

func TestRecursoVersionarRolBolsaB103CierreSeparaAmbitosYEsquemas(t *testing.T) {
	base, _, _, _ := contratoV2Prueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{"organizacion:uno"}},
		{Clave: "unidad_ref", Valores: []string{"unidad:uno"}},
	}
	s := domain.SolicitudCierreVersionarRolBolsaB103{
		OperacionRef:          "cierre_admin:" + strings.Repeat("4", 32),
		PropuestaRef:          "propuesta_admin:" + strings.Repeat("1", 32),
		PropuestaHuellaSHA256: strings.Repeat("a", 64),
		Aprobador:             base.Actor, Evidencia: base.Evidencia,
		InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Decision:                domain.DecisionAprobarPropuestaPerfil,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("c", 64),
			EntradaClave: "motivo_" + strings.Repeat("a", 32)},
		CorrelacionRef: "correlacion_" + strings.Repeat("5", 32),
	}
	e, err := materialCierreVersionarRolBolsaB103(s)
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoVersionarRolBolsaB103(e, s.InstantaneaAutorizacion.AsignacionPerfil)
	if err != nil || r.Referencia != s.PropuestaRef ||
		r.Ambitos["organizacion_ref"] != "organizacion:uno" || r.Ambitos["unidad_ref"] != "unidad:uno" {
		t.Fatalf("recurso ADMIN v2: %+v %v", r, err)
	}
	if _, err := RecursoVersionarRolBolsa(e, s.InstantaneaAutorizacion.AsignacionPerfil); err == nil {
		t.Fatal("B1 aceptó sobre v2")
	}
	for _, caso := range []string{"referencia", "ambito_rrhh", "esquema_v1", "accion"} {
		t.Run(caso, func(t *testing.T) {
			x := e
			a := s.InstantaneaAutorizacion.AsignacionPerfil
			switch caso {
			case "referencia":
				x.Referencia = "propuesta_admin:" + strings.Repeat("f", 32)
			case "ambito_rrhh":
				a.Ambitos[0].Clave = "ambito_ref"
			case "esquema_v1":
				x.Material = []byte(strings.Replace(string(e.Material),
					"administracion_version_rol_bolsa_cierre_v2", "administracion_version_rol_bolsa_cierre_v1", 1))
			case "accion":
				x.Accion = "administracion.perfiles.version_bolsa.rechazar"
			}
			if _, err := RecursoVersionarRolBolsaB103(x, a); err == nil {
				t.Fatal("recurso ajeno aceptado")
			}
		})
	}
}
