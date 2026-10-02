package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
)

func argsComparacionExperiencia(t *testing.T) []string {
	t.Helper()
	args := []string{"--modo", "experiencia", "--comparar-reglas"}
	for i, nombre := range []string{"comparacion_experiencia_v1", "comparacion_experiencia_v2"} {
		ruta := filepath.Join("testdata", nombre+".json")
		b, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		flag := "--reglas"
		if i == 1 {
			flag = "--reglas-nuevas"
		}
		args = append(args, flag, ruta, flag+"-sha256", hex.EncodeToString(h[:]))
	}
	return args
}

func TestCLIComparacionExperienciaUsableYReproducible(t *testing.T) {
	args := argsComparacionExperiencia(t)
	var original []byte
	for i := 0; i < 2; i++ {
		var salida, diagnostico bytes.Buffer
		if c := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); c != 0 || diagnostico.Len() != 0 {
			t.Fatalf("%d %s", c, diagnostico.String())
		}
		var d struct {
			Esquema, Alcance string
			Cambios          []struct{ Campo string }
		}
		if err := json.Unmarshal(salida.Bytes(), &d); err != nil {
			t.Fatal(err)
		}
		if d.Esquema != "vec.bolsa.diferencia_reglas_experiencia.v1" || d.Alcance != "comparacion_reglas" || len(d.Cambios) != 10 {
			t.Fatal("salida incorrecta")
		}
		if i == 1 && !bytes.Equal(original, salida.Bytes()) {
			t.Fatal("salida no reproducible")
		}
		original = append([]byte(nil), salida.Bytes()...)
	}
	args[8], args[10] = args[4], args[6]
	var salida, diagnostico bytes.Buffer
	if c := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); c != 0 || !bytes.Contains(salida.Bytes(), []byte(`"cambios":[]`)) {
		t.Fatal("identico no es vacio")
	}
}

func TestCLIComparacionExperienciaLimitesEsquemasYDiagnosticos(t *testing.T) {
	for _, caso := range []string{"orden", "sha", "entrada", "modo", "archivo", "esquema", "exceso_4mib", "canonico"} {
		t.Run(caso, func(t *testing.T) {
			args := argsComparacionExperiencia(t)
			switch caso {
			case "orden":
				args[4], args[8], args[6], args[10] = args[8], args[4], args[10], args[6]
			case "sha":
				args[10] = strings.Repeat("0", 64)
			case "entrada":
				args = append(args, "--entrada", "ruta_privada_no_leer")
			case "modo":
				args[1] = "meritos"
			case "archivo":
				args[8] = filepath.Join(t.TempDir(), "ruta_privada_ausente")
			case "esquema":
				otros := argsComparacion(t)
				args[8], args[10] = otros[8], otros[10]
			case "exceso_4mib", "canonico":
				ruta := filepath.Join(t.TempDir(), "ruta_privada")
				b := make([]byte, 4*1024*1024+1)
				if caso == "canonico" {
					var err error
					b, err = os.ReadFile(args[8])
					if err != nil {
						t.Fatal(err)
					}
					b = append(b, '\n')
				}
				if err := os.WriteFile(ruta, b, 0600); err != nil {
					t.Fatal(err)
				}
				h := sha256.Sum256(b)
				args[8], args[10] = ruta, hex.EncodeToString(h[:])
			}
			var salida, diagnostico bytes.Buffer
			if c := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); c != 2 || salida.Len() != 0 || !json.Valid(diagnostico.Bytes()) {
				t.Fatalf("rechazo incorrecto: %d %s", c, diagnostico.String())
			}
			if strings.Contains(diagnostico.String(), "ruta_privada") {
				t.Fatal("diagnostico filtra ruta")
			}
		})
	}
}
