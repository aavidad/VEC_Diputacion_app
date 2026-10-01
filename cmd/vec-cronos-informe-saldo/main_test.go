package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestCLISoloEscenarioSinteticoEIdiomaReal(t *testing.T) {
	args := []string{"../../web/static/textos/es/cronos-informe-saldo.json", "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-")) {
		t.Fatal(err)
	}
	args[0] = "../../web/static/textos/en/cronos-informe-saldo.json"
	salida.Reset()
	if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("idioma discordante publico bytes")
	}
	fixture := filepath.Join(t.TempDir(), "sin-demo.json")
	if err := os.WriteFile(fixture, []byte(`{"demo":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	args[0] = "../../web/static/textos/es/cronos-informe-saldo.json"
	args[1] = fixture
	salida.Reset()
	if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("datos sin marca sintetica aceptados")
	}
}

func TestCLIErrorEmiteCodigoSinMensajeOriginal(t *testing.T) {
	casos := []struct {
		fallo  error
		codigo string
	}{
		{fmt.Errorf("detalle_sensible_sintetico: %w", ports.ErrExportacionSaldoInvalida), "cronos_exportacion_saldo_invalida"},
		{fmt.Errorf("detalle_sensible_sintetico: %w", context.Canceled), "cronos_exportacion_saldo_cancelada"},
		{fmt.Errorf("detalle_sensible_sintetico: %w", context.DeadlineExceeded), "cronos_exportacion_saldo_tiempo_agotado"},
		{fmt.Errorf("detalle_sensible_sintetico"), "cronos_exportacion_saldo_no_disponible"},
	}
	for _, c := range casos {
		var salida bytes.Buffer
		if err := informarError(&salida, c.fallo); err != nil {
			t.Fatal(err)
		}
		if salida.String() != c.codigo+"\n" {
			t.Fatalf("stderr: %q", salida.String())
		}
	}
}
