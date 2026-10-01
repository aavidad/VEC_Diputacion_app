package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func datosLiquidacionRutaImportada() (DocumentoComision, CatalogoLiquidacionPropuesta, []RevisionLineaLiquidacion) {
	d, c, revisiones := datosLiquidacion()
	d.VehiculoPropio = true
	d.Lineas = append(d.Lineas, LineaDocumentoComision{
		Tipo: "kilometraje", RutaIndice: 1, OrigenCodigo: "granada", DestinoCodigo: "motril",
		Kilometros: "102.5000", KilometrosBase: "100.0000", AjusteKilometros: "2.5000",
		MotivoAjuste: "Desvío declarado por obras", VersionGrafo: "grafo:sintetico:v1",
		VersionTarifaRef: d.VersionTarifaAceptada, ImporteCentimos: 2000,
	})
	d.KilometrajeCentimos = 2000
	d.TotalOrientativoCentimos += 2000
	c.Reglas = append(c.Reglas, ReglaLiquidacionPropuesta{
		Referencia: "propuesta:kilometraje", Tipo: "kilometraje", Concepto: "vehiculo_propio", CentimosPorKM: 26,
	})
	revisiones = append(revisiones, RevisionLineaLiquidacion{Indice: 1, ReglaRef: "propuesta:kilometraje", ReconocidoPropuestoCentimos: 2000})
	return d, c, revisiones
}

func TestLiquidacionRutasImportadasCoherentesConservanHistoria(t *testing.T) {
	casos := []struct {
		nombre, base, ajuste, final, motivo string
	}{
		{"positivo", "100.0000", "2.5000", "102.5000", "Desvío declarado por obras"},
		{"negativo", "105.0000", "-2.5000", "102.5000", "Corrección del recorrido declarado"},
		{"sin_ajuste", "102.5000", "0.0000", "102.5000", ""},
		{"precision", "102.4999", "0.0001", "102.5000", "Corrección de distancia declarada"},
		{"limite_ajuste", "100.0000", "1000.0000", "1100.0000", "Recorrido adicional declarado"},
		{"limite_final", "11000.0000", "-1000.0000", "10000.0000", "Corrección del recorrido declarado"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, c, revisiones := datosLiquidacionRutaImportada()
			l := &d.Lineas[1]
			l.KilometrosBase, l.AjusteKilometros, l.Kilometros, l.MotivoAjuste = caso.base, caso.ajuste, caso.final, caso.motivo
			antes := clonarDocumentoLiquidacion(d)
			// La revisión puede venir en otro orden que las líneas del documento.
			revisiones[0], revisiones[1] = revisiones[1], revisiones[0]
			s := propuestaTest(t, d, c, revisiones).Instantanea()
			if !reflect.DeepEqual(d, antes) || !reflect.DeepEqual(s.Documento, antes) || s.Liquidable {
				t.Fatal("la propuesta alteró el documento histórico o lo hizo liquidable")
			}
			if s.Lineas[1].OriginalCentimos != 2000 || s.Lineas[1].ReconocidoPropuestoCentimos != 2000 || s.Totales.OriginalCentimos != 3870 {
				t.Fatalf("importe histórico recalculado: %+v", s)
			}
			h, err := HuellaDatosLiquidacion(s.Documento)
			if err != nil || h != s.DocumentoSHA256 {
				t.Fatal("huella documental modificada")
			}
		})
	}
}

func TestLiquidacionRutasImportadasRechazaIncoherenciasConHuellaCorrecta(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*LineaDocumentoComision)
	}{
		{"base_ausente", func(l *LineaDocumentoComision) { l.KilometrosBase = "" }},
		{"base_cero", func(l *LineaDocumentoComision) { l.KilometrosBase = "0.0000" }},
		{"base_negativa", func(l *LineaDocumentoComision) { l.KilometrosBase = "-100.0000" }},
		{"base_precision", func(l *LineaDocumentoComision) { l.KilometrosBase = "100.00001" }},
		{"base_formato", func(l *LineaDocumentoComision) { l.KilometrosBase = "0100.0000" }},
		{"base_desbordada", func(l *LineaDocumentoComision) { l.KilometrosBase = "9223372036854775807.0000" }},
		{"base_incoherente", func(l *LineaDocumentoComision) { l.KilometrosBase = "101.0000" }},
		{"ajuste_ausente", func(l *LineaDocumentoComision) { l.AjusteKilometros = "" }},
		{"ajuste_precision", func(l *LineaDocumentoComision) { l.AjusteKilometros = "2.50001" }},
		{"ajuste_signo_mas", func(l *LineaDocumentoComision) { l.AjusteKilometros = "+2.5000" }},
		{"ajuste_incoherente", func(l *LineaDocumentoComision) { l.AjusteKilometros = "-2.5000" }},
		{"ajuste_fuera_limite", func(l *LineaDocumentoComision) {
			l.AjusteKilometros, l.Kilometros = "1000.0001", "1100.0001"
		}},
		{"ajuste_negativo_fuera_limite", func(l *LineaDocumentoComision) {
			l.KilometrosBase, l.AjusteKilometros = "1102.5001", "-1000.0001"
		}},
		{"cero_negativo", func(l *LineaDocumentoComision) {
			l.KilometrosBase, l.AjusteKilometros, l.MotivoAjuste = l.Kilometros, "-0.0000", ""
		}},
		{"motivo_ausente", func(l *LineaDocumentoComision) { l.MotivoAjuste = "" }},
		{"motivo_corto", func(l *LineaDocumentoComision) { l.MotivoAjuste = "ab" }},
		{"motivo_largo", func(l *LineaDocumentoComision) { l.MotivoAjuste = strings.Repeat("a", 501) }},
		{"motivo_espacios", func(l *LineaDocumentoComision) { l.MotivoAjuste = " Desvío declarado " }},
		{"motivo_salto", func(l *LineaDocumentoComision) { l.MotivoAjuste = "Desvío\ndeclarado" }},
		{"motivo_nulo", func(l *LineaDocumentoComision) { l.MotivoAjuste = "Desvío\x00declarado" }},
		{"motivo_sin_ajuste", func(l *LineaDocumentoComision) {
			l.KilometrosBase, l.AjusteKilometros = l.Kilometros, "0.0000"
		}},
		{"final_cero", func(l *LineaDocumentoComision) {
			l.KilometrosBase, l.AjusteKilometros, l.Kilometros = "100.0000", "-100.0000", "0.0000"
		}},
		{"final_fuera_limite", func(l *LineaDocumentoComision) {
			l.KilometrosBase, l.AjusteKilometros, l.Kilometros, l.MotivoAjuste = "10000.0001", "0.0000", "10000.0001", ""
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, c, revisiones := datosLiquidacionRutaImportada()
			caso.cambiar(&d.Lineas[1])
			antes := clonarDocumentoLiquidacion(d)
			// Recalcular la huella evita que el rechazo se deba solo al hash.
			h, err := HuellaDatosLiquidacion(d)
			if err != nil {
				t.Fatal(err)
			}
			p, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, revisiones)
			if p != nil || !errors.Is(err, ErrPreparacionLiquidacion) {
				t.Fatalf("ruta incoherente aceptada: propuesta=%v error=%v", p, err)
			}
			if !reflect.DeepEqual(d, antes) {
				t.Fatal("el rechazo alteró el documento importado")
			}
		})
	}
}

func TestLiquidacionRutasImportadasLimiteTotalDocumental(t *testing.T) {
	for _, caso := range []struct {
		nombre, segundaDistancia string
		rechazar                 bool
	}{
		{"justo_limite", "5000.0000", false},
		{"supera_por_una_unidad", "5000.0001", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d, c, revisiones := datosLiquidacionRutaImportada()
			primera := &d.Lineas[1]
			primera.KilometrosBase, primera.Kilometros = "5000.0000", "5000.0000"
			primera.AjusteKilometros, primera.MotivoAjuste = "0.0000", ""
			segunda := *primera
			segunda.RutaIndice = 2
			segunda.KilometrosBase, segunda.Kilometros = caso.segundaDistancia, caso.segundaDistancia
			d.Lineas = append(d.Lineas, segunda)
			d.KilometrajeCentimos += segunda.ImporteCentimos
			d.TotalOrientativoCentimos += segunda.ImporteCentimos
			revisiones = append(revisiones, RevisionLineaLiquidacion{Indice: 2, ReglaRef: "propuesta:kilometraje", ReconocidoPropuestoCentimos: segunda.ImporteCentimos})
			antes := clonarDocumentoLiquidacion(d)
			h, err := HuellaDatosLiquidacion(d)
			if err != nil {
				t.Fatal(err)
			}
			p, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, revisiones)
			if caso.rechazar {
				if p != nil || !errors.Is(err, ErrPreparacionLiquidacion) {
					t.Fatalf("suma superior al límite aceptada: propuesta=%v error=%v", p, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				s := p.Instantanea()
				if !reflect.DeepEqual(s.Documento, antes) || s.DocumentoSHA256 != h || s.Totales.OriginalCentimos != 5870 || s.Lineas[1].OriginalCentimos != 2000 || s.Lineas[2].OriginalCentimos != 2000 || s.Liquidable {
					t.Fatal("la propuesta alteró el documento o los importes originales")
				}
			}
			if !reflect.DeepEqual(d, antes) {
				t.Fatal("la validación alteró el documento importado")
			}
		})
	}
}
