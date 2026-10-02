package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/web"
)

func ejemploRevision(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("ejemplo.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func revisarCLI(t *testing.T, args []string, datos []byte) (int, salida) {
	t.Helper()
	var output bytes.Buffer
	codigo := ejecutar(args, bytes.NewReader(datos), &output)
	var s salida
	if err := json.Unmarshal(output.Bytes(), &s); err != nil {
		t.Fatalf("salida inválida: %v", err)
	}
	return codigo, s
}

func TestRevisionCLIFormatoExistenteYCatalogos(t *testing.T) {
	datos := ejemploRevision(t)
	codigo, es := revisarCLI(t, nil, datos)
	codigoEN, en := revisarCLI(t, []string{"--idioma", "en"}, datos)
	if codigo != 0 || codigoEN != 0 || !es.Informe.Valido || es.Informe.PaqueteHuellaSHA256 != en.Informe.PaqueteHuellaSHA256 {
		t.Fatal("revisión válida o huella por idioma")
	}
	if es.Mensajes["alcance"] == en.Mensajes["alcance"] || es.Mensajes["huella"] == "" {
		t.Fatal("traducción ausente")
	}
	catalogo, mensajes, err := web.CatalogoRevisionOrganizacion()
	if err != nil {
		t.Fatal(err)
	}
	for idioma, tabla := range mensajes {
		if len(tabla) != len(es.Mensajes) {
			t.Fatal("catálogos distintos")
		}
		for key, value := range tabla {
			if value == "" || catalogo.T(idioma, key) != value || es.Mensajes[key] == "" {
				t.Fatalf("clave %s", key)
			}
		}
	}
	for _, clave := range es.Informe.PendientesPublicacion {
		if es.Mensajes[clave] == "" {
			t.Fatalf("pendiente sin traducción: %s", clave)
		}
	}
}

func TestRevisionCLIRechazaAmbiguedad(t *testing.T) {
	datos := string(ejemploRevision(t))
	casos := map[string]string{
		"clave duplicada":          strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "tipo": "plantilla"`, 1),
		"clave escapada duplicada": strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "t\u0069po": "plantilla"`, 1),
		"clave desconocida":        strings.Replace(datos, `"tipo": "rpt"`, `"tipo": "rpt", "actor": "inventado"`, 1),
		"mayúsculas":               strings.Replace(datos, `"manifiesto"`, `"MANIFIESTO"`, 1),
		"anidada mayúsculas":       strings.Replace(datos, `"revision": 1`, `"Revision": 1`, 1),
		"nulo":                     strings.Replace(datos, `"vigente_desde": "2026-01-01"`, `"vigente_desde": null`, 1),
		"hechos nulos":             `{"manifiesto":{},"hechos":null}`,
		"tipo incorrecto":          strings.Replace(datos, `"revision": 1`, `"revision": "1"`, 1),
		"múltiples documentos":     datos + datos,
		"raíz array":               `[]`,
		"raíz nula":                `null`,
		"malformado":               `{`,
		"utf8 inválido":            datos + string([]byte{0xff}),
	}
	for nombre, entrada := range casos {
		t.Run(nombre, func(t *testing.T) {
			codigo, s := revisarCLI(t, nil, []byte(entrada))
			if codigo != 1 || s.Informe.Valido || s.Informe.ClaveError != "entrada_invalida" || s.Informe.PaqueteHuellaSHA256 != "" {
				t.Fatalf("informe: %+v", s.Informe)
			}
		})
	}
}

func TestRevisionCLIFilaYDecisiones(t *testing.T) {
	datos := ejemploRevision(t)
	var p map[string]json.RawMessage
	if err := json.Unmarshal(datos, &p); err != nil {
		t.Fatal(err)
	}
	p["decisiones"] = json.RawMessage(`[{"fila_fuente_ref":"fila:1","clase":"unidad","resultado":"pendiente","motivo":"Correspondencia sin acreditar","evidencia_ref":"evidencia:sintetica"}]`)
	conDecisiones, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if codigo, s := revisarCLI(t, nil, conDecisiones); codigo != 0 || s.Informe.Decisiones != 1 {
		t.Fatalf("decisiones: %+v", s.Informe)
	}
	invalido := bytes.Replace(datos, []byte(`"vigente_desde": "2026-01-01"`), []byte(`"vigente_desde": "2026-02-30"`), 1)
	codigo, s := revisarCLI(t, nil, invalido)
	if codigo != 1 || s.Informe.FilaFallida != 1 || s.Informe.ClaveError != "hecho_invalido" || s.Mensajes[s.Informe.ClaveError] == "" {
		t.Fatalf("fila: %+v", s.Informe)
	}
}

func TestRevisionCLILimitesArgumentosYSinEfectos(t *testing.T) {
	datos := ejemploRevision(t)
	copia := bytes.Clone(datos)
	for _, args := range [][]string{{"--idioma"}, {"--idioma", "xx"}, {"--idioma", "en", "otro"}, {"fichero.json"}} {
		codigo, s := revisarCLI(t, args, datos)
		if codigo != 1 || s.Informe.ClaveError != "argumentos_invalidos" {
			t.Fatal("argumentos admitidos")
		}
	}
	padded := append(bytes.Clone(datos), bytes.Repeat([]byte(" "), limiteEntrada-len(datos))...)
	if codigo, _ := revisarCLI(t, nil, padded); codigo != 0 {
		t.Fatal("límite exacto rechazado")
	}
	if codigo, s := revisarCLI(t, nil, append(padded, ' ')); codigo != 1 || s.Informe.ClaveError != "entrada_invalida" {
		t.Fatal("exceso admitido")
	}
	if !reflect.DeepEqual(datos, copia) {
		t.Fatal("entrada modificada")
	}
	if codigo := ejecutar(nil, lectorFallido{}, io.Discard); codigo != 1 {
		t.Fatal("fallo de lectura")
	}
	if codigo := ejecutar(nil, bytes.NewReader(datos), escritorFallido{}); codigo != 2 {
		t.Fatal("fallo de escritura")
	}
}

type lectorFallido struct{}

func (lectorFallido) Read([]byte) (int, error) { return 0, errors.New("lectura") }

type escritorFallido struct{}

func (escritorFallido) Write([]byte) (int, error) { return 0, errors.New("escritura") }
