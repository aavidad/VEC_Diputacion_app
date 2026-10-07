package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/restauracioncopias"
)

func entradaFixture(t *testing.T) entrada {
	t.Helper()
	b, e := os.ReadFile("../vec-copias-comprobar/testdata/compatible.json")
	if e != nil {
		t.Fatal(e)
	}
	var x entrada
	if json.Unmarshal(b, &x) != nil {
		t.Fatal("json")
	}
	n := time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)
	x.Accion = "proponer"
	x.Ahora = n
	x.PersonaDeclaradaRef = "persona:1"
	x.Configuracion = d.Configuracion{Entorno: "sintetico_offline"}
	x.Propuesta = d.Propuesta{FormatoVersion: 1, Ref: "propuesta:1", ConjuntoRef: x.Manifiesto.ConjuntoRef, ConjuntoSHA256: d.Huella("vec-restauracion-conjunto-v1", x.Manifiesto), DestinoRef: x.Destino.Ref, PreimagenSHA256: strings.Repeat("a", 64), MotivoRef: "motivo:1", VentanaRef: "ventana:1", VentanaInicio: n, VentanaFin: n.Add(time.Hour), PoliticaRef: x.Politica.Ref, PoliticaSHA256: d.Huella("vec-restauracion-politica-v1", x.Politica), ProponentePersonaRef: "persona:1", Creada: n, Caduca: n.Add(time.Hour), DobleControl: true, Entorno: "sintetico_offline"}
	return x
}
func TestCLIProponerRevisarRecuperar(t *testing.T) {
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("chmod")
	}
	x := entradaFixture(t)
	call := func(x entrada) (int, salida) {
		t.Helper()
		b, e := json.Marshal(x)
		if e != nil {
			t.Fatal(e)
		}
		var out bytes.Buffer
		code := run([]string{"-registro", dir}, bytes.NewReader(b), &out)
		var s salida
		if json.Unmarshal(out.Bytes(), &s) != nil {
			t.Fatal(out.String())
		}
		if s.HabilitaRestauracion || s.Autorizacion != "no_comprobada" {
			t.Fatal(s)
		}
		return code, s
	}
	code, s := call(x)
	if code != 0 || s.Registro == nil || s.Registro.Version != 1 {
		t.Fatal(code, s)
	}
	x.SHA256 = s.Registro.Propuesta.SHA256
	x.Version = 1
	x.Accion = "revisar"
	if code, s = call(x); code != 2 || s.EstadoClave != "copias_restauracion_misma_persona" {
		t.Fatal(code, s)
	}
	x.PersonaDeclaradaRef = "persona:2"
	code, s = call(x)
	if code != 0 || s.Registro.Version != 2 {
		t.Fatal(code, s)
	}
	x.Accion = "consultar"
	code, s = call(x)
	if code != 0 || s.Registro.Version != 2 || s.Registro.Revision.PersonaRef != "persona:2" {
		t.Fatal(code, s)
	}
	x.Accion = "revisar"
	code, s = call(x)
	if code != 2 || s.EstadoClave != "copias_restauracion_cas_conflicto" {
		t.Fatal(code, s)
	}
}
func TestCLIEntradaCerrada(t *testing.T) {
	for _, b := range []string{`{}`, `{"accion":"proponer","accion":"revisar"}`, strings.Repeat(" ", maxEntradaBytes+1), `null`} {
		var out bytes.Buffer
		if code := run([]string{"-registro", t.TempDir()}, strings.NewReader(b), &out); code != 3 {
			t.Fatal(code)
		}
	}
}
func TestEjemploDocumentado(t *testing.T) {
	b, e := os.ReadFile("testdata/proponer.json")
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if code := run([]string{"-registro", dir}, bytes.NewReader(b), &out); code != 0 {
		t.Fatal(code, out.String())
	}
}
