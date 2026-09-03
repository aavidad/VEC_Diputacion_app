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
	configuracion, err := NuevaConfiguracionPostgreSQLContratacionTemporal(ejecutor, gobierno)
	if err != nil {
		t.Fatal(err)
	}
	obtenidoEjecutor, obtenidoGobierno, err := configuracion.DSNSeparados()
	if err != nil || obtenidoEjecutor != ejecutor || obtenidoGobierno != gobierno {
		t.Fatalf("DSN separados = (%q, %q, %v)", obtenidoEjecutor, obtenidoGobierno, err)
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
	if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal("", "gobierno"); !errors.Is(
		err, ErrConfiguracionPostgreSQLContratacionTemporalIncompleta,
	) {
		t.Fatalf("configuracion incompleta: %v", err)
	}
	if _, err := NuevaConfiguracionPostgreSQLContratacionTemporal("misma", " misma "); !errors.Is(
		err, ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada,
	) {
		t.Fatalf("configuracion no separada: %v", err)
	}
}
