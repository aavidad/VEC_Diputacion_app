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

var argsExpediente = []string{"--expediente-sintetico", "../../data/catalogos/carrera/politica_grado_ejemplo.json", "testdata/antecedentes_expediente.json"}

func TestCLIExpedienteExportaRevisionDeterministaConContradiccionSolapeYFaltantes(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada_expediente.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, otra, errs bytes.Buffer
	if runArgs(argsExpediente, bytes.NewReader(in), &out, &errs) != 0 || runArgs(argsExpediente, bytes.NewReader(in), &otra, &errs) != 0 {
		t.Fatal(errs.String())
	}
	if !bytes.Equal(out.Bytes(), otra.Bytes()) || errs.Len() != 0 {
		t.Fatal("no es determinista")
	}
	for _, f := range []string{`"revision_expediente"`, `"nivel_puesto": 24`, `"grado_personal": 20`, `"estado_global": "pendiente"`, `"Contradicciones": [`, `"nivel_puesto"`, `carrera.motivo.periodos_solapados`, `autorizacion_carrera_h08`, `servicio:servicio-declarado`, `carrera.politica.pendiente.aprobacion_competente`, `"aprobacion_referencia": ""`} {
		if !strings.Contains(out.String(), f) {
			t.Fatalf("falta %s", f)
		}
	}
	var r struct {
		Preparacion struct {
			Casos []struct {
				Antecedentes struct {
					Periodos []any `json:"periodos"`
				} `json:"antecedentes"`
			} `json:"casos"`
		} `json:"preparacion"`
	}
	if json.Unmarshal(out.Bytes(), &r) != nil || len(r.Preparacion.Casos) != 1 || len(r.Preparacion.Casos[0].Antecedentes.Periodos) != 2 {
		t.Fatal("suma o importa periodos declarados")
	}
}

func TestCLIExpedienteRechazaDatosAmbiguosAjenoVersionYSinSalidaParcial(t *testing.T) {
	in, _ := os.ReadFile("testdata/entrada_expediente.json")
	snapshot, _ := os.ReadFile("testdata/antecedentes_expediente.json")
	for _, dato := range []string{`null`, `{} {}`, `{"instantaneas":[],"secreto":"dato-privado"}`, `{"instantaneas":[],"instantaneas":[]}`, strings.Repeat(" ", 1024*1024+1), strings.Replace(string(snapshot), "carrera-preparacion-1", "version-ajena", 1), strings.Replace(string(snapshot), "expediente-grado-1", "caso-ajeno", 1), strings.Replace(string(snapshot), "preparacion_sintetica", "produccion", 1)} {
		ruta := filepath.Join(t.TempDir(), "antecedentes.json")
		if os.WriteFile(ruta, []byte(dato), 0600) != nil {
			t.Fatal("escritura")
		}
		args := append([]string{}, argsExpediente...)
		args[2] = ruta
		var out, errs bytes.Buffer
		if runArgs(args, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-privado") || strings.Contains(errs.String(), ruta) {
			t.Fatal("datos inválidos dan salida o filtran contenido")
		}
	}
	fifo := filepath.Join(t.TempDir(), "fifo")
	if syscall.Mkfifo(fifo, 0600) != nil {
		t.Fatal("fifo")
	}
	for _, ruta := range []string{fifo, t.TempDir(), filepath.Join(t.TempDir(), "ausente")} {
		args := append([]string{}, argsExpediente...)
		args[2] = ruta
		var out, errs bytes.Buffer
		if runArgs(args, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 {
			t.Fatal("archivo especial o ausente aceptado")
		}
	}
}
