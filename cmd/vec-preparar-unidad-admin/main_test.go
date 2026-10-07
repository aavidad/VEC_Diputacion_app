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

func prepararFixture(t *testing.T) ([]string, string, string, []byte) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "privado")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directorio")
	}
	material, fuente, plan := filepath.Join(dir, "material.json"), filepath.Join(dir, "fuente.json"), filepath.Join(dir, "plan.json")
	b, e := os.ReadFile("testdata/material.sintetico.json")
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.ReadFile("testdata/fuente.sintetica.json")
	if e != nil {
		t.Fatal(e)
	}
	if os.WriteFile(material, b, 0600) != nil || os.WriteFile(fuente, f, 0600) != nil {
		t.Fatal("archivo")
	}
	textos, e := filepath.Abs("../../web/static/textos/es/admin-unidad-preparar.json")
	if e != nil {
		t.Fatal(e)
	}
	return []string{"--material", material, "--fuente", fuente, "--plan", plan, "--textos", textos}, plan, fuente, b
}
func relojFixture() time.Time { return time.Date(2026, 10, 4, 8, 30, 0, 0, time.UTC) }
func TestUnidadPrepararCotejarCanon(t *testing.T) {
	args, path, _, _ := prepararFixture(t)
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) != 0 {
		t.Fatal(errores.String())
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	info, e := os.Stat(path)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("plan público")
	}
	var doc documento
	if decodificarEstricto(b, &doc) != nil {
		t.Fatal("documento")
	}
	_, sha, e := doc.Plan.CanonicoYHuella()
	if e != nil || sha != doc.HuellaPlanSHA256 {
		t.Fatal("huella falsa")
	}
	for _, cotejar := range []bool{false, true} {
		a := append([]string{}, args...)
		if cotejar {
			a = append(a, "--cotejar")
		}
		salida.Reset()
		errores.Reset()
		if ejecutar(a, &salida, &errores, relojFixture) != 0 {
			t.Fatal(errores.String())
		}
		actual, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(b, actual) {
			t.Fatal("reescribe plan")
		}
	}
	if strings.Contains(salida.String(), "denominacion") || strings.Contains(salida.String(), "nodo_ref") || strings.Contains(salida.String(), path) {
		t.Fatal("expone fuente")
	}
}
func TestUnidadFuenteCruzadaYCerrada(t *testing.T) {
	casos := map[string]func([]byte) []byte{
		"denominacion_cambiada": func(b []byte) []byte {
			return bytes.Replace(b, []byte("Unidad sintética"), []byte("Otra unidad sintética"), 1)
		},
		"claves_repetidas": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1)
		},
		"campo_desconocido": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"perfil":"administrador"`), 1)
		},
		"fuente_self_sha": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"huella_sha256":"falsa"`), 1)
		},
		"version_null": func(b []byte) []byte { return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":null`), 1) },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			args, path, fuente, _ := prepararFixture(t)
			b, e := os.ReadFile(fuente)
			if e != nil {
				t.Fatal(e)
			}
			if os.WriteFile(fuente, cambiar(b), 0600) != nil {
				t.Fatal("fuente")
			}
			var salida, errores bytes.Buffer
			if ejecutar(args, &salida, &errores, relojFixture) == 0 || salida.Len() != 0 {
				t.Fatal("fuente inválida aceptada")
			}
			if _, e := os.Stat(path); !os.IsNotExist(e) {
				t.Fatal("plan de fuente falsa")
			}
		})
	}
}
func TestUnidadMaterialCerradoYCaducidad(t *testing.T) {
	args, path, _, b := prepararFixture(t)
	b = bytes.Replace(b, []byte(`"version": 1`), []byte(`"version": 1, "cuenta_ref": "secreto"`), 1)
	if os.WriteFile(args[1], b, 0600) != nil {
		t.Fatal("material")
	}
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) == 0 || strings.Contains(errores.String(), "secreto") {
		t.Fatal("entrada abierta o expuesta")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("crea plan de material abierto")
	}
	args, path, _, _ = prepararFixture(t)
	vencido := func() time.Time { return time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC) }
	if ejecutar(args, &salida, &errores, vencido) == 0 {
		t.Fatal("acepta caducado")
	}
	if os.WriteFile(path, []byte("no modificar"), 0600) != nil {
		t.Fatal("plan")
	}
	if ejecutar(args, &salida, &errores, relojFixture) == 0 {
		t.Fatal("sobrescribe divergente")
	}
	actual, e := os.ReadFile(path)
	if e != nil || string(actual) != "no modificar" {
		t.Fatal("plan reescrito")
	}
}
func TestUnidadCatalogoAusenteCodigoCerrado(t *testing.T) {
	args, path, _, _ := prepararFixture(t)
	args[7] = filepath.Join(filepath.Dir(path), "ausente.json")
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) != 2 || salida.Len() != 0 || errores.String() != "{\"codigo\":\"catalogo_no_disponible\"}\n" {
		t.Fatal("fallo catálogo silencioso o con ruta")
	}
}
func TestUnidadFuentePrettyMismoCanon(t *testing.T) {
	args, _, fuente, _ := prepararFixture(t)
	b, e := os.ReadFile(fuente)
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("fixture")
	}
	b, e = json.MarshalIndent(m, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if os.WriteFile(fuente, b, 0600) != nil {
		t.Fatal("archivo")
	}
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) != 0 {
		t.Fatal("formato confundido con contenido")
	}
}
