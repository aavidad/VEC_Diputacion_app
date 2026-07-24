package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLPublicaFallaCerradaYRedacta(t *testing.T) {
	if _, err := NuevaConfiguracionPostgreSQLPublica(" "); !errors.Is(err, ErrConfiguracionPostgreSQLPublicaIncompleta) {
		t.Fatalf("conexion vacia: %v", err)
	}
	secreto := "postgres://lector:clave-super-secreta@db-publica/vec?sslmode=verify-full"
	configuracion, err := NuevaConfiguracionPostgreSQLPublica(secreto)
	if err != nil {
		t.Fatal(err)
	}
	serializada, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	representaciones := []string{
		fmt.Sprint(configuracion), fmt.Sprintf("%#v", configuracion),
		fmt.Sprintf("%+v", configuracion), string(serializada),
		configuracion.LogValue().String(), slog.Any("postgresql", configuracion).Value.String(),
	}
	for _, representacion := range representaciones {
		if strings.Contains(representacion, "clave-super-secreta") || strings.Contains(representacion, "db-publica") {
			t.Fatalf("la configuracion filtro el DSN: %q", representacion)
		}
	}
	if obtenida, err := configuracion.DSN(); err != nil || obtenida != secreto {
		t.Fatalf("DSN encapsulado = %q, %v", obtenida, err)
	}
}

func TestLoadLeeSoloLaConexionPublicaConfigurada(t *testing.T) {
	const dsn = "postgres://lector:clave@db-publica/vec?sslmode=verify-full"
	const manifiesto = "2a85abd0a1e78d828fe27baf619349caf8e4e8a3e0bf20815279dd98a966889a"
	t.Setenv(EnvBolsaPublicaDatabaseURL, " "+dsn+" ")
	t.Setenv(EnvBolsaPublicaManifiestoSHA256, " "+manifiesto+" ")
	cargada := Load()
	obtenido, err := cargada.BolsaPublicaPostgreSQL.DSN()
	if err != nil || obtenido != dsn {
		t.Fatalf("conexion publica = %q, %v", obtenido, err)
	}
	if cargada.BolsaPublicaManifiestoSHA256 != manifiesto {
		t.Fatalf("manifiesto publico = %q", cargada.BolsaPublicaManifiestoSHA256)
	}
}

func TestHuellaManifiestoPublicoReservaTestigoDeInvalidacion(t *testing.T) {
	validos := []string{
		"2a85abd0a1e78d828fe27baf619349caf8e4e8a3e0bf20815279dd98a966889a",
		" 2a85abd0a1e78d828fe27baf619349caf8e4e8a3e0bf20815279dd98a966889a ",
	}
	for _, valor := range validos {
		if err := ValidarHuellaManifiestoPublico(valor); err != nil {
			t.Fatalf("huella valida %q: %v", valor, err)
		}
	}
	for _, valor := range []string{"", strings.Repeat("0", 64), strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		if !errors.Is(ValidarHuellaManifiestoPublico(valor), ErrHuellaManifiestoPublicoInvalida) {
			t.Fatalf("huella invalida aceptada: %q", valor)
		}
	}
}
