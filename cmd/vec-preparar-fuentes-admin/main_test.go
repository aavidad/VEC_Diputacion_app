package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func prepararFixture(t *testing.T) (string, string, string, []byte) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/fuente.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	fuente, destino := filepath.Join(dir, "fuente.json"), filepath.Join(dir, "plan.json")
	if err := os.WriteFile(fuente, b, 0600); err != nil {
		t.Fatal(err)
	}
	textos, err := filepath.Abs("../../web/static/textos/es/admin-fuentes-preparar.json")
	if err != nil {
		t.Fatal(err)
	}
	return fuente, destino, textos, b
}
func relojFixture() time.Time { return time.Date(2026, 10, 3, 12, 30, 0, 0, time.UTC) }
func TestCLIPrepararYCotejar(t *testing.T) {
	fuente, destino, textos, _ := prepararFixture(t)
	args := []string{"--fuente", fuente, "--plan", destino, "--textos", textos}
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) != 0 {
		t.Fatal(errores.String())
	}
	var d diagnostico
	if json.Unmarshal(salida.Bytes(), &d) != nil || !d.Preparado || d.Cotejado || len(d.HuellaPlanSHA256) != 64 {
		t.Fatal("diagnostico invalido")
	}
	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(destino)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("destino no privado")
	}
	var doc documento
	if decodificarEstricto(b, &doc) != nil || doc.HuellaPlanSHA256 != d.HuellaPlanSHA256 {
		t.Fatal("documento incoherente")
	}
	_, h, err := doc.Plan.CanonicoYHuella()
	if err != nil || h != doc.HuellaPlanSHA256 {
		t.Fatal("huella incoherente")
	}
	for _, cotejar := range []bool{false, true} {
		salida.Reset()
		errores.Reset()
		a := append([]string{}, args...)
		if cotejar {
			a = append(a, "--cotejar")
		}
		if ejecutar(a, &salida, &errores, relojFixture) != 0 {
			t.Fatal(errores.String())
		}
		actual, err := os.ReadFile(destino)
		if err != nil || !bytes.Equal(b, actual) {
			t.Fatal("reescribe plan")
		}
	}
	if strings.Contains(salida.String(), "persona_ref") || strings.Contains(salida.String(), "fuente_hmac") || strings.Contains(salida.String(), fuente) {
		t.Fatal("expone contenido")
	}
}
func TestCLIEntradaCerrada(t *testing.T) {
	casos := map[string]func([]byte) []byte{
		"duplicado": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version": 1`), []byte(`"version": 1, "version": 1`), 1)
		},
		"desconocido": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version": 1`), []byte(`"version": 1, "cuenta_ref": "secreto"`), 1)
		},
		"mayusculas": func(b []byte) []byte { return bytes.Replace(b, []byte(`"version": 1`), []byte(`"Version": 1`), 1) },
		"null":       func(b []byte) []byte { return bytes.Replace(b, []byte(`"version": 1`), []byte(`"version": null`), 1) },
		"incompleto": func(b []byte) []byte { return bytes.Replace(b, []byte(`"version": 1,`), nil, 1) },
		"tercera_persona": func(b []byte) []byte {
			var m map[string]any
			if json.Unmarshal(b, &m) != nil {
				panic("fixture")
			}
			m["personas"] = append(m["personas"].([]any), m["personas"].([]any)[0])
			r, _ := json.Marshal(m)
			return r
		},
		"secreto": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"fuente_hmac": {`), []byte(`"clave_hmac_hex": "secreto", "fuente_hmac": {`), 1)
		},
		"sobra":         func(b []byte) []byte { return append(b, []byte(`{}`)...) },
		"fraccion_cero": func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("12:00:00Z"), []byte("12:00:00.000Z")) },
		"offset_cero":   func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("12:00:00Z"), []byte("12:00:00+00:00")) },
		"overflow": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version": 1`), []byte(`"version": 18446744073709551616`), 1)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f, p, textos, b := prepararFixture(t)
			if err := os.WriteFile(f, cambiar(b), 0600); err != nil {
				t.Fatal(err)
			}
			var salida, errores bytes.Buffer
			if ejecutar([]string{"--fuente", f, "--plan", p, "--textos", textos}, &salida, &errores, relojFixture) == 0 || salida.Len() != 0 || strings.Contains(errores.String(), "secreto") {
				t.Fatal("entrada aceptada o expuesta")
			}
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Fatal("crea plan de entrada invalida")
			}
		})
	}
}
func TestCLICaducidadDivergenciaYDestinoSeguro(t *testing.T) {
	f, p, textos, b := prepararFixture(t)
	args := []string{"--fuente", f, "--plan", p, "--textos", textos}
	var salida, errores bytes.Buffer
	vencido := func() time.Time { return time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC) }
	if ejecutar(args, &salida, &errores, vencido) == 0 {
		t.Fatal("acepta caducado")
	}
	if err := os.WriteFile(p, []byte("no cambiar"), 0600); err != nil {
		t.Fatal(err)
	}
	if ejecutar(args, &salida, &errores, relojFixture) == 0 {
		t.Fatal("sobrescribe divergente")
	}
	previo, err := os.ReadFile(p)
	if err != nil || string(previo) != "no cambiar" {
		t.Fatal("reescrito")
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f, p); err != nil {
		t.Fatal(err)
	}
	if ejecutar(args, &salida, &errores, relojFixture) == 0 {
		t.Fatal("acepta enlace")
	}
	actual, err := os.ReadFile(f)
	if err != nil || !bytes.Equal(actual, b) {
		t.Fatal("fuente modificada")
	}
	if err := os.Chmod(f, 0644); err != nil {
		t.Fatal(err)
	}
	if ejecutar(args, &salida, &errores, relojFixture) == 0 {
		t.Fatal("acepta fuente publica")
	}
}
