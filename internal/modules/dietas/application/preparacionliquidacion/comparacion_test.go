package preparacionliquidacion

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func modificarPropuesta(t *testing.T, s domain.InstantaneaLiquidacionPropuesta, modificar func(*Entrada)) domain.InstantaneaLiquidacionPropuesta {
	t.Helper()
	e := Entrada{Esquema: s.Esquema, ComisionRef: s.ComisionRef, ComisionVersion: s.ComisionVersion, DocumentoSHA256: s.DocumentoSHA256, Documento: s.Documento, Catalogo: s.Catalogo}
	for _, l := range s.Lineas {
		e.Revisiones = append(e.Revisiones, domain.RevisionLineaLiquidacion{Indice: l.Indice, ReglaRef: l.ReglaRef, ReconocidoPropuestoCentimos: l.ReconocidoPropuestoCentimos, MotivoCodigo: l.MotivoCodigo})
	}
	modificar(&e)
	p, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	return p.Instantanea()
}

func TestCompararIgualdadReduccionYRestitucion(t *testing.T) {
	anterior := instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json")
	casos := []struct {
		nombre     string
		importe    int64
		motivo     string
		diferencia int64
	}{
		{"igualdad", 1500, "revision_justificante", 0},
		{"reduccion", 0, "justificante_rechazado", -1500},
		{"restitucion", 1800, "", 300},
		{"solo_motivo", 1500, "tope_revisado", 0},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			propuesta := modificarPropuesta(t, instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json"), func(e *Entrada) {
				e.Revisiones[2].ReconocidoPropuestoCentimos, e.Revisiones[2].MotivoCodigo = caso.importe, caso.motivo
			})
			entrada := EntradaComparacion{EsquemaComparacionLiquidacion, anterior, propuesta}
			c, err := Comparar(entrada)
			if err != nil {
				t.Fatal(err)
			}
			l := c.Lineas[2]
			if c.Liquidable || c.Procedencia != "comparacion_local_sin_registrar" || c.Esquema != EsquemaComparacionLiquidacion || c.CatalogosDistintos || c.DocumentoSHA256 != anterior.DocumentoSHA256 || c.ComisionRef != anterior.ComisionRef || c.ComisionVersion != anterior.ComisionVersion {
				t.Fatalf("metadatos discordantes: %+v", c)
			}
			if l.Indice != 2 || l.OriginalCentimos != 1800 || l.ReconocidoAnteriorCentimos != 1500 || l.ReconocidoPropuestoCentimos != caso.importe || l.DiferenciaReconocidoCentimos != caso.diferencia || l.RechazadoAnteriorCentimos != 300 || l.RechazadoPropuestoCentimos != 1800-caso.importe || l.DiferenciaRechazadoCentimos != -caso.diferencia || l.MotivoAnteriorCodigo != "revision_justificante" || l.MotivoPropuestoCodigo != caso.motivo || l.CambioMotivo != (caso.motivo != "revision_justificante") || l.CambioRegla {
				t.Fatalf("línea discordante: %+v", l)
			}
			if c.Totales.OriginalCentimos != 6670 || c.Totales.ReconocidoAnteriorCentimos != 5970 || c.Totales.ReconocidoPropuestoCentimos != 5970+caso.diferencia || c.Totales.RechazadoAnteriorCentimos != 700 || c.Totales.RechazadoPropuestoCentimos != 700-caso.diferencia || c.Totales.DiferenciaReconocidoCentimos != caso.diferencia || c.Totales.DiferenciaRechazadoCentimos != -caso.diferencia {
				t.Fatalf("totales discordantes: %+v", c.Totales)
			}
			for _, i := range []int{0, 1, 3} {
				if c.Lineas[i].DiferenciaReconocidoCentimos != 0 || c.Lineas[i].DiferenciaRechazadoCentimos != 0 || c.Lineas[i].CambioRegla || c.Lineas[i].CambioMotivo {
					t.Fatal("línea sin cambios alterada")
				}
			}
			if !reflect.DeepEqual(entrada.Anterior, anterior) || !reflect.DeepEqual(entrada.Propuesta, propuesta) {
				t.Fatal("entrada mutada")
			}
		})
	}
}

func TestCompararCatalogosYReglasDiferentesSinCambioMonetario(t *testing.T) {
	a := instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json")
	p := modificarPropuesta(t, instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json"), func(e *Entrada) {
		e.Catalogo.Version = "propuesta:catalogo:v2"
		e.Catalogo.Reglas[2].Referencia = "propuesta:regla:tren:v2"
		e.Catalogo.Reglas[2].TopeCentimos = 1900
		e.Revisiones[2].ReglaRef = e.Catalogo.Reglas[2].Referencia
	})
	c, err := Comparar(EntradaComparacion{EsquemaComparacionLiquidacion, a, p})
	if err != nil || !c.CatalogosDistintos || !c.Lineas[2].CambioRegla || c.Lineas[2].CambioMotivo || c.Lineas[2].ReglaAnteriorRef != a.Lineas[2].ReglaRef || c.Lineas[2].ReglaPropuestaRef != p.Lineas[2].ReglaRef || c.Totales.DiferenciaReconocidoCentimos != 0 || c.Totales.DiferenciaRechazadoCentimos != 0 || c.Anterior != fuentesComparacion(a) || c.Propuesta != fuentesComparacion(p) {
		t.Fatalf("catálogos o reglas omitidos: %+v, %v", c, err)
	}
	// También se declara el cambio de contenido aunque se reutilicen las etiquetas.
	p = modificarPropuesta(t, instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json"), func(e *Entrada) { e.Catalogo.Reglas[2].TopeCentimos++ })
	c, err = Comparar(EntradaComparacion{EsquemaComparacionLiquidacion, a, p})
	if err != nil || !c.CatalogosDistintos || c.Lineas[2].CambioRegla || c.Totales.DiferenciaReconocidoCentimos != 0 {
		t.Fatalf("contenido de catálogo omitido: %+v, %v", c, err)
	}
}

func TestCompararRechazaOtraComisionVersionODocumento(t *testing.T) {
	a := instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json")
	for nombre, mutar := range map[string]func(*Entrada){
		"comision": func(e *Entrada) { e.ComisionRef = "dco_otro_sintetico_20261001" },
		"version":  func(e *Entrada) { e.ComisionVersion++ },
		"documento": func(e *Entrada) {
			e.Documento.Lineas[2].Concepto = "Otro billete sintético"
			e.DocumentoSHA256, _ = domain.HuellaDatosLiquidacion(e.Documento)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			p := modificarPropuesta(t, instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json"), mutar)
			if c, err := Comparar(EntradaComparacion{EsquemaComparacionLiquidacion, a, p}); c != nil || !errors.Is(err, ErrComparacionLiquidacion) {
				t.Fatalf("sustitución aceptada: %v", err)
			}
		})
	}
}

func TestCompararRevalidaAmbasInstantaneas(t *testing.T) {
	for _, lado := range []string{"anterior", "propuesta"} {
		t.Run(lado, func(t *testing.T) {
			a := instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json")
			p := instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json")
			s := &a
			if lado == "propuesta" {
				s = &p
			}
			s.Lineas[2].RechazadoCentimos++
			s.SnapshotSHA256 = ""
			s.SnapshotSHA256, _ = domain.HuellaDatosLiquidacion(*s)
			if c, err := Comparar(EntradaComparacion{EsquemaComparacionLiquidacion, a, p}); c != nil || !errors.Is(err, domain.ErrPreparacionLiquidacion) {
				t.Fatalf("importación incoherente aceptada: %v", err)
			}
		})
	}
	if c, err := Comparar(EntradaComparacion{}); c != nil || !errors.Is(err, ErrComparacionLiquidacion) {
		t.Fatal("esquema admitido", err)
	}
}

func TestCompararDiferenciasEnLimiteInt64(t *testing.T) {
	a := modificarPropuesta(t, instantaneaRecuperacion(t, "preparacion_liquidacion_gastos.json"), func(e *Entrada) {
		e.Documento.Lineas[0].ImporteCentimos = math.MaxInt64 - 4800
		e.Documento.ManutencionCentimos = math.MaxInt64 - 4800
		e.Documento.TotalOrientativoCentimos = math.MaxInt64
		e.DocumentoSHA256, _ = domain.HuellaDatosLiquidacion(e.Documento)
		e.Catalogo.Reglas[0].TopeCentimos = math.MaxInt64
		e.Revisiones[0].ReconocidoPropuestoCentimos = e.Documento.Lineas[0].ImporteCentimos
		e.Revisiones[0].MotivoCodigo = ""
	})
	p := modificarPropuesta(t, a, func(e *Entrada) {
		e.Revisiones[0].ReconocidoPropuestoCentimos = 0
		e.Revisiones[0].MotivoCodigo = "gasto_rechazado"
	})
	for _, entrada := range []EntradaComparacion{{EsquemaComparacionLiquidacion, a, p}, {EsquemaComparacionLiquidacion, p, a}} {
		c, err := Comparar(entrada)
		if err != nil || c.Totales.OriginalCentimos != math.MaxInt64 || c.Totales.DiferenciaReconocidoCentimos != -c.Totales.DiferenciaRechazadoCentimos || c.Totales.DiferenciaReconocidoCentimos != c.Lineas[0].DiferenciaReconocidoCentimos {
			t.Fatalf("diferencia fuera de rango: %+v, %v", c, err)
		}
	}
}
