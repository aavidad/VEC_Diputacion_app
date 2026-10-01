package ejecucioncopias

import (
	"strings"
	"testing"
)

func TestSesionArchivadaExigeMTLSYPermisoDeSesion(t *testing.T) {
	casos := []struct {
		nombre, cuerpo string
		valida         bool
	}{
		{"dnie autorizado", `{"principal":{"id":"actor:sintetico","auth_assurance":"alto","auth_method":"dnie","permissions":["vec.session.read"]}}`, true},
		{"certificado autorizado", `{"principal":{"id":"actor:sintetico","auth_assurance":"alto","auth_method":"certificado","permissions":["vec.session.read"]}}`, true},
		{"demo rechazado", `{"principal":{"id":"actor:sintetico","auth_assurance":"alto","auth_method":"demo","permissions":["vec.session.read"]}}`, false},
		{"permiso ajeno", `{"principal":{"id":"actor:sintetico","auth_assurance":"alto","auth_method":"dnie","permissions":["vec.modules.read"]}}`, false},
		{"garantia insuficiente", `{"principal":{"id":"actor:sintetico","auth_assurance":"bajo","auth_method":"dnie","permissions":["vec.session.read"]}}`, false},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			if got := sesionAutorizadaRuntime([]byte(tc.cuerpo)); got != tc.valida {
				t.Fatalf("sesión aceptada=%v, esperada=%v", got, tc.valida)
			}
		})
	}
}

func TestPerfilSesionArchivadaImpideDSNYTransportes(t *testing.T) {
	c := ConfiguracionRuntimeLocal{Perfil: "sesion_sintetica", BinarioID: "fisica:vec-server", MaterialID: "fisica:material", CatalogoPersonalID: "fisica:catalogo_personal", Puerto: 18083, UsuarioPostgreSQL: "cs06_ejecutor"}
	env, ok := entornoRuntime(c, "/componentes/0002/material/contenido", "/componentes/0003/contenido")
	if !ok {
		t.Fatal("perfil sintético válido rechazado")
	}
	for k := range env {
		if strings.HasSuffix(k, "_DATABASE_URL") || strings.Contains(k, "SMTP") || strings.Contains(k, "DESPACH") {
			t.Fatalf("transporte no admitido: %s", k)
		}
	}
	if env["VEC_HTTP_ADDR"] != "127.0.0.1:18083" {
		t.Fatal("el listener salió de loopback")
	}
	c.Entorno = map[string]string{"VEC_CT_DATABASE_URL": "postgresql://usuario@principal/vec"}
	if _, ok := entornoRuntime(c, "/componentes/0002/material/contenido", "/componentes/0003/contenido"); ok {
		t.Fatal("el perfil de sesión aceptó una conexión CT")
	}
}
