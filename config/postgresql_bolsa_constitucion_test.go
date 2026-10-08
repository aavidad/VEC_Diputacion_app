package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDSNBolsaConstitucionSeparado(t *testing.T) {
	t.Setenv(EnvBolsaConstitucionDatabaseURL, "")
	if _, err := Load().DSNBolsaConstitucionSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaConstitucionIncompleta) {
		t.Fatalf("sin DSN: %v", err)
	}
	const web = "postgres://web:secreto@localhost/vec?sslmode=require"
	const cli = "postgres://cli:secreto@localhost/vec?sslmode=require"
	t.Setenv(EnvBolsaLlamamientosDatabaseURL, web)
	t.Setenv(EnvBolsaConstitucionDatabaseURL, web)
	if _, err := Load().DSNBolsaConstitucionSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaConstitucionNoSeparada) {
		t.Fatalf("LOGIN web reutilizado: %v", err)
	}
	t.Setenv(EnvBolsaConstitucionDatabaseURL, cli)
	if got, err := Load().DSNBolsaConstitucionSeparado(); err != nil || got != cli {
		t.Fatalf("LOGIN CLI separado: %v", err)
	}
	t.Setenv(EnvBolsaConstitucionDatabaseURL, "postgres://web:otra@localhost/vec?sslmode=require")
	if _, err := Load().DSNBolsaConstitucionSeparado(); !errors.Is(err, ErrConfiguracionPostgreSQLBolsaConstitucionNoSeparada) {
		t.Fatalf("mismo LOGIN con distinto secreto: %v", err)
	}
}

func TestConfiguracionBolsaConstitucionNoFiltraDSN(t *testing.T) {
	const secreto = "supersecreto-cli"
	c := ConfiguracionPostgreSQLBolsaConstitucion{dsn: secreto}
	json, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, representacion := range []string{fmt.Sprint(c), fmt.Sprintf("%#v", c), fmt.Sprintf("%+v", c), string(json)} {
		if strings.Contains(representacion, secreto) {
			t.Fatal("la configuracion expuso el DSN")
		}
	}
}
