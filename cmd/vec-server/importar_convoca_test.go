package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	importacionconvoca "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestLeerArgumentosImportarConvoca(t *testing.T) {
	a, e := leerArgumentosImportarConvoca([]string{"--fichero", "entrada.xls", "--categoria", "administrativo", "--bolsa-ref", "bolsa:administrativo:2026-09-18", "--admitir-rechazos"}, &bytes.Buffer{})
	if e != nil || a.fichero != "entrada.xls" || a.categoria != "administrativo" || a.bolsaRef == "" || !a.admitirRechazos {
		t.Fatalf("%#v %v", a, e)
	}
}
func TestLeerArgumentosImportarConvocaRechazaObligatorios(t *testing.T) {
	if _, e := leerArgumentosImportarConvoca([]string{"--fichero", "entrada.xls"}, &bytes.Buffer{}); e == nil {
		t.Fatal("acepto categoria ausente")
	}
}

func TestDescribirSustituidas(t *testing.T) {
	if got := describirSustituidas(nil); got != "ninguna" {
		t.Fatalf("sin sustituidas: %q", got)
	}
	got := describirSustituidas([]ports.BolsaSustituida{{BolsaRef: "bolsa:a", VersionBolsa: 1}, {BolsaRef: "bolsa:b", VersionBolsa: 3}})
	if got != "bolsa:a@1,bolsa:b@3" {
		t.Fatalf("sustituidas: %q", got)
	}
}

func TestEscribirReciboImportacionConvocaEmiteJSONDeAltaYReplay(t *testing.T) {
	huella := strings.Repeat("a", 64)
	categoria := "categoria:rpt:administrativo"
	referencia := dominio.ReferenciaContexto(huella, categoria)
	acta := dominio.ActaImportacion{
		CategoriaRef: categoria, BolsaRef: "bolsa:administrativo:2026-09-18",
		ActaRef:             "acta:importacion-convoca:" + referencia,
		ImportacionRef:      "importacion:convoca:" + referencia,
		HuellaFicheroSHA256: huella, FicheroCustodiadoRef: "almacen:objeto:convoca:" + huella,
		NombreFichero: "sintetico.xls", ActorRef: "actor:rrhh:prueba",
		RegistradaEn: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC),
		Esquema:      dominio.EsquemaResumenPersona, FilasLeidas: 1, FilasAceptadas: 1,
		Procedencia: dominio.NuevaProcedenciaNoAutoritativa(),
	}
	for _, caso := range []struct {
		reutilizada bool
		estado      importacionconvoca.EstadoReciboImportacion
	}{{false, importacionconvoca.EstadoReciboNueva}, {true, importacionconvoca.EstadoReciboReutilizada}} {
		var salida bytes.Buffer
		recibo, err := escribirReciboImportacionConvoca(&salida, importacionconvoca.ResultadoImportacion{Acta: acta, Reutilizada: caso.reutilizada})
		if err != nil || recibo.Estado != caso.estado || recibo.RegistradaEn != acta.RegistradaEn {
			t.Fatalf("recibo %q: %+v %v", caso.estado, recibo, err)
		}
		var publicado map[string]any
		if err := json.Unmarshal(salida.Bytes(), &publicado); err != nil {
			t.Fatalf("salida no es JSON: %v", err)
		}
		if publicado["estado"] != string(caso.estado) || publicado["acta_ref"] != acta.ActaRef || publicado["autoridad"] != "no_autoritativa" {
			t.Fatalf("salida sin recibo esperado: %#v", publicado)
		}
		for _, sensible := range []string{acta.NombreFichero, acta.FicheroCustodiadoRef, acta.ActorRef} {
			if strings.Contains(salida.String(), sensible) {
				t.Fatalf("salida revela %q", sensible)
			}
		}
	}
}

func TestEscribirReciboImportacionConvocaFallaCerrado(t *testing.T) {
	var salida bytes.Buffer
	if _, err := escribirReciboImportacionConvoca(&salida, importacionconvoca.ResultadoImportacion{}); err == nil || salida.Len() != 0 {
		t.Fatalf("acta invalida produjo salida: %q, %v", salida.String(), err)
	}
}
