package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
)

const catalogo = "../../web/static/textos/es/copias-operacion.json"

func fixture(t *testing.T) entrada {
	t.Helper()
	b, err := os.ReadFile("testdata/doble-ensayo.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var e entrada
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func correr(t *testing.T, e entrada) (int, string, string) {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	code := ejecutar([]string{"-textos", catalogo}, bytes.NewReader(b), &out, &diag)
	return code, out.String(), diag.String()
}

func TestCLIUsableDeclaradoNuncaHabilitaEfectos(t *testing.T) {
	e := fixture(t)
	for n := 0; n <= len(e.Comandos); n++ {
		partial := e
		partial.Comandos = e.Comandos[:n]
		code, out, diag := correr(t, partial)
		if code != 0 {
			t.Fatal(diag)
		}
		var r struct {
			Alcance      string `json:"alcance"`
			Autenticidad string `json:"autenticidad"`
			Estado       string `json:"estado"`
			Copia        bool   `json:"habilita_copia"`
			Restore      bool   `json:"habilita_restauracion"`
			Durable      bool   `json:"registro_durable"`
			Version      int    `json:"version"`
		}
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatal(err)
		}
		if r.Alcance != "simulacion_offline" || r.Autenticidad != "no_comprobada" || r.Copia || r.Restore || r.Durable || r.Version != n {
			t.Fatal(out)
		}
		if (r.Estado == "verificada_declarada") != (n == 5) {
			t.Fatal(out)
		}
		if strings.Contains(out, e.Solicitud.Destino) || strings.Contains(out, e.Solicitud.Clave) {
			t.Fatal("input references leaked")
		}
	}
}

func TestCLIRecuperaHistoriaYReplay(t *testing.T) {
	e := fixture(t)
	o, err := operacionescopias.Nueva(e.Solicitud)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range e.Comandos[:4] {
		o, _, _, err = o.Aplicar(c)
		if err != nil {
			t.Fatal(err)
		}
	}
	e.Historia = o.Historia()
	e.Comandos = []operacionescopias.Comando{e.Comandos[3], e.Comandos[4]}
	code, out, diag := correr(t, e)
	if code != 0 || !strings.Contains(out, `"replay":true`) || !strings.Contains(out, `"version":5`) {
		t.Fatal(out, diag)
	}
	e.Historia[1].Version++
	code, out, diag = correr(t, e)
	if code == 0 || out != "" || !strings.Contains(diag, "operacion_historia_invalida") {
		t.Fatal(out, diag)
	}
}

func TestCLIDiagnosticosNoReflejanContenido(t *testing.T) {
	for _, alterar := range []func(*entrada){
		func(e *entrada) { e.Sintetica = false },
		func(e *entrada) { e.Solicitud.Operacion = "secret/private/value" },
		func(e *entrada) { e.Comandos[0].VersionEsperada = 99 },
		func(e *entrada) { e.Comandos[4].Clave = e.Comandos[3].Clave },
	} {
		e := fixture(t)
		alterar(&e)
		code, out, diag := correr(t, e)
		if code == 0 || out != "" || strings.Contains(diag, "secret/private/value") {
			t.Fatal(out, diag)
		}
	}
	for _, in := range []string{`{"sintetica":true,"secreto":"no_imprimir"}`, `{} {}`, strings.Repeat(" ", maxBytes+1)} {
		var out, diag bytes.Buffer
		if ejecutar([]string{"-textos", catalogo}, strings.NewReader(in), &out, &diag) == 0 || strings.Contains(diag.String(), "no_imprimir") {
			t.Fatal(diag.String())
		}
	}
}

func TestCatalogosConTraductorReal(t *testing.T) {
	for _, ruta := range []string{catalogo, "../../web/static/textos/en/copias-operacion.json"} {
		c, err := cargarCatalogo(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if c.T(c.DefaultLocale(), "aviso_simulacion") == "aviso_simulacion" {
			t.Fatal("missing translation")
		}
	}
}

func TestCLIRechazaClavesDuplicadas(t *testing.T) {
	for _, in := range []string{
		`{"sintetica":false,"sintetica":true}`,
		`{"comandos":[{"version_esperada":1,"version_esperada":2}]}`,
		`{"historia":[{"comando":{"evidencia":{"resultado":"fallido","resultado":"satisfactorio"}}}]}`,
	} {
		var out, diag bytes.Buffer
		if ejecutar([]string{"-textos", catalogo}, strings.NewReader(in), &out, &diag) != 2 || out.Len() != 0 || !strings.Contains(diag.String(), `"error":"entrada_invalida"`) {
			t.Fatal(out.String(), diag.String())
		}
		// Verify lexical rejection independently of missing required fields.
		var target any
		if err := decodificar(strings.NewReader(in), &target); err == nil {
			t.Fatal("duplicate key accepted")
		}
	}
}

func TestParserRechazaAliasDeMayusculas(t *testing.T) {
	for _, in := range []string{
		`{"Sintetica":true}`,
		`{"sintetica":false,"SINTETICA":true}`,
		`{"comandos":[{"Version_Esperada":1}]}`,
		`{"ſintetica":true}`,
	} {
		var target any
		if err := decodificar(strings.NewReader(in), &target); err == nil {
			t.Fatal("case alias accepted")
		}
	}
	// The same key is valid in separate objects; values may contain capitals.
	var target any
	if err := decodificar(strings.NewReader(`[{"clave":"Valor"},{"clave":"Otro"}]`), &target); err != nil {
		t.Fatal(err)
	}
}
