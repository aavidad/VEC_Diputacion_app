package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDSNBolsaRelevoCeseExigeLoginSeparadoYRedacta(t *testing.T) {
	if _, err := (Config{}).DSNBolsaRelevoCeseSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaRelevoCeseIncompleta) {
		t.Fatalf("sin DSN: %v", err)
	}
	const secreto = "postgres://relevo_cese:secreto-cese@localhost/vec?sslmode=require"
	t.Setenv(EnvBolsaRelevoCeseDatabaseURL, " "+secreto+" ")
	c := Load()
	if dsn, err := c.DSNBolsaRelevoCeseSeparado(); err != nil || dsn != secreto {
		t.Fatalf("DSN nominal: %v", err)
	}
	paraJSON, err := json.Marshal(c.BolsaRelevoCesePostgreSQL)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", c.BolsaRelevoCesePostgreSQL, c.BolsaRelevoCesePostgreSQL, paraJSON), "secreto-cese") {
		t.Fatal("el DSN se imprimió")
	}
	c.BolsaRelevoNoIncorporacionPostgreSQL = NuevaConfiguracionPostgreSQLBolsaRelevoNoIncorporacion("postgres://relevo_cese:otro@localhost/vec?sslmode=require")
	if _, err := c.DSNBolsaRelevoCeseSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaRelevoCeseNoSeparada) {
		t.Fatalf("LOGIN reutilizado: %v", err)
	}
}
