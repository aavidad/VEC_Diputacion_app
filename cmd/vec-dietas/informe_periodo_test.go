package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const (
	csvTextosES = "../../web/static/textos/es/dietas-informes-csv.json"
	csvTextosEN = "../../web/static/textos/en/dietas-informes-csv.json"
	csvConfig   = "../../data/catalogos/dietas/informes-ejemplo-v1.json"
	csvDatos    = "../../data/demo/dietas/informes.json"
)

func ejecutarCSV(t *testing.T, args ...string) (int, string) {
	t.Helper()
	datos, err := os.ReadFile(csvDatos)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	codigo := ejecutarConArgumentos(append([]string{"--informe-periodo-csv"}, args...), bytes.NewReader(datos), &out)
	return codigo, out.String()
}

func TestInformePeriodoCSVCompletoYFiltrado(t *testing.T) {
	codigo, salida := ejecutarCSV(t, "--textos", csvTextosES, "--configuracion", csvConfig)
	lineas := strings.Split(strings.TrimSpace(salida), "\n")
	if codigo != 0 || len(lineas) != 9 || !strings.HasPrefix(lineas[0], "Referencia,Versión,Persona") ||
		!strings.HasPrefix(lineas[1], "DI-001,1,Ana Molina,Servicios Generales,Orientativo,02/09/2026,4250,1860,0,6110,") {
		t.Fatalf("%d %q", codigo, salida)
	}
	codigo, salida = ejecutarCSV(t, "--textos", csvTextosEN, "--configuracion", csvConfig,
		"--unidad", "unidad-demo-01", "--situacion", "liquidado", "--desde", "2026-09-01", "--hasta", "2026-09-30")
	lineas = strings.Split(strings.TrimSpace(salida), "\n")
	if codigo != 0 || len(lineas) != 2 || !strings.HasPrefix(lineas[0], "Reference,Version,Person") ||
		!strings.HasPrefix(lineas[1], "DI-007,2,María Ruiz,Servicios Generales,Settled,22/09/2026,4250,0,2100,6350,SYNTHETIC") {
		t.Fatalf("%d %q", codigo, salida)
	}
}

func TestInformePeriodoCSVFallaSinEscribirCSV(t *testing.T) {
	casos := map[string][]string{
		"sin configuracion":  {"--textos", csvTextosES},
		"argumento suelto":   {"--textos", csvTextosES, "--configuracion", csvConfig, "extra"},
		"periodo invertido":  {"--textos", csvTextosES, "--configuracion", csvConfig, "--desde", "2026-09-30", "--hasta", "2026-09-01"},
		"catalogo cambiado":  {"--textos", "../../web/static/textos/es/cronos-informe-saldo-csv.json", "--configuracion", csvConfig},
		"configuracion rota": {"--textos", csvTextosES, "--configuracion", csvTextosES},
	}
	for nombre, args := range casos {
		codigo, salida := ejecutarCSV(t, args...)
		if codigo != 2 || !strings.HasPrefix(salida, `{"codigo":`) || strings.Contains(salida, "DI-0") {
			t.Fatalf("%s: %d %q", nombre, codigo, salida)
		}
	}
}

func TestInformePeriodoCSVRechazaClavesConOtraCapitalizacionORepetidas(t *testing.T) {
	datos, err := os.ReadFile(csvDatos)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, alterado := range map[string]string{
		"mayusculas": strings.Replace(string(datos), `"persona_ref"`, `"PERSONA_REF"`, 1),
		"repetida":   strings.Replace(string(datos), `"persona": "Ana Molina"`, `"persona": "Ana Molina", "persona": "Otra"`, 1),
	} {
		if alterado == string(datos) {
			t.Fatalf("%s: la alteración no se aplicó", nombre)
		}
		var out bytes.Buffer
		codigo := ejecutarConArgumentos([]string{"--informe-periodo-csv", "--textos", csvTextosES, "--configuracion", csvConfig},
			strings.NewReader(alterado), &out)
		if codigo != 2 || strings.Contains(out.String(), "DI-0") {
			t.Fatalf("%s: %d %q", nombre, codigo, out.String())
		}
	}
}

func TestInformePeriodoPDFEsEnYFallos(t *testing.T) {
	datos, err := os.ReadFile(csvDatos)
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		var out bytes.Buffer
		codigo := ejecutarConArgumentos([]string{"--informe-periodo-pdf", "--textos", "../../web/static/textos/" + idioma + "/dietas-informes-pdf.json",
			"--configuracion", csvConfig, "--situacion", "liquidado"}, bytes.NewReader(datos), &out)
		if codigo != 0 || !strings.HasPrefix(out.String(), "%PDF-") {
			t.Fatalf("%s: %d %q", idioma, codigo, out.String()[:min(out.Len(), 80)])
		}
	}
	var out bytes.Buffer
	codigo := ejecutarConArgumentos([]string{"--informe-periodo-pdf", "--textos", csvTextosES, "--configuracion", csvConfig},
		bytes.NewReader(datos), &out)
	if codigo != 2 || !strings.HasPrefix(out.String(), `{"codigo":`) {
		t.Fatalf("catálogo CSV en PDF: %d %q", codigo, out.String())
	}
}
