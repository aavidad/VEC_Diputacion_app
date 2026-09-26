package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAuditoriaFronteraBolsaSoloSeExigeAlComponerBolsaYSeRedacta(t *testing.T) {
	base, err := NuevaConfiguracionPostgreSQLContratacionTemporal("ejecucion", "gobierno", "registro", "confirmador", "lector")
	if err != nil {
		t.Fatal(err)
	}
	if dsn, err := (Config{ContratacionTemporalPostgreSQL: base}).DSNBolsaAuditoriaFronteraSeparado(); err != nil || dsn != "" {
		t.Fatalf("sin Bolsa = (%q, %v)", dsn, err)
	}
	base.dsnBolsaLlamamientos = "bolsa"
	configuracion := Config{ContratacionTemporalPostgreSQL: base}
	if _, err := configuracion.DSNBolsaAuditoriaFronteraSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraIncompleta) {
		t.Fatalf("Bolsa parcial no fallo cerrada: %v", err)
	}
	configuracion.BolsaAuditoriaFronteraPostgreSQL.dsn = " bolsa "
	if _, err := configuracion.DSNBolsaAuditoriaFronteraSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraNoSeparada) {
		t.Fatalf("DSN de Bolsa reutilizado: %v", err)
	}
	const secreto = "postgres://auditoria:secreto-frontera@localhost/vec"
	t.Setenv(EnvBolsaLlamamientosDatabaseURL, " postgres://bolsa@localhost/vec ")
	t.Setenv(EnvBolsaAuditoriaFronteraDatabaseURL, " "+secreto+" ")
	cargada := Load()
	if dsn, err := cargada.DSNBolsaAuditoriaFronteraSeparado(); err != nil || dsn != secreto {
		t.Fatalf("DSN auditoría Bolsa = (%q, %v)", dsn, err)
	}
	t.Setenv(EnvBolsaPublicaDatabaseURL, "postgres://auditoria:otra-clave@localhost/otra_base")
	if _, err := Load().DSNBolsaAuditoriaFronteraSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraNoSeparada) {
		t.Fatalf("LOGIN reutilizado con URL distinta: %v", err)
	}
	serializada, err := json.Marshal(cargada.BolsaAuditoriaFronteraPostgreSQL)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", cargada.BolsaAuditoriaFronteraPostgreSQL, cargada.BolsaAuditoriaFronteraPostgreSQL, serializada), "secreto-frontera") {
		t.Fatal("DSN de auditoría Bolsa expuesto")
	}
}

// El relevo de no incorporaciones exige su propio LOGIN: ni el de Bolsa, ni
// el de auditoría, ni ninguno de CT.
func TestRelevoNoIncorporacionExigeLoginPropioYSeRedacta(t *testing.T) {
	if _, err := (Config{}).DSNBolsaRelevoNoIncorporacionSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionIncompleta) {
		t.Fatalf("sin conexión: %v", err)
	}
	const secreto = "postgres://relevo:secreto-relevo@localhost/vec"
	t.Setenv(EnvBolsaLlamamientosDatabaseURL, "postgres://bolsa@localhost/vec")
	t.Setenv(EnvBolsaAuditoriaFronteraDatabaseURL, "postgres://auditoria@localhost/vec")
	t.Setenv(EnvBolsaRelevoNoIncorporacionDatabaseURL, " "+secreto+" ")
	cargada := Load()
	if dsn, err := cargada.DSNBolsaRelevoNoIncorporacionSeparado(); err != nil || dsn != secreto {
		t.Fatalf("DSN relevo = (%q, %v)", dsn, err)
	}
	for _, reutilizado := range []string{"postgres://bolsa:otra@localhost/otra", "postgres://auditoria:otra@localhost/otra"} {
		t.Setenv(EnvBolsaRelevoNoIncorporacionDatabaseURL, reutilizado)
		if _, err := Load().DSNBolsaRelevoNoIncorporacionSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionNoSeparada) {
			t.Fatalf("LOGIN reutilizado %q: %v", reutilizado, err)
		}
	}
	serializada, err := json.Marshal(cargada.BolsaRelevoNoIncorporacionPostgreSQL)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", cargada.BolsaRelevoNoIncorporacionPostgreSQL, cargada.BolsaRelevoNoIncorporacionPostgreSQL, serializada), "secreto-relevo") {
		t.Fatal("DSN del relevo expuesto")
	}
}
