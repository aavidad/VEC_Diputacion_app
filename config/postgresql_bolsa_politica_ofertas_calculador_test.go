package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDSNBolsaPoliticaOfertasCalculadorExigeLoginSeparado(t *testing.T) {
	if _, err := (Config{}).DSNBolsaPoliticaOfertasCalculadorSeparado(); !errors.Is(err, ErrBolsaPoliticaOfertasCalculadorIncompleto) {
		t.Fatalf("sin calculador: %v", err)
	}
	const dsn = "postgres://calculador_politica:secreto-calculador@localhost/vec?sslmode=require"
	t.Setenv(EnvBolsaPoliticaOfertasCalculadorDatabaseURL, dsn)
	c := Load()
	if got, err := c.DSNBolsaPoliticaOfertasCalculadorSeparado(); err != nil || got != dsn {
		t.Fatalf("conexión calculador: %v", err)
	}
	serializada, err := json.Marshal(c.BolsaPoliticaOfertasCalculadorPostgreSQL)
	if err != nil || strings.Contains(fmt.Sprintf("%+v %#v %s", c.BolsaPoliticaOfertasCalculadorPostgreSQL, c.BolsaPoliticaOfertasCalculadorPostgreSQL, serializada), "secreto-calculador") {
		t.Fatal("configuración del calculador expuesta")
	}
	c.BolsaRelevoCesePostgreSQL = NuevaConfiguracionPostgreSQLBolsaRelevoCese("postgres://calculador_politica:otra@localhost/vec?sslmode=require")
	if _, err := c.DSNBolsaPoliticaOfertasCalculadorSeparado(); !errors.Is(err, ErrBolsaPoliticaOfertasCalculadorNoSeparado) {
		t.Fatalf("LOGIN compartido con relevo: %v", err)
	}
}
