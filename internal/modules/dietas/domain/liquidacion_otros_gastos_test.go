package domain

import (
	"reflect"
	"strings"
	"testing"
)

func datosLiquidacionConGastos() (DocumentoComision, CatalogoLiquidacionPropuesta, []RevisionLineaLiquidacion) {
	d, c, revisiones := datosLiquidacion()
	trenRef, trenSHA := "justificante:tren:sintetico", strings.Repeat("a", 64)
	peajeRef, peajeSHA := "justificante:peaje:sintetico", strings.Repeat("b", 64)
	d.Lineas = append(d.Lineas,
		LineaDocumentoComision{Tipo: ClaseOtroMedio, Fecha: "2026-10-01", Concepto: "Billete de tren sintético", ImporteCentimos: 1800, JustificanteRef: &trenRef, JustificanteSHA256: &trenSHA, TipoGasto: "tren", CatalogoVersion: VersionCatalogoOtrosGastos},
		LineaDocumentoComision{Tipo: ClaseOtroGasto, Fecha: "2026-10-01", Concepto: "Peaje sintético", ImporteCentimos: 400, JustificanteRef: &peajeRef, JustificanteSHA256: &peajeSHA, TipoGasto: "peaje", CatalogoVersion: VersionCatalogoOtrosGastos},
	)
	d.OtrosCentimos = 2200
	d.TotalOrientativoCentimos = 4070
	c.CatalogoOtrosGastosVersionRef = VersionCatalogoOtrosGastos
	c.Reglas = append(c.Reglas,
		ReglaLiquidacionPropuesta{Referencia: "propuesta:tren", Tipo: ClaseOtroMedio, Concepto: "tren", TopeCentimos: 2000},
		ReglaLiquidacionPropuesta{Referencia: "propuesta:peaje", Tipo: ClaseOtroGasto, Concepto: "peaje", TopeCentimos: 500},
	)
	revisiones = append(revisiones,
		RevisionLineaLiquidacion{Indice: 1, ReglaRef: "propuesta:tren", ReconocidoPropuestoCentimos: 1500, MotivoCodigo: "revision_justificante"},
		RevisionLineaLiquidacion{Indice: 2, ReglaRef: "propuesta:peaje", ReconocidoPropuestoCentimos: 0, MotivoCodigo: "gasto_no_admitido"},
	)
	return d, c, revisiones
}

func TestLiquidacionOtrosGastosRecorteRechazoYCopia(t *testing.T) {
	d, c, revisiones := datosLiquidacionConGastos()
	p := propuestaTest(t, d, c, revisiones)
	antes := p.Instantanea()
	if antes.Liquidable || antes.Totales != (TotalesLiquidacionPropuesta{OriginalCentimos: 4070, ReconocidoPropuestoCentimos: 3370, RechazadoCentimos: 700}) || antes.Lineas[1].RechazadoCentimos != 300 || antes.Lineas[2].RechazadoCentimos != 400 {
		t.Fatalf("propuesta D5 incorrecta: %+v", antes)
	}
	*d.Lineas[1].JustificanteRef = "justificante:modificado"
	*d.Lineas[2].JustificanteSHA256 = strings.Repeat("c", 64)
	c.Reglas[1].TopeCentimos = 1
	revisiones[1].ReconocidoPropuestoCentimos = 0
	copia := p.Instantanea()
	*copia.Documento.Lineas[1].JustificanteRef = "justificante:modificado"
	copia.Catalogo.Reglas[1].TopeCentimos = 1
	copia.Lineas[1].ReconocidoPropuestoCentimos = 0
	if !reflect.DeepEqual(antes, p.Instantanea()) {
		t.Fatal("instantánea D5 alterada")
	}
}

func TestLiquidacionOtrosGastosRechazaContratosIncompletos(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*DocumentoComision, *CatalogoLiquidacionPropuesta, *[]RevisionLineaLiquidacion)
	}{
		{"tipo_ajeno", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			d.Lineas[1].TipoGasto = "avion"
		}},
		{"clase_ajena", func(_ *DocumentoComision, c *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			c.Reglas[1].Tipo = ClaseOtroGasto
		}},
		{"version_ajena", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			d.Lineas[1].CatalogoVersion = "provisional:otra"
		}},
		{"version_catalogo_ausente", func(_ *DocumentoComision, c *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			c.CatalogoOtrosGastosVersionRef = ""
		}},
		{"justificante_ausente", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			d.Lineas[1].JustificanteRef = nil
		}},
		{"huella_ajena", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			*d.Lineas[1].JustificanteSHA256 = "invalida"
		}},
		{"fecha_invalida", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			d.Lineas[1].Fecha = "2026-02-30"
		}},
		{"suma_otros", func(d *DocumentoComision, _ *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			d.OtrosCentimos++
		}},
		{"tope", func(_ *DocumentoComision, c *CatalogoLiquidacionPropuesta, _ *[]RevisionLineaLiquidacion) {
			c.Reglas[1].TopeCentimos = 1499
		}},
		{"sin_motivo", func(_ *DocumentoComision, _ *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[1].MotivoCodigo = ""
		}},
		{"importe_superior", func(_ *DocumentoComision, _ *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[1].ReconocidoPropuestoCentimos = 1801
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, c, revisiones := datosLiquidacionConGastos()
			caso.cambiar(&d, &c, &revisiones)
			h, _ := HuellaDatosLiquidacion(d)
			if _, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, revisiones); err == nil {
				t.Fatal("entrada D5 inválida aceptada")
			}
		})
	}
}

func TestLiquidacionOtrosGastosRespetaMaximoDocumental(t *testing.T) {
	d, c, revisiones := datosLiquidacionConGastos()
	for i := 0; i < 31; i++ {
		d.Lineas = append(d.Lineas, d.Lineas[1])
		d.OtrosCentimos += 1800
		d.TotalOrientativoCentimos += 1800
		revisiones = append(revisiones, RevisionLineaLiquidacion{Indice: len(d.Lineas) - 1, ReglaRef: "propuesta:tren", ReconocidoPropuestoCentimos: 1800})
	}
	h, _ := HuellaDatosLiquidacion(d)
	if _, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, revisiones); err == nil {
		t.Fatal("más de 32 gastos D5 aceptados")
	}
}
