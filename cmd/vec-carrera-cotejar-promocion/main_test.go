package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCLICotejoESENDeterministaYHuellaLocal(t *testing.T) {
	b, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"es", "en"} {
		var a, otra, errs bytes.Buffer
		args := []string{"--catalogos", "../../web/static/textos", "--idioma", locale}
		if run(args, bytes.NewReader(b), &a, &errs) != 0 || run(args, bytes.NewReader(b), &otra, &errs) != 0 {
			t.Fatal(errs.String())
		}
		if !bytes.Equal(a.Bytes(), otra.Bytes()) || !strings.Contains(a.String(), `"estado_global": "pendiente"`) || !strings.Contains(a.String(), `"bases_verificadas": false`) {
			t.Fatal("exportación incoherente")
		}
		if !strings.Contains(a.String(), `"etiqueta_clave": "carrera.cotejo.dictamen_ensayo"`) || !strings.Contains(a.String(), `"carrera.cotejo.estado.no_cumple"`) {
			t.Fatal("renombra referencias técnicas del sobre")
		}
		if !strings.Contains(a.String(), map[string]string{"es": "Dictamen aportado de ensayo", "en": "Supplied trial opinion"}[locale]) {
			t.Fatal("no traduce límite")
		}
		var huella bytes.Buffer
		if run(append(args, "--huella-consulta"), bytes.NewReader(b), &huella, &errs) != 0 || !strings.Contains(huella.String(), `"huella_consulta_sha256"`) || strings.Contains(huella.String(), `"dictamen_aportado"`) {
			t.Fatal("huella local aparenta cotejo")
		}
	}
}
func TestCLIRechazaEntradaAmbiguaProveedorAusenteSinSalidaNiContenido(t *testing.T) {
	b, _ := os.ReadFile("testdata/entrada.json")
	var e map[string]any
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("fixture")
	}
	delete(e, "dictamen_sintetico")
	ausente, _ := json.Marshal(e)
	for _, in := range []string{string(ausente), `{"consulta":{},"consulta":{}}`, `{"CONSULTA":{}}`, `{"secreto":"dato-no-publicable"}`, `null`, `{} {}`, strings.Repeat(" ", 1<<20+1)} {
		var out, errs bytes.Buffer
		if run([]string{"--catalogos", "../../web/static/textos"}, strings.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-no-publicable") {
			t.Fatal("rechazo filtra contenido o da lista parcial")
		}
	}
}

func TestCLICatalogoConfigurableRechazaDuplicadosConcatenadosExcesoYFifo(t *testing.T) {
	in, _ := os.ReadFile("testdata/entrada.json")
	for _, cat := range []string{`{"clave":"dato-privado","clave":"otra"}`, `{"clave":"dato-privado"} {}`, `{"clave":"dato-privado"}` + strings.Repeat(" ", 64*1024), `null`} {
		dir := t.TempDir()
		if os.Mkdir(filepath.Join(dir, "es"), 0700) != nil {
			t.Fatal("dir")
		}
		if os.WriteFile(filepath.Join(dir, "es", "carrera-cotejo-promocion.json"), []byte(cat), 0600) != nil {
			t.Fatal("catálogo")
		}
		var out, errs bytes.Buffer
		if run([]string{"--catalogos", dir}, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-privado") || strings.Contains(errs.String(), dir) {
			t.Fatal("catálogo ambiguo produce salida o filtra dato")
		}
	}
	dir := t.TempDir()
	if os.Mkdir(filepath.Join(dir, "es"), 0700) != nil {
		t.Fatal("dir")
	}
	if syscall.Mkfifo(filepath.Join(dir, "es", "carrera-cotejo-promocion.json"), 0600) != nil {
		t.Fatal("fifo")
	}
	var out, errs bytes.Buffer
	if run([]string{"--catalogos", dir}, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 {
		t.Fatal("acepta catálogo especial")
	}
}
