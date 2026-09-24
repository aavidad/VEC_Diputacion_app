package config

import (
	"errors"
	"strings"
	"testing"
)

func TestContactoUsuarioOpcionalSeparadoYRedactado(t *testing.T) {
	if (Config{}).ContactoUsuarioPostgreSQL.Configurada() {
		t.Fatal("contacto debe ser opt-in")
	}
	if _, _, _, _, err := (ConfiguracionPostgreSQLContactoUsuario{}).DSNSeparados(Config{}); !errors.Is(err, ErrConfiguracionPostgreSQLContactoUsuarioIncompleta) {
		t.Fatalf("ausencia: %v", err)
	}
	cfg := Config{ContactoUsuarioPostgreSQL: NuevaConfiguracionPostgreSQLContactoUsuario("postgres://escritor@localhost/vec", "postgres://lector@localhost/vec", "postgres://fuente@localhost/vec", "postgres://motivos@localhost/vec")}
	w, r, f, m, err := cfg.ContactoUsuarioPostgreSQL.DSNSeparados(cfg)
	if err != nil || w == "" || r == "" || f == "" || m == "" {
		t.Fatalf("tres logins propios: %v", err)
	}
	if strings.Contains(cfg.ContactoUsuarioPostgreSQL.String(), "escritor") {
		t.Fatal("DSN expuesto")
	}
	mismos := Config{ContactoUsuarioPostgreSQL: NuevaConfiguracionPostgreSQLContactoUsuario("postgres://igual@localhost/vec", "postgres://igual@localhost/vec", "postgres://fuente@localhost/vec", "postgres://motivos@localhost/vec")}
	if _, _, _, _, err := mismos.ContactoUsuarioPostgreSQL.DSNSeparados(mismos); !errors.Is(err, ErrConfiguracionPostgreSQLContactoUsuarioNoSeparada) {
		t.Fatalf("LOGIN compartido: %v", err)
	}
	parcial := Config{ContactoUsuarioPostgreSQL: NuevaConfiguracionPostgreSQLContactoUsuario("postgres://escritor@localhost/vec", "", "", "")}
	if _, _, _, _, err := parcial.ContactoUsuarioPostgreSQL.DSNSeparados(parcial); !errors.Is(err, ErrConfiguracionPostgreSQLContactoUsuarioIncompleta) {
		t.Fatalf("parcial: %v", err)
	}
}
