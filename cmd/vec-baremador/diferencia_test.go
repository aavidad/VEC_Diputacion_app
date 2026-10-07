package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/domain/calculomeritos"
)

func argsComparacion(t *testing.T) []string {
	t.Helper()
	args := []string{"--modo", "meritos", "--comparar-reglas"}
	for i, nombre := range []string{"comparacion_meritos_v1", "comparacion_meritos_v2"} {
		ruta := filepath.Join("testdata", nombre+".json")
		b, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		flag := "--reglas"
		if i == 1 {
			flag = "--reglas-nuevas"
		}
		args = append(args, flag, ruta, flag+"-sha256", calculomeritos.HuellaSHA256(b))
	}
	return args
}

func TestCLIComparaReglasSinEntradaNiPuntuacion(t *testing.T) {
	var salida, diagnostico bytes.Buffer
	args := argsComparacion(t)
	if codigo := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); codigo != 0 || diagnostico.Len() != 0 {
		t.Fatalf("%d %s", codigo, diagnostico.String())
	}
	var d calculomeritos.DiferenciaReglas
	if err := json.Unmarshal(salida.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.Alcance != "comparacion_reglas" || len(d.Cambios) != 7 || d.Anterior.Version != 1 || d.Nuevo.Version != 2 {
		t.Fatalf("comparacion inesperada: %+v", d)
	}
	if bytes.Contains(salida.Bytes(), []byte(`"total"`)) || bytes.Contains(salida.Bytes(), []byte(`"meritos"`)) {
		t.Fatal("comparacion calcula o publica meritos")
	}
	args[8], args[10] = args[4], args[6]
	salida.Reset()
	if codigo := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); codigo != 0 {
		t.Fatalf("identico %d: %s", codigo, diagnostico.String())
	}
	if !bytes.Contains(salida.Bytes(), []byte(`"cambios":[]`)) {
		t.Fatal("identico no devuelve cambios vacios")
	}
}

func TestCLIComparacionRechazaMezclasHuellasYFicherosSinFiltrarDatos(t *testing.T) {
	for _, caso := range []string{"entrada", "entrada_sha", "experiencia", "concursos", "ejemplo", "listar", "sin_accion", "falta_huella", "huella", "limite", "archivo", "posicional", "orden", "esquema"} {
		t.Run(caso, func(t *testing.T) {
			args := argsComparacion(t)
			switch caso {
			case "entrada":
				args = append(args, "--entrada", "ruta_privada_no_leer")
			case "entrada_sha":
				args = append(args, "--entrada-sha256", strings.Repeat("0", 64))
			case "experiencia", "concursos":
				args[1] = caso
			case "ejemplo":
				args = append(args, "--ejemplo", "ejemplo")
			case "listar":
				args = append(args, "--listar-ejemplos")
			case "sin_accion":
				args = append(args[:2], args[3:]...)
			case "falta_huella":
				args = args[:9]
			case "huella":
				args[10] = strings.Repeat("0", 64)
			case "limite":
				args = append(args, "--limite-bytes", "10")
			case "archivo":
				args[8] = filepath.Join(t.TempDir(), "ruta_privada_ausente")
			case "posicional":
				args = append(args, "resto")
			case "orden":
				args[4], args[8], args[6], args[10] = args[8], args[4], args[10], args[6]
			case "esquema":
				reglas := argsPrueba(t, "reglas_a", "entrada")
				args[8], args[10] = reglas[1], reglas[3]
			}
			var salida, diagnostico bytes.Buffer
			if codigo := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); codigo != 2 || salida.Len() != 0 || !json.Valid(diagnostico.Bytes()) {
				t.Fatalf("%d %s %s", codigo, salida.String(), diagnostico.String())
			}
			if strings.Contains(diagnostico.String(), "ruta_privada") {
				t.Fatal("diagnostico filtra ruta")
			}
		})
	}
}
