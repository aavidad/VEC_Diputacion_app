package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
)

func argsPrueba(t *testing.T, reglas, entrada string) []string {
	t.Helper()
	base := filepath.Join("..", "..", "internal", "modules", "bolsa", "application", "simulacionbaremo", "testdata")
	huella := func(n string) string {
		v, e := os.ReadFile(filepath.Join(base, n+".sha256"))
		if e != nil {
			t.Fatal(e)
		}
		return strings.TrimSpace(string(v))
	}
	return []string{"--reglas", filepath.Join(base, reglas+".json"), "--reglas-sha256", huella(reglas), "--entrada", filepath.Join(base, entrada+".json"), "--entrada-sha256", huella(entrada)}
}
func TestCLIRecorreDosConvocatoriasYBloqueo(t *testing.T) {
	for _, caso := range []struct{ reglas, entrada, estado string }{{"reglas_a", "entrada", "completado"}, {"reglas_b", "entrada", "completado"}, {"reglas_a", "entrada_bloqueada", "bloqueado"}} {
		t.Run(caso.reglas+caso.entrada, func(t *testing.T) {
			var salida, diagnostico bytes.Buffer
			if codigo := ejecutar(argsPrueba(t, caso.reglas, caso.entrada), &salida, &diagnostico, simulacionbaremo.Servicio{}); codigo != 0 || diagnostico.Len() != 0 {
				t.Fatalf("codigo=%d diagnostico=%s", codigo, diagnostico.String())
			}
			var v struct {
				Alcance   string `json:"alcance"`
				Resultado struct {
					Estado string          `json:"estado"`
					Total  json.RawMessage `json:"total"`
				} `json:"resultado"`
			}
			if err := json.Unmarshal(salida.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
			if v.Alcance != "simulacion" || v.Resultado.Estado != caso.estado {
				t.Fatal("resultado equivoco")
			}
			if caso.estado == "bloqueado" && len(v.Resultado.Total) != 0 {
				t.Fatal("bloqueado con total")
			}
		})
	}
}
func TestCLIRechazaEntradasSinPublicarPuntuacion(t *testing.T) {
	for _, nombre := range []string{"sin_argumentos", "huella_incorrecta", "limite_excedido", "argumento_sobrante", "archivo_ausente"} {
		t.Run(nombre, func(t *testing.T) {
			args := argsPrueba(t, "reglas_a", "entrada")
			switch nombre {
			case "sin_argumentos":
				args = nil
			case "huella_incorrecta":
				args[3] = strings.Repeat("0", 64)
			case "limite_excedido":
				args = append(args, "--limite-bytes", "10")
			case "argumento_sobrante":
				args = append(args, "dato")
			case "archivo_ausente":
				args[1] = filepath.Join(t.TempDir(), "dato_privado_ausente")
			}
			var salida, diagnostico bytes.Buffer
			if codigo := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); codigo != 2 || salida.Len() != 0 || !json.Valid(diagnostico.Bytes()) {
				t.Fatalf("codigo=%d salida=%s diagnostico=%s", codigo, salida.String(), diagnostico.String())
			}
			if strings.Contains(diagnostico.String(), "dato_privado_ausente") {
				t.Fatal("ruta filtrada")
			}
		})
	}
}
func TestLecturaRechazaNoRegularesYLimitaBytes(t *testing.T) {
	dir := t.TempDir()
	fichero := filepath.Join(dir, "entrada")
	if err := os.WriteFile(fichero, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerArchivoLimitado(fichero, 4); err == nil {
		t.Fatal("sin limite")
	}
	if _, err := leerArchivoLimitado(dir, 100); err == nil {
		t.Fatal("directorio aceptado")
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerArchivoLimitado(fifo, 100); err == nil {
		t.Fatal("FIFO aceptado")
	}
	if v, err := leerArchivoLimitado(fichero, 5); err != nil || string(v) != "12345" {
		t.Fatalf("lectura=%q error=%v", v, err)
	}
}
