package preparacionliquidacion

import (
	"encoding/json"
	"os"
	"testing"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func TestPrepararEjemploCatalogoImportado(t *testing.T) {
	b, err := os.ReadFile("../../../../../cmd/vec-dietas/testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	var e Entrada
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("fixture")
	}
	p, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	s := p.Instantanea()
	if s.Totales.OriginalCentimos != 4470 || s.Totales.ReconocidoPropuestoCentimos != 4470 || s.Liquidable || s.CatalogoSHA256 != "c764f00ddd96116a1181094761ec8e3936bf494497c72ac5b80c9859112410fd" || s.SnapshotSHA256 != "465cc4ada3eacb15b7dfb1ad566c42e6d84b35649cb6f76924f0bdc2a7a5193e" {
		t.Fatalf("%+v", s)
	}
	c, err := os.ReadFile("../../../../../data/catalogos/dietas/liquidacion-ejemplo-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var cat domain.CatalogoLiquidacionPropuesta
	if json.Unmarshal(c, &cat) != nil {
		t.Fatal("catalogo")
	}
	h, _ := domain.HuellaDatosLiquidacion(cat)
	if h != s.CatalogoSHA256 {
		t.Fatal("catalogo ejemplo discordante")
	}
	// Orden de las revisiones no cambia la instantánea de líneas en orden documental.
	e.Revisiones[0], e.Revisiones[1] = e.Revisiones[1], e.Revisiones[0]
	other, err := Preparar(e)
	if err != nil || other.Instantanea().SnapshotSHA256 != s.SnapshotSHA256 {
		t.Fatal("orden accidental de revision")
	}
	e.Revisiones[1].Indice = e.Revisiones[0].Indice
	if _, err := Preparar(e); err == nil {
		t.Fatal("revision duplicada")
	}
	e.Esquema = "otro"
	if _, err := Preparar(e); err == nil {
		t.Fatal("esquema ajeno")
	}
}

func TestKilometrajeRevisionLimiteRedondeo(t *testing.T) {
	b, err := os.ReadFile("../../../../../cmd/vec-dietas/testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	var e Entrada
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("fixture")
	}
	antes, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	e.Catalogo.Reglas[1].CentimosPorKM = 25
	if _, err := Preparar(e); err == nil {
		t.Fatal("tarifa rebajada no restringe propuesta")
	}
	e.Revisiones[1].ReconocidoPropuestoCentimos = 2500
	e.Revisiones[1].MotivoCodigo = "revision_tarifa_importada"
	despues, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	if despues.Instantanea().Totales.RechazadoCentimos != 100 || despues.Instantanea().Totales.ReconocidoPropuestoCentimos != 4370 || antes.Instantanea().Totales.ReconocidoPropuestoCentimos != 4470 {
		t.Fatal("pierde original o snapshot anterior")
	}
	e.Documento.Lineas[1].Kilometros = "0.0200"
	e.Documento.Lineas[1].KilometrosBase = "0.0200"
	e.Documento.Lineas[1].ImporteCentimos = 1
	e.Documento.KilometrajeCentimos = 1
	e.Documento.TotalOrientativoCentimos = 1871
	e.Revisiones[1].ReconocidoPropuestoCentimos = 1
	e.Revisiones[1].MotivoCodigo = ""
	e.DocumentoSHA256, _ = domain.HuellaDatosLiquidacion(e.Documento)
	if _, err := Preparar(e); err != nil {
		t.Fatal("medio centimo no redondeado arriba", err)
	}
	e.Documento.Lineas[1].Kilometros = "0.0199"
	e.Documento.Lineas[1].KilometrosBase = "0.0199"
	e.DocumentoSHA256, _ = domain.HuellaDatosLiquidacion(e.Documento)
	if _, err := Preparar(e); err == nil {
		t.Fatal("importe propuesto superior al redondeo")
	}
}
