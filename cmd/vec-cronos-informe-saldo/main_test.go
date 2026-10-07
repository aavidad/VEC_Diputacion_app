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
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-")) || !bytes.Contains(salida.Bytes(), []byte("/Lang (es-ES)")) {
		t.Fatal(err)
	}
	args[0] = "../../web/static/textos/en/cronos-informe-saldo.json"
	salida.Reset()
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.Contains(salida.Bytes(), []byte("/Lang (en-GB)")) {
		t.Fatal("catálogo inglés no produjo el idioma pedido", err)
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

func TestCLICatalogoIdiomaInvalidoSinBytes(t *testing.T) {
	catalogo, err := os.ReadFile("../../web/static/textos/en/cronos-informe-saldo.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"en_GB", "en-gb", "", "en-GB) /OpenAction ("} {
		t.Run(idioma, func(t *testing.T) {
			ruta := t.TempDir() + "/catalogo.json"
			datos := bytes.Replace(catalogo, []byte(`"en-GB"`), []byte(`"`+idioma+`"`), 1)
			if err := os.WriteFile(ruta, datos, 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{ruta, "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("catálogo inválido produjo bytes", err)
			}
		})
	}
}
