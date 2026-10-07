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

func TestCLIExportaPreparacionDeterminista(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	var a, b, errs bytes.Buffer
	if run(bytes.NewReader(in), &a, &errs) != 0 || run(bytes.NewReader(in), &b, &errs) != 0 {
		t.Fatal(errs.String())
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) || errs.Len() != 0 || !strings.Contains(a.String(), `"estado_global": "pendiente"`) || strings.Contains(a.String(), `"elegible"`) {
		t.Fatal("exportacion incoherente")
	}
	var salida struct {
		Casos []struct {
			Antecedentes struct {
				Convocatoria struct{ Referencia, Fuente, Version string }
				Periodos     []struct{ Inicio, Fin string }
			}
		}
	}
	if err := json.Unmarshal(a.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if len(salida.Casos) != 3 || salida.Casos[2].Antecedentes.Convocatoria.Referencia != "convocatoria-sintetica-modulo-selectivos-A" ||
		salida.Casos[2].Antecedentes.Convocatoria.Fuente != "fuente-sintetica-carrera" || salida.Casos[2].Antecedentes.Convocatoria.Version != "ensayo-1" ||
		len(salida.Casos[1].Antecedentes.Periodos) != 2 {
		t.Fatal("exportacion pierde antecedentes")
	}
}
func TestCLIFallaSinSalidaParcialNiContenidoDeEntrada(t *testing.T) {
	for _, in := range []string{`{"alcance":"preparacion_sintetica","alcance":"produccion"}`, `{"secreto":"dato-no-publicable"}`, `{} {}`, `null`, strings.Repeat(" ", 1024*1024+1)} {
		var out, errs bytes.Buffer
		if run(strings.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-no-publicable") {
			t.Fatalf("rechazo incorrecto: %s", errs.String())
		}
	}
}
func TestCLIRechazaMiembrosConMayusculas(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	in = bytes.Replace(in, []byte(`"alcance": "preparacion_sintetica"`),
		[]byte(`"alcance": "preparacion_sintetica", "ALCANCE": "preparacion_sintetica"`), 1)
	var out, errs bytes.Buffer
	if run(bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || !strings.Contains(errs.String(), "carrera.error.json_invalido") {
		t.Fatal("permite claves equivalentes por plegado de mayusculas")
	}
}

func TestCLIAntecedentesSinteticosConservaDeclaracionYFaltantes(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if runArgs([]string{"--antecedentes-sinteticos"}, bytes.NewReader(in), &out, &errs) != 0 {
		t.Fatal(errs.String())
	}
	var resultado struct {
		Preparacion struct {
			Casos []struct {
				EstadoGlobal               string `json:"estado_global"`
				GradoPersonal, NivelPuesto *int   `json:"-"`
			} `json:"casos"`
		} `json:"preparacion"`
	}
	if json.Unmarshal(out.Bytes(), &resultado) != nil || len(resultado.Preparacion.Casos) != 3 || resultado.Preparacion.Casos[0].EstadoGlobal != "pendiente" {
		t.Fatal("no prepara los casos existentes")
	}
	for _, fragmento := range []string{`"antecedentes_sinteticos"`, `"autorizacion_carrera_h08"`, `"lector_autorizado_personal"`, `"Version": "carrera-preparacion-1"`, `"Estado": "declarado"`, `"grado_personal": null`, `"nivel_puesto": null`} {
		if !strings.Contains(out.String(), fragmento) {
			t.Fatalf("falta %s", fragmento)
		}
	}
}

func TestCLIAntecedentesRechazaEntradaYArgumentosSinDatos(t *testing.T) {
	for _, args := range [][]string{{"--antecedentes-sinteticos"}, {"--produccion"}, {"--antecedentes-sinteticos", "ruta-privada"}} {
		var out, errs bytes.Buffer
		if runArgs(args, strings.NewReader(`{"secreto":"dato-privado"}`), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-privado") || strings.Contains(errs.String(), "ruta-privada") {
			t.Fatal("rechazo filtra datos o devuelve preparación parcial")
		}
	}
}

func TestCLIRevisaCatalogoGradoJuntoConPreparacion(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if runArgs([]string{"--politica-grado-sintetica", "../../data/catalogos/carrera/politica_grado_ejemplo.json"}, bytes.NewReader(in), &out, &errs) != 0 {
		t.Fatal(errs.String())
	}
	for _, s := range []string{`"politica_grado_sintetica"`, `"borrador-1"`, `"ensayo-2"`, `"carrera.politica.pendiente.periodos"`, `"carrera.politica.pendiente.aprobacion_competente"`, `"estado_global": "pendiente"`, `"aprobacion_referencia": ""`} {
		if !strings.Contains(out.String(), s) {
			t.Fatalf("falta %s", s)
		}
	}
}

func TestCLIPoliticaRechazaAmbiguedadTamanioYFuenteNoRegularSinFiltrar(t *testing.T) {
	in, err := os.ReadFile("testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, catalogo := range []string{`{"alcance":"preparacion_sintetica","alcance":"produccion"}`, `{"alcance":"preparacion_sintetica","ALCANCE":"produccion"}`, `{"alcance":"preparacion_sintetica","aprobada":true}`, `{} {}`, strings.Repeat(" ", 1024*1024+1), `{"secreto":"dato-privado"}`} {
		ruta := filepath.Join(t.TempDir(), "catalogo.json")
		if err := os.WriteFile(ruta, []byte(catalogo), 0600); err != nil {
			t.Fatal(err)
		}
		var out, errs bytes.Buffer
		if runArgs([]string{"--politica-grado-sintetica", ruta}, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), "dato-privado") || strings.Contains(errs.String(), ruta) {
			t.Fatal("entrada ambigua produce salida o filtra datos")
		}
	}
	for _, ruta := range []string{t.TempDir(), filepath.Join(t.TempDir(), "no-existe")} {
		var out, errs bytes.Buffer
		if runArgs([]string{"--politica-grado-sintetica", ruta}, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), ruta) {
			t.Fatal("fuente no regular produce preparación o filtra ruta")
		}
	}
	fifo := filepath.Join(t.TempDir(), "catalogo-fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	if runArgs([]string{"--politica-grado-sintetica", fifo}, bytes.NewReader(in), &out, &errs) != 1 || out.Len() != 0 || strings.Contains(errs.String(), fifo) {
		t.Fatal("un fichero especial bloquea o expone la ruta")
	}
}
