package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogoAusenteEmiteCodigoCerrado(t *testing.T) {
	fuente, plan, _, _ := prepararFixture(t)
	ausente := filepath.Join(filepath.Dir(fuente), "catalogo-ausente-privado.json")
	var salida, errores bytes.Buffer
	relojInvocado := false
	reloj := func() time.Time { relojInvocado = true; return relojFixture() }
	codigo := ejecutar([]string{"--fuente", fuente, "--plan", plan, "--textos", ausente}, &salida, &errores, reloj)
	if codigo != 2 || salida.Len() != 0 || errores.String() != "{\"codigo\":\"catalogo_no_disponible\"}\n" {
		t.Fatal("diagnostico de catálogo ausente incorrecto")
	}
	if relojInvocado {
		t.Fatal("procesa fuente sin catálogo")
	}
	if _, err := os.Stat(plan); !os.IsNotExist(err) {
		t.Fatal("crea plan sin catálogo")
	}
}

type escrituraDiagnosticoFallida struct{}

func (escrituraDiagnosticoFallida) Write([]byte) (int, error) { return 0, errors.New("fallo_writer") }
func TestFalloEscrituraDiagnosticoCatalogo(t *testing.T) {
	if informarFalloCatalogo(escrituraDiagnosticoFallida{}, os.ErrNotExist) != 2 {
		t.Fatal("no propaga fallo de diagnóstico")
	}
}
