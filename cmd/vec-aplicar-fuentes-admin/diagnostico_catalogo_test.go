package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogoAusenteEmiteCodigoCerradoSinDB(t *testing.T) {
	args, acuse := argsFixture(t, planFixture(t))
	args[9] = filepath.Join(filepath.Dir(acuse), "catalogo-ausente-privado.json")
	llamadas := 0
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
		llamadas++
		return nil, nil
	}
	var salida, errores bytes.Buffer
	codigo := ejecutar(args, &salida, &errores, abrir)
	if codigo != 2 || salida.Len() != 0 || errores.String() != "{\"codigo\":\"catalogo_no_disponible\"}\n" {
		t.Fatal("diagnostico de catálogo ausente incorrecto")
	}
	if llamadas != 0 {
		t.Fatal("abre DB sin catálogo")
	}
	if _, err := os.Stat(acuse); !os.IsNotExist(err) {
		t.Fatal("crea acuse sin catálogo")
	}
}

type escrituraDiagnosticoFallida struct{}

func (escrituraDiagnosticoFallida) Write([]byte) (int, error) { return 0, errors.New("fallo_writer") }
func TestFalloEscrituraDiagnosticoCatalogo(t *testing.T) {
	if informarFalloCatalogo(escrituraDiagnosticoFallida{}, os.ErrNotExist) != 2 {
		t.Fatal("no propaga fallo de diagnóstico")
	}
}
