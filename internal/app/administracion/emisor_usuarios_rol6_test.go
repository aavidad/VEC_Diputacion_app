package administracion

import (
	"context"
	"testing"
)

// AUT48: la versión del rol de Aplicación no se fija; decide la concesión
// exacta de usuarios presente en la versión que señala la asignación.
func TestEmisorUsuariosVersionVigenteConConcesionExacta(t *testing.T) {
	for _, v := range []int{4, 5, 6, 7} {
		e, actor, v2, s, m := escenarioSolicitudUsuarios(t)
		s.VersionRol.Version = v
		s.AsignacionPerfil.VersionRolRef = s.VersionRol.Referencia()
		s.ControlVigenciaVersionRol.VersionRolRef = s.VersionRol.Referencia()
		if _, _, _, err := e.solicitud(context.Background(), actor, v2, s, m); err != nil {
			t.Fatalf("version%d con concesion exacta rechazada", v)
		}
		s.VersionRol.Concesiones[0].Obligaciones = nil
		if _, _, _, err := e.solicitud(context.Background(), actor, v2, s, m); err == nil {
			t.Fatalf("version%d sin auditar admitida", v)
		}
	}
	e, actor, v2, s, m := escenarioSolicitudUsuarios(t)
	s.VersionRol.Concesiones = s.VersionRol.Concesiones[1:]
	if _, _, _, err := e.solicitud(context.Background(), actor, v2, s, m); err == nil {
		t.Fatal("version sin la concesion de usuarios admitida")
	}
	e, actor, v2, s, m = escenarioSolicitudUsuarios(t)
	s.VersionRol.RolID = "operador_plataforma"
	s.AsignacionPerfil.VersionRolRef = s.VersionRol.Referencia()
	s.ControlVigenciaVersionRol.VersionRolRef = s.VersionRol.Referencia()
	if _, _, _, err := e.solicitud(context.Background(), actor, v2, s, m); err == nil {
		t.Fatal("rol distinto de Aplicación admitido")
	}
}
