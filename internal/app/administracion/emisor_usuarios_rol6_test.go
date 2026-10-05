package administracion

import (
	"context"
	"testing"
)

func TestEmisorUsuariosRol5Y6ConHerenciaExacta(t *testing.T) {
	for _, v := range []int{4, 5, 6, 7} {
		e, actor, v2, s, m := escenarioSolicitudUsuarios(t)
		s.VersionRol.Version = v
		s.AsignacionPerfil.VersionRolRef = s.VersionRol.Referencia()
		s.ControlVigenciaVersionRol.VersionRolRef = s.VersionRol.Referencia()
		_, _, _, err := e.solicitud(context.Background(), actor, v2, s, m)
		if (err == nil) != (v == 5 || v == 6) {
			t.Fatalf("version%d admisionincorrecta", v)
		}
		if v == 6 {
			s.VersionRol.Concesiones[0].Obligaciones = nil
			if _, _, _, err := e.solicitud(context.Background(), actor, v2, s, m); err == nil {
				t.Fatal("Rol6_sin_auditar")
			}
		}
	}
}
