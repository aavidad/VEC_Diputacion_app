package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestConexionesNominalesExternasSeCarganSinExponerContrasena(t *testing.T) {
	const secretoSintetico = "clave_sintetica_no_real"
	t.Setenv(EnvExternoBolsaDatabaseURL, "postgres://vec_externo_bolsa_desarrollo:"+secretoSintetico+"@localhost/vec")
	cfg := Load()
	dsn, err := cfg.ExternoBolsaPostgreSQL.DSN()
	if err != nil || !strings.Contains(dsn, secretoSintetico) {
		t.Fatal("no se cargó la conexión nominal")
	}
	if strings.Contains(fmt.Sprintf("%+v", cfg.ExternoBolsaPostgreSQL), secretoSintetico) ||
		strings.Contains(fmt.Sprintf("%#v", cfg.ExternoBolsaPostgreSQL), secretoSintetico) {
		t.Fatal("la conexión nominal aparece en el volcado de configuración")
	}
	if _, err := (ConfiguracionPostgreSQLExterna{}).DSN(); err == nil {
		t.Fatal("conexión ausente aceptada")
	}
}
