package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConfiguracionCalendariosRedactaYSepara(t *testing.T) {
	t.Setenv(EnvCalendariosDatabaseURL, " postgres://lector_calendarios:secreto@127.0.0.1:5432/vec?sslmode=disable ")
	cfg := Load().Normalize()
	if !cfg.CalendariosPostgreSQL.Configurada() {
		t.Fatal("debe leerse del entorno")
	}
	dsn, err := cfg.DSNCalendarios()
	if err != nil || !strings.HasPrefix(dsn, "postgres://lector_calendarios") {
		t.Fatalf("dsn: %v", err)
	}
	texto, _ := json.Marshal(cfg)
	if strings.Contains(string(texto), "secreto") || strings.Contains(fmt.Sprintf("%v %+v %#v", cfg.CalendariosPostgreSQL, cfg, cfg.CalendariosPostgreSQL), "secreto") {
		t.Fatal("la conexión no se redacta")
	}
	cfg.DietasBorradoresPostgreSQL = ConfiguracionDietasBorradores{dsnDietas: "postgres://lector_calendarios:otra@127.0.0.1:5432/vec?sslmode=disable"}
	if _, err := cfg.DSNCalendarios(); !errors.Is(err, ErrConfiguracionCalendariosNoSeparada) {
		t.Fatalf("login compartido: %v", err)
	}
	if dsn, err := (Config{}).DSNCalendarios(); dsn != "" || err != nil {
		t.Fatal("sin configurar no hay conexión ni error")
	}
}
