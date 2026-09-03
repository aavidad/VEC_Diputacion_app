package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConfiguracionPostgreSQLContratacionTemporalSeparaYRedacta(t *testing.T) {
	ejecutor := "postgres://ejecutor:secreto-ejecucion@127.0.0.1/vec"
	gobierno := "postgres://gobierno:secreto-gobierno@127.0.0.1/vec"
	confirmador := "postgres://confirmador:secreto-confirmador@127.0.0.1/vec"
	lector := "postgres://lector:secreto-lector@127.0.0.1/vec"
	configuracion, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
		ejecutor,
		gobierno,
		confirmador,
		lector,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenidoEjecutor, obtenidoGobierno, err := configuracion.DSNSeparados()
	if err != nil || obtenidoEjecutor != ejecutor || obtenidoGobierno != gobierno {
		t.Fatalf("DSN separados = (%q, %q, %v)", obtenidoEjecutor, obtenidoGobierno, err)
	}
	obtenidoConfirmador, obtenidoLector, err := configuracion.DSNCoberturaSeparados()
	if err != nil || obtenidoConfirmador != confirmador || obtenidoLector != lector {
		t.Fatalf(
			"DSN de cobertura separados = (%q, %q, %v)",
			obtenidoConfirmador,
			obtenidoLector,
			err,
		)
	}
	jsonConfiguracion, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	for _, salida := range []string{
		fmt.Sprint(configuracion),
		fmt.Sprintf("%+v", configuracion),
		fmt.Sprintf("%#v", configuracion),
		string(jsonConfiguracion),
	} {
		if strings.Contains(salida, "secreto-") || !strings.Contains(salida, "redactada") {
			t.Fatalf("representacion no redactada: %q", salida)
		}
	}
}

func TestConfiguracionPostgreSQLContratacionTemporalFallaCerrado(t *testing.T) {
	completas := [4]string{"ejecucion", "gobierno", "confirmador", "lector"}
	for indice := range completas {
		dsn := completas
		dsn[indice] = "  "
		if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
			dsn[0],
			dsn[1],
			dsn[2],
			dsn[3],
		); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
			t.Fatalf("configuracion incompleta en posicion %d: %v", indice, err)
		}
	}
	for izquierda := 0; izquierda < len(completas); izquierda++ {
		for derecha := izquierda + 1; derecha < len(completas); derecha++ {
			dsn := completas
			dsn[derecha] = " " + dsn[izquierda] + " "
			if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal(
				dsn[0],
				dsn[1],
				dsn[2],
				dsn[3],
			); !errors.Is(err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada) {
				t.Fatalf("configuracion no separada entre %d y %d: %v", izquierda, derecha, err)
			}
		}
	}
}

func TestLoadCargaCuatroConexionesPostgreSQLContratacionTemporal(t *testing.T) {
	dsn := [4]string{
		"postgres://ejecucion@127.0.0.1/vec",
		"postgres://gobierno@127.0.0.1/vec",
		"postgres://confirmador@127.0.0.1/vec",
		"postgres://lector@127.0.0.1/vec",
	}
	t.Setenv(EnvContratacionTemporalDatabaseURL, " "+dsn[0]+" ")
	t.Setenv(EnvContratacionTemporalGobiernoDatabaseURL, " "+dsn[1]+" ")
	t.Setenv(EnvContratacionTemporalConfirmadorDatabaseURL, " "+dsn[2]+" ")
	t.Setenv(EnvContratacionTemporalLectorResultadoDatabaseURL, " "+dsn[3]+" ")

	configuracion := Load().ContratacionTemporalPostgreSQL
	ejecutor, gobierno, err := configuracion.DSNSeparados()
	if err != nil || ejecutor != dsn[0] || gobierno != dsn[1] {
		t.Fatalf("conexiones base = (%q, %q, %v)", ejecutor, gobierno, err)
	}
	confirmador, lector, err := configuracion.DSNCoberturaSeparados()
	if err != nil || confirmador != dsn[2] || lector != dsn[3] {
		t.Fatalf("conexiones de cobertura = (%q, %q, %v)", confirmador, lector, err)
	}
}
