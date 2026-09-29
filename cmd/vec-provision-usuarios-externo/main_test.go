package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestCLIProvisionUsuariosRechazaEntradaAntesDeLeerDSN(t *testing.T) {
	for _, args := range [][]string{nil, {"-plan", "relativo.json"}, {"-plan", "/inexistente-plan-sintetico.json"}, {"-desconocido"}} {
		var out bytes.Buffer
		codigo := ejecutar(args, &out, func(string) string { t.Fatal("se leyó DSN para entrada inválida"); return "" })
		if codigo != 2 || !strings.Contains(out.String(), "entrada_invalida") {
			t.Fatalf("salida incorrecta: %d %s", codigo, out.String())
		}
	}
}

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) { return 0, errors.New("salida interrumpida") }

func TestCLIProvisionUsuariosPropagaFalloDeSalida(t *testing.T) {
	if codigo := fallo(salidaFallida{}, "entrada_invalida", 2); codigo != 1 {
		t.Fatalf("fallo de salida oculto: %d", codigo)
	}
}

func TestCLIProvisionUsuariosTLSVerificaServidorSinFallbackPlano(t *testing.T) {
	for _, modo := range []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		cfg, err := pgx.ParseConfig("host=localhost user=sintetico dbname=sintetica sslmode=" + modo)
		if err != nil {
			t.Fatal(err)
		}
		if tlsVerificado(cfg) != (modo == "verify-full") {
			t.Fatalf("modo TLS %s admitido incorrectamente", modo)
		}
	}
}
