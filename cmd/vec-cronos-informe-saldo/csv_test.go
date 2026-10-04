package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestCLISaldoCSVConservaMinutosYExigeEjemplo(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		args := []string{"--formato=csv", "../../web/static/textos/" + idioma + "/cronos-informe-saldo-csv.json", "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"}
		var salida bytes.Buffer
		if err := ejecutar(context.Background(), args, &salida); err != nil {
			t.Fatal(err)
		}
		filas, err := csv.NewReader(&salida).ReadAll()
		if err != nil || len(filas) != 2 || filas[1][5] != "-30" {
			t.Fatal("CSV negativo", err, filas)
		}
		for _, demo := range []string{`{"demo":false}`, `{"demo":true,"desconocido":1}`} {
			p := filepath.Join(t.TempDir(), "entrada.json")
			if err := os.WriteFile(p, []byte(demo), 0600); err != nil {
				t.Fatal(err)
			}
			salida.Reset()
			args[2] = p
			if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("entrada inválida produjo salida")
			}
		}
	}
}

func TestCLISaldoCSVRechazaOpcionesYCatalogoPDF(t *testing.T) {
	catalogo := "../../web/static/textos/es/cronos-informe-saldo-csv.json"
	fixture := "../../internal/modules/cronos/adapters/informesaldo/escenario_sintetico.json"
	for _, args := range [][]string{
		{"--formato=otro", catalogo, fixture},
		{"--formato=csv", "--vista=movimientos", catalogo, fixture},
		{"--formato=csv", "../../web/static/textos/es/cronos-informe-saldo.json", fixture},
		{"--formato=csv", catalogo},
		{catalogo, fixture},
	} {
		var salida bytes.Buffer
		if err := ejecutar(context.Background(), args, &salida); err == nil || salida.Len() != 0 {
			t.Fatal("opciones incompatibles aceptadas", args)
		}
	}
}
