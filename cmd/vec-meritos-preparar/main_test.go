package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/meritos/domain"
)

func ejemplo(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../data/ejemplos/meritos/preparacion.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLIEjemploExportaResultadoUtil(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(bytes.NewReader(ejemplo(t)), &out, &errOut); code != 0 {
		t.Fatal(code, errOut.String())
	}
	var resultado domain.Preparacion
	if err := json.Unmarshal(out.Bytes(), &resultado); err != nil {
		t.Fatal(err)
	}
	if len(resultado.Paquete.Hechos) != 5 || len(resultado.Revisiones) != 5 || resultado.Persistido || resultado.AcreditacionReal {
		t.Fatal("exportación incompleta o efectos afirmados")
	}
	if resultado.Paquete.Hechos[0].Estado != domain.Declarado || resultado.Paquete.Hechos[2].Estado != domain.Acreditado {
		t.Fatal("ha promovido o degradado estados aportados")
	}
}

func TestCLIRechazaJSONAmbiguoYSinFiltrarDatos(t *testing.T) {
	for _, input := range []string{
		`{"alcance":"real","alcance":"preparacion_sintetica"}`,
		`{"alcance":"real","\u0061lcance":"preparacion_sintetica"}`,
		`{"Alcance":"preparacion_sintetica"}`,
		`{"puntos":20}`,
		string(ejemplo(t)) + ` {}`,
		strings.Repeat(" ", maxBytes+1),
		`{"dato_personal":"dato-reservado-prueba"}`,
	} {
		var out, errOut bytes.Buffer
		if code := run(strings.NewReader(input), &out, &errOut); code != 1 || out.Len() != 0 || strings.Contains(errOut.String(), "dato-reservado-prueba") {
			t.Fatal("entrada inválida aceptada o expuesta", code)
		}
	}
}

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("test") }

func TestCLIFalloDeSalida(t *testing.T) {
	var errOut bytes.Buffer
	if run(bytes.NewReader(ejemplo(t)), escritorFallido{}, &errOut) != 1 || !strings.Contains(errOut.String(), "meritos.error.salida") {
		t.Fatal(errOut.String())
	}
}
