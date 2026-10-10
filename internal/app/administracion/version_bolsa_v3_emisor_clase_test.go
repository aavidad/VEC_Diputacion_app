package administracion

import (
	"strings"
	"testing"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestClaseSnapshotVersionarRolBolsaSeparaInstantaneaAmbitoYConcesion(t *testing.T) {
	for _, caso := range []struct{ nombre, clase string }{
		{"valida", ""}, {"sin_concesion", "v3_sin_concesion"}, {"finalidad_ajena", "v3_sin_concesion"},
		{"ambito_ajeno", "v3_ambito"}, {"rol_sistemas", "v3_instantanea"}, {"actor_ajeno", "v3_instantanea"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			u, actor, _, snapshot, _ := escenarioSolicitudUsuarios(t)
			snapshot.VersionRol.Concesiones = []domain.ConcesionRol{{Accion: gobierno.AccionVersionarRolBolsaProponer,
				ModuloID: "administracion", TipoRecurso: "definicion_rol",
				Finalidades: []string{gobierno.FinalidadVersionarRolBolsa}, GarantiaMinima: domain.AuthAssuranceHigh}}
			snapshot.AsignacionPerfil.VersionRolRef = snapshot.VersionRol.Referencia()
			snapshot.ControlVigenciaVersionRol.VersionRolRef = snapshot.VersionRol.Referencia()
			recurso := domain.RecursoAutorizable{Referencia: "rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v7",
				ModuloID: "administracion", Tipo: "definicion_rol",
				Ambitos: map[string]string{"organizacion_ref": snapshot.AsignacionPerfil.Ambitos[0].Valores[0],
					"unidad_ref": snapshot.AsignacionPerfil.Ambitos[1].Valores[0]},
				Atributos: map[string]string{"solicitud_sha256": strings.Repeat("a", 64)}}
			switch caso.nombre {
			case "sin_concesion":
				snapshot.VersionRol.Concesiones[0].Accion = gobierno.AccionVersionarRolBolsaAprobar
			case "finalidad_ajena":
				snapshot.VersionRol.Concesiones[0].Finalidades = []string{"otra"}
			case "ambito_ajeno":
				recurso.Ambitos["unidad_ref"] = "unidad:ajena"
			case "rol_sistemas":
				snapshot.VersionRol.RolID = "administracion_sistemas"
			case "actor_ajeno":
				actor.PersonaRef = "per_" + strings.Repeat("f", 22)
			}
			if got := claseSnapshotVersionarRolBolsa(snapshot, actor, recurso,
				gobierno.AccionVersionarRolBolsaProponer, u.reloj.Ahora()); got != caso.clase {
				t.Fatalf("clase=%q quería %q", got, caso.clase)
			}
			if caso.clase != "" && !ports.ClaseVersionBolsaAdmitida(caso.clase) {
				t.Fatal("clase fuera de la lista cerrada")
			}
		})
	}
}
