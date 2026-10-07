package domain

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func datosLiquidacion() (DocumentoComision, CatalogoLiquidacionPropuesta, []RevisionLineaLiquidacion) {
	i := 0
	d := DocumentoComision{GrupoDieta: 2, VersionTarifaAceptada: "provisional:ensayo:v1", TramosAceptados: []int{0}, Lineas: []LineaDocumentoComision{{Tipo: "dieta", Grupo: 2, IndiceTramo: &i, Fecha: "2026-10-01", Concepto: "manutencion", ImporteCentimos: 1870, VersionTarifaRef: "provisional:ensayo:v1"}}, ManutencionCentimos: 1870, TotalOrientativoCentimos: 1870}
	c := CatalogoLiquidacionPropuesta{Referencia: "propuesta:catalogo", Version: "propuesta:v1", VersionTarifaRef: d.VersionTarifaAceptada, PaisISO2: "ES", Ambito: "nacional_ordinario_sin_alojamiento", Procedencia: "ejemplo_retirable_sin_aprobacion", Fuentes: []string{"https://www.boe.es/buscar/doc.php?id=BOE-A-2005-19988"}, Reglas: []ReglaLiquidacionPropuesta{{Referencia: "propuesta:manutencion", Tipo: "dieta", Concepto: "manutencion", Grupo: 2, TopeCentimos: 3740}}}
	return d, c, []RevisionLineaLiquidacion{{Indice: 0, ReglaRef: c.Reglas[0].Referencia, ReconocidoPropuestoCentimos: 1870}}
}
func propuestaTest(t *testing.T, d DocumentoComision, c CatalogoLiquidacionPropuesta, r []RevisionLineaLiquidacion) *PreparacionLiquidacion {
	t.Helper()
	h, err := HuellaDatosLiquidacion(d)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, r)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestLiquidacionRecorteRechazoYSumas(t *testing.T) {
	for _, reconocido := range []int64{1870, 1000, 0} {
		d, c, r := datosLiquidacion()
		r[0].ReconocidoPropuestoCentimos = reconocido
		if reconocido != 1870 {
			r[0].MotivoCodigo = "revision_documental"
		}
		s := propuestaTest(t, d, c, r).Instantanea()
		if s.Liquidable || s.Procedencia != "propuesta_sin_registrar" || s.Totales.OriginalCentimos != 1870 || s.Totales.ReconocidoPropuestoCentimos != reconocido || s.Totales.RechazadoCentimos != 1870-reconocido || s.Lineas[0].RechazadoCentimos != 1870-reconocido {
			t.Fatalf("%+v", s)
		}
		h := s.SnapshotSHA256
		s.SnapshotSHA256 = ""
		calculado, err := HuellaDatosLiquidacion(s)
		if err != nil || calculado != h {
			t.Fatal("snapshot no reproducible")
		}
	}
}
func TestLiquidacionCopiasProfundasYHuellas(t *testing.T) {
	d, c, r := datosLiquidacion()
	ref, sha := "justificante:ejemplo", strings.Repeat("a", 64)
	d.Lineas[0].JustificanteRef = &ref
	d.Lineas[0].JustificanteSHA256 = &sha
	p := propuestaTest(t, d, c, r)
	antes := p.Instantanea()
	*d.Lineas[0].IndiceTramo = 31
	*d.Lineas[0].JustificanteRef = "otro"
	*d.Lineas[0].JustificanteSHA256 = "otro"
	d.TramosAceptados[0] = 31
	c.Reglas[0].TopeCentimos = 1
	c.Fuentes[0] = "otro"
	r[0].ReconocidoPropuestoCentimos = 0
	copia := p.Instantanea()
	*copia.Documento.Lineas[0].IndiceTramo = 4
	*copia.Documento.Lineas[0].JustificanteRef = "otro"
	*copia.Documento.Lineas[0].JustificanteSHA256 = "otro"
	copia.Documento.TramosAceptados[0] = 9
	copia.Catalogo.Reglas[0].TopeCentimos = 1
	copia.Catalogo.Fuentes[0] = "otro"
	copia.Lineas[0].OriginalCentimos = 0
	doc := p.Documento()
	*doc.Lineas[0].IndiceTramo = 8
	cat := p.Catalogo()
	cat.Reglas[0].TopeCentimos = 1
	ls := p.Lineas()
	ls[0].OriginalCentimos = 1
	if !reflect.DeepEqual(antes, p.Instantanea()) {
		t.Fatal("instantanea mutable")
	}
	d, c, r = datosLiquidacion()
	primero := propuestaTest(t, d, c, r).Instantanea()
	c.Reglas[0].TopeCentimos++
	segundo := propuestaTest(t, d, c, r).Instantanea()
	if primero.CatalogoSHA256 == segundo.CatalogoSHA256 || primero.SnapshotSHA256 == segundo.SnapshotSHA256 {
		t.Fatal("configuracion invisible en huella")
	}
}
func TestLiquidacionRechazaEntradasYOverflow(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*DocumentoComision, *CatalogoLiquidacionPropuesta, *[]RevisionLineaLiquidacion)
	}{
		{"sin_motivo", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[0].ReconocidoPropuestoCentimos--
		}},
		{"motivo_libre", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[0].ReconocidoPropuestoCentimos--
			(*r)[0].MotivoCodigo = "motivo con texto"
		}},
		{"reconocido_supera_original", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[0].ReconocidoPropuestoCentimos++
		}},
		{"reconocido_negativo", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			(*r)[0].ReconocidoPropuestoCentimos = -1
		}},
		{"original_negativo", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			d.Lineas[0].ImporteCentimos = -1
		}},
		{"tope", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			c.Reglas[0].TopeCentimos = 1
		}},
		{"version", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			c.VersionTarifaRef = "otra:version"
		}},
		{"grupo", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			c.Reglas[0].Grupo = 3
		}},
		{"totales", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			d.TotalOrientativoCentimos++
		}},
		{"indices", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			d.TramosAceptados[0] = 1
		}},
		{"duplicado", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			c.Reglas = append(c.Reglas, c.Reglas[0])
		}},
		{"alojamiento_como_otros", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			convertirGastoLiquidacionTest(d, c, "otro_gasto", "alojamiento")
		}},
		{"otro_medio", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			convertirGastoLiquidacionTest(d, c, "otro_medio", "tren")
		}},
		{"alojamiento", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			d.Lineas[0].Concepto = "alojamiento_tope_pendiente_justificante"
		}},
		{"extranjero", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			c.PaisISO2 = "FR"
		}},
		{"overflow", func(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, r *[]RevisionLineaLiquidacion) {
			d.Lineas[0].ImporteCentimos = math.MaxInt64
			d.ManutencionCentimos = math.MaxInt64
			d.TotalOrientativoCentimos = math.MaxInt64
			c.Reglas[0].TopeCentimos = math.MaxInt64
			(*r)[0].ReconocidoPropuestoCentimos = math.MaxInt64
			i := 1
			l := d.Lineas[0]
			l.IndiceTramo = &i
			l.ImporteCentimos = 1
			d.Lineas = append(d.Lineas, l)
			d.TramosAceptados = append(d.TramosAceptados, 1)
			*r = append(*r, RevisionLineaLiquidacion{Indice: 1, ReglaRef: (*r)[0].ReglaRef, ReconocidoPropuestoCentimos: 1})
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			d, c, r := datosLiquidacion()
			caso.cambiar(&d, &c, &r)
			h, _ := HuellaDatosLiquidacion(d)
			if _, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, h, d, c, r); err == nil {
				t.Fatal("aceptado")
			}
		})
	}
	d, c, r := datosLiquidacion()
	if _, err := PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 2, strings.Repeat("0", 64), d, c, r); err == nil {
		t.Fatal("huella documento ajena aceptada")
	}
}

// Datos compatibles con el anterior bypass: gasto sin tramos de dieta y sumas coherentes.
func convertirGastoLiquidacionTest(d *DocumentoComision, c *CatalogoLiquidacionPropuesta, tipo, concepto string) {
	c.Reglas[0].Tipo = tipo
	c.Reglas[0].Concepto = concepto
	c.Reglas[0].Grupo = 0
	ref, sha := "justificante:ejemplo", strings.Repeat("a", 64)
	d.Lineas[0] = LineaDocumentoComision{Tipo: tipo, Fecha: "2026-10-01", Concepto: "Gasto declarado", ImporteCentimos: 1870, JustificanteRef: &ref, JustificanteSHA256: &sha, TipoGasto: concepto, CatalogoVersion: "provisional:otros:v1"}
	d.TramosAceptados = []int{}
	d.ManutencionCentimos = 0
	d.OtrosCentimos = 1870
}
