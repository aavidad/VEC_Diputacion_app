package preparacionliquidacion

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func TestPrepararEjemploGastosD5(t *testing.T) {
	b, err := os.ReadFile("../../../../../cmd/vec-dietas/testdata/preparacion_liquidacion_gastos.json")
	if err != nil {
		t.Fatal(err)
	}
	var entrada Entrada
	if err := json.Unmarshal(b, &entrada); err != nil {
		t.Fatal(err)
	}
	p, err := Preparar(entrada)
	if err != nil {
		t.Fatal(err)
	}
	s := p.Instantanea()
	if s.Liquidable || s.Procedencia != "propuesta_sin_registrar" || s.Totales != (domain.TotalesLiquidacionPropuesta{OriginalCentimos: 6670, ReconocidoPropuestoCentimos: 5970, RechazadoCentimos: 700}) || len(s.Lineas) != 4 {
		t.Fatalf("resultado D5: %+v", s)
	}
	if s.Lineas[2].MotivoCodigo != "revision_justificante" || s.Lineas[3].MotivoCodigo != "gasto_no_admitido" || s.DocumentoSHA256 != entrada.DocumentoSHA256 {
		t.Fatal("motivos o justificantes del documento perdidos")
	}
	catDatos, err := os.ReadFile("../../../../../data/catalogos/dietas/liquidacion-gastos-ejemplo-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var cat domain.CatalogoLiquidacionPropuesta
	if err := json.Unmarshal(catDatos, &cat); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cat, entrada.Catalogo) || cat.CatalogoOtrosGastosVersionRef != domain.VersionCatalogoOtrosGastos {
		t.Fatal("catálogo D5 de ejemplo discordante")
	}
	h, err := domain.HuellaDatosLiquidacion(cat)
	if err != nil || h != s.CatalogoSHA256 {
		t.Fatal("huella del catálogo D5 discordante", err)
	}
}
