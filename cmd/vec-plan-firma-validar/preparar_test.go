package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func entradasPreparacionPrueba(t *testing.T) ([]string, []byte) {
	t.Helper()
	b, _ := materialPrueba(t)
	var m material
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	cfg, err := json.Marshal(configuracionPreparacion{m.Operacion, m.RevisionEsperada, m.HuellaEsperada, m.ClaveOperacion})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	args := []string{"preparar"}
	bloques := [][]byte{cfg}
	for _, codificado := range []string{m.CatalogoBase64, m.TrazaBase64, m.EventoBase64} {
		bloque, err := base64.StdEncoding.DecodeString(codificado)
		if err != nil {
			t.Fatal(err)
		}
		bloques = append(bloques, bloque)
	}
	for i, nombre := range []string{"config.json", "catalogo.json", "traza.json", "evento.json"} {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, bloques[i], 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, ruta)
	}
	return append(args, filepath.Join(dir, "material.json")), b
}

func TestPrepararConservaBytesDeterministasYValidadorHistorico(t *testing.T) {
	args, esperado := entradasPreparacionPrueba(t)
	var salida bytes.Buffer
	if codigo := ejecutar(args, &salida); codigo != 0 {
		t.Fatalf("preparar=%d", codigo)
	}
	var r resumen
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil || r.Estado != "preparado_sin_autorizacion" || r.SHA256 != huella(esperado) {
		t.Fatal("resumen de preparación incorrecto")
	}
	b, err := os.ReadFile(args[5])
	if err != nil || !bytes.Equal(b, esperado) {
		t.Fatal("el paquete no conservó exactamente la envoltura esperada")
	}
	info, err := os.Stat(args[5])
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("permisos de salida")
	}
	var m material
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for i, bloque := range []string{m.CatalogoBase64, m.TrazaBase64, m.EventoBase64} {
		conservado, err := base64.StdEncoding.DecodeString(bloque)
		original, errOriginal := os.ReadFile(args[i+2])
		if err != nil || errOriginal != nil || !bytes.Equal(conservado, original) {
			t.Fatal("componente canónico alterado")
		}
	}
	var validacion bytes.Buffer
	if codigo := ejecutar([]string{args[5], r.SHA256}, &validacion); codigo != 0 ||
		!strings.Contains(validacion.String(), "material_validado_sin_autorizacion") {
		t.Fatal("la invocación histórica no valida el paquete conservado")
	}
	args[5] += ".segunda"
	if codigo := ejecutar(args, &salida); codigo != 0 {
		t.Fatal("segunda preparación")
	}
	segunda, err := os.ReadFile(args[5])
	if err != nil || !bytes.Equal(b, segunda) {
		t.Fatal("preparación no determinista")
	}
	for _, privado := range []string{args[1], args[5], "actor:publicador:001", "clave-sintetica-0001", "catalogo_canonico_base64"} {
		if strings.Contains(salida.String(), privado) {
			t.Fatal("resumen expone entradas privadas")
		}
	}
}

func TestPrepararRechazaEntradasSinCrearSalida(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		indice int
		cambio func([]byte) []byte
	}{
		{"config_extra", 1, func(b []byte) []byte { return append(b[:len(b)-1], []byte(`,"actor":"inventado"}`)...) }},
		{"config_duplicada", 1, func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"operacion":`), []byte(`"operacion":"crear","operacion":`), 1)
		}},
		{"config_mayuscula", 1, func(b []byte) []byte { return bytes.Replace(b, []byte(`"operacion":`), []byte(`"Operacion":`), 1) }},
		{"revision_nula", 1, func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"revision_esperada":1`), []byte(`"revision_esperada":null`), 1)
		}},
		{"clave_nula", 1, func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"clave_operacion":"clave-sintetica-0001"`), []byte(`"clave_operacion":null`), 1)
		}},
		{"config_grande", 1, func([]byte) []byte { return bytes.Repeat([]byte{' '}, maxConfiguracionPreparacion+1) }},
		{"catalogo_no_canonico", 2, func(b []byte) []byte { return append(b, '\n') }},
		{"traza_no_canonica", 3, func(b []byte) []byte { return append(b, '\n') }},
		{"evento_no_canonico", 4, func(b []byte) []byte { return append(b, '\n') }},
		{"actor_cruzado", 3, func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("actor:publicador:001"), []byte("actor:otro:001"))
		}},
		{"revision_cruzada", 1, func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"revision_esperada":1`), []byte(`"revision_esperada":2`), 1)
		}},
		{"traza_grande", 3, func([]byte) []byte { return bytes.Repeat([]byte{'x'}, 65537) }},
		{"evento_grande", 4, func([]byte) []byte { return bytes.Repeat([]byte{'x'}, 65537) }},
		{"catalogo_grande", 2, func([]byte) []byte { return bytes.Repeat([]byte{'x'}, (2<<20)+1) }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			args, _ := entradasPreparacionPrueba(t)
			b, err := os.ReadFile(args[caso.indice])
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(args[caso.indice], caso.cambio(b), 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, &salida); codigo != 1 || salida.Len() != 0 {
				t.Fatalf("entrada inválida aceptada: %d", codigo)
			}
			if _, err := os.Lstat(args[5]); !os.IsNotExist(err) {
				t.Fatal("se creó salida con material rechazado")
			}
		})
	}
}
