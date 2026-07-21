package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLBorradoresFallaCerradaSiFaltaUnaConexion(t *testing.T) {
	casos := []struct {
		nombre              string
		ejecutor, proyector string
		verificador         string
	}{
		{nombre: "todas ausentes"},
		{nombre: "falta ejecutor", proyector: "postgres://proyector", verificador: "postgres://verificador"},
		{nombre: "falta proyector", ejecutor: "postgres://ejecutor", verificador: "postgres://verificador"},
		{nombre: "falta verificador", ejecutor: "postgres://ejecutor", proyector: "postgres://proyector"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := NuevaConfiguracionPostgreSQLBorradores(
				caso.ejecutor, caso.proyector, caso.verificador,
			)
			if !errors.Is(err, ErrConfiguracionPostgreSQLBorradoresIncompleta) {
				t.Fatalf("configuracion parcial aceptada: %v", err)
			}
		})
	}
}

func TestConfiguracionPostgreSQLBorradoresExigeTresDSNDistintos(t *testing.T) {
	for _, dsns := range [][3]string{
		{"postgres://misma", "postgres://misma", "postgres://otra"},
		{"postgres://misma", "postgres://otra", "postgres://misma"},
		{"postgres://otra", "postgres://misma", "postgres://misma"},
	} {
		_, err := NuevaConfiguracionPostgreSQLBorradores(dsns[0], dsns[1], dsns[2])
		if !errors.Is(err, ErrConfiguracionPostgreSQLBorradoresNoSeparada) {
			t.Fatalf("DSN reutilizado aceptado: %v", err)
		}
	}
}

func TestLoadCargaLasTresConexionesSinValoresPredeterminados(t *testing.T) {
	for _, variable := range []string{
		EnvBolsaBorradoresEjecutorConsultaDatabaseURL,
		EnvBolsaBorradoresProyectorGobiernoDatabaseURL,
		EnvBolsaBorradoresVerificadorReciboDatabaseURL,
	} {
		t.Setenv(variable, "")
	}
	if err := Load().BolsaBorradoresPostgreSQL.Validar(); !errors.Is(err, ErrConfiguracionPostgreSQLBorradoresIncompleta) {
		t.Fatalf("la ausencia ambiental obtuvo un valor implicito: %v", err)
	}

	t.Setenv(EnvBolsaBorradoresEjecutorConsultaDatabaseURL, " postgres://ejecutor:uno@bd/vec ")
	t.Setenv(EnvBolsaBorradoresProyectorGobiernoDatabaseURL, "postgres://proyector:dos@bd/vec")
	t.Setenv(EnvBolsaBorradoresVerificadorReciboDatabaseURL, "postgres://verificador:tres@bd/vec")
	ejecutor, proyector, verificador, err := Load().BolsaBorradoresPostgreSQL.DSNSeparados()
	if err != nil {
		t.Fatalf("configuracion completa rechazada: %v", err)
	}
	if ejecutor != "postgres://ejecutor:uno@bd/vec" ||
		proyector != "postgres://proyector:dos@bd/vec" ||
		verificador != "postgres://verificador:tres@bd/vec" {
		t.Fatal("Load altero o mezclo las conexiones separadas")
	}
}

func TestConfiguracionPostgreSQLBorradoresRedactaTodaRepresentacion(t *testing.T) {
	configuracion, err := NuevaConfiguracionPostgreSQLBorradores(
		"postgres://ejecutor:secreto-ejecutor@bd/vec",
		"postgres://proyector:secreto-proyector@bd/vec",
		"postgres://verificador:secreto-verificador@bd/vec",
	)
	if err != nil {
		t.Fatal(err)
	}
	serializada, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	representaciones := []string{
		fmt.Sprintf("%v", configuracion),
		fmt.Sprintf("%+v", configuracion),
		fmt.Sprintf("%#v", configuracion),
		string(serializada),
		configuracion.LogValue().String(),
		slog.AnyValue(configuracion).Resolve().String(),
	}
	configuracionGlobal := Config{BolsaBorradoresPostgreSQL: configuracion}
	globalJSON, err := json.Marshal(configuracionGlobal)
	if err != nil {
		t.Fatal(err)
	}
	representaciones = append(representaciones,
		fmt.Sprintf("%v", configuracionGlobal),
		fmt.Sprintf("%+v", configuracionGlobal),
		fmt.Sprintf("%#v", configuracionGlobal),
		string(globalJSON),
	)
	for _, representacion := range representaciones {
		for _, secreto := range []string{"secreto-ejecutor", "secreto-proyector", "secreto-verificador"} {
			if strings.Contains(representacion, secreto) {
				t.Fatalf("representacion filtro %q: %q", secreto, representacion)
			}
		}
	}
}
