package domain

import "testing"

func TestVersionRolAplicacionAdmitida(t *testing.T) {
	for _, v := range []string{"rol:administracion_perfiles:v5", "rol:administracion_perfiles:v6", "rol:administracion_perfiles:v7", "rol:administracion_perfiles:v123456789"} {
		if !VersionRolAplicacionAdmitida(v) {
			t.Fatalf("debe admitir %q", v)
		}
	}
	for _, v := range []string{"", "rol:administracion_perfiles:v", "rol:administracion_perfiles:v0", "rol:administracion_perfiles:v05",
		"rol:administracion_perfiles:v1234567890", "rol:administracion_perfiles:v5a", "rol:administracion_perfiles:v-5",
		"rol:operador_plataforma:v1", "rol:administracion_perfiles_x:v5", " rol:administracion_perfiles:v5"} {
		if VersionRolAplicacionAdmitida(v) {
			t.Fatalf("no debe admitir %q", v)
		}
	}
}
