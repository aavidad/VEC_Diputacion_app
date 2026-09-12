package config

import (
	"errors"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLAdministracionFallaCerradaYRedacta(t *testing.T) {
	if (ConfiguracionPostgreSQLAdministracion{}).Configurada() {
		t.Fatal("la configuración vacía no debe activar ADMIN")
	}
	if err := (ConfiguracionPostgreSQLAdministracion{dsnCorreo: "postgres://solo"}).Validar(); !errors.Is(err, ErrConfiguracionPostgreSQLAdministracionIncompleta) {
		t.Fatalf("error parcial = %v", err)
	}
	c, err := NuevaConfiguracionPostgreSQLAdministracion("postgres://correo", "postgres://autorizacion", "postgres://registro", "postgres://revalidacion", "postgres://contexto")
	if err != nil || !c.Configurada() {
		t.Fatalf("configuración válida = %#v, %v", c, err)
	}
	if _, err = NuevaConfiguracionPostgreSQLAdministracion("postgres://misma", "postgres://misma", "postgres://registro", "postgres://revalidacion", "postgres://contexto"); !errors.Is(err, ErrConfiguracionPostgreSQLAdministracionNoSeparada) {
		t.Fatalf("error de separación = %v", err)
	}
	for _, representacion := range []string{c.String(), c.GoString()} {
		if strings.Contains(representacion, "postgres://") {
			t.Fatalf("DSN expuesto: %q", representacion)
		}
	}
}

func TestConfiguracionPostgreSQLAdministracionEntregaDSNSeparados(t *testing.T) {
	c, err := NuevaConfiguracionPostgreSQLAdministracion("correo", "autorizacion", "registro", "revalidacion", "contexto")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := c.DSNCorreo(); got != "correo" {
		t.Fatalf("correo = %q", got)
	}
	if got, _ := c.DSNRegistroAutorizacion(); got != "autorizacion" {
		t.Fatalf("autorizacion = %q", got)
	}
	r, rv, ca, err := c.DSNIdentidad()
	if err != nil || r != "registro" || rv != "revalidacion" || ca != "contexto" {
		t.Fatalf("identidad = %q/%q/%q, %v", r, rv, ca, err)
	}
}
