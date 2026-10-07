package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/adapters/inventariocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func TestCLIConDatosSinteticos(t *testing.T) {
	var salida bytes.Buffer
	codigo := ejecutar([]string{"-descriptor", "testdata/descriptor.json", "-observado", "testdata/observado.json", "-raiz", "testdata/instalacion"}, &salida)
	var informe inventariocopias.Informe
	if err := json.Unmarshal(salida.Bytes(), &informe); err != nil {
		t.Fatal(err)
	}
	if codigo != 0 || informe.Resultado.Estado != copias.Compatible || informe.AutorizaCopia || informe.AutorizaRestauracion {
		t.Fatalf("codigo=%d salida=%s", codigo, salida.String())
	}
}

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) {
	return 0, errors.New("salida_sintetica_no_disponible")
}

func TestCLIPublicaCodigoDeFalloSiNoPuedeEscribir(t *testing.T) {
	for _, argumentos := range [][]string{nil, {"-descriptor", "testdata/descriptor.json", "-observado", "testdata/observado.json", "-raiz", "testdata/instalacion"}} {
		if codigo := ejecutar(argumentos, salidaFallida{}); codigo != 4 {
			t.Fatalf("codigo=%d", codigo)
		}
	}
}

func TestCLISinInventarioYSinErroresPrivados(t *testing.T) {
	var salida bytes.Buffer
	privado := filepath.Join(t.TempDir(), "ruta-no-publica")
	for _, argumentos := range [][]string{
		{},
		{"-descriptor", privado, "-observado", "testdata/observado.json", "-raiz", "testdata/instalacion"},
		{"-descriptor", "testdata/descriptor.json", "-observado", privado, "-raiz", "testdata/instalacion"},
		{"-descriptor", "testdata/descriptor.json", "-observado", "testdata/observado.json", "-raiz", "testdata/instalacion", "extra"},
	} {
		salida.Reset()
		if codigo := ejecutar(argumentos, &salida); codigo != 2 || !json.Valid(salida.Bytes()) || strings.Contains(salida.String(), privado) {
			t.Fatalf("codigo=%d salida=%s", codigo, salida.String())
		}
	}
	archivo := filepath.Join(t.TempDir(), "observado.json")
	if err := os.WriteFile(archivo, []byte(`{"completo":true,"completo":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	salida.Reset()
	if codigo := ejecutar([]string{"-descriptor", "testdata/descriptor.json", "-observado", archivo, "-raiz", "testdata/instalacion"}, &salida); codigo != 2 {
		t.Fatalf("codigo=%d salida=%s", codigo, salida.String())
	}
}

func TestCLIBloqueaMaterialAusente(t *testing.T) {
	var salida bytes.Buffer
	codigo := ejecutar([]string{"-descriptor", "testdata/descriptor.json", "-observado", "testdata/observado.json", "-raiz", t.TempDir()}, &salida)
	var informe inventariocopias.Informe
	if err := json.Unmarshal(salida.Bytes(), &informe); err != nil {
		t.Fatal(err)
	}
	if codigo != 1 || informe.Resultado.Estado != copias.NoComprobable || len(informe.Resultado.Razones) == 0 {
		t.Fatalf("codigo=%d salida=%s", codigo, salida.String())
	}
}
