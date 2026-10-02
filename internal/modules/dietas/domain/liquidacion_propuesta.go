package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const EsquemaPreparacionLiquidacion = "vec_dietas_preparacion_liquidacion_v1"

var ErrPreparacionLiquidacion = errors.New("preparacion_liquidacion_invalida")
var codigoMotivoLiquidacion = regexp.MustCompile(`^[a-z][a-z0-9_]{2,79}$`)

// El catálogo importado restringe propuestas; no publica tarifas ni concede competencias.
type CatalogoLiquidacionPropuesta struct {
	Referencia                    string                      `json:"referencia"`
	Version                       string                      `json:"version"`
	VersionTarifaRef              string                      `json:"version_tarifa_ref"`
	CatalogoOtrosGastosVersionRef string                      `json:"catalogo_otros_gastos_version_ref,omitempty"`
	PaisISO2                      string                      `json:"pais_iso2"`
	Ambito                        string                      `json:"ambito"`
	Procedencia                   string                      `json:"procedencia"`
	Fuentes                       []string                    `json:"fuentes"`
	Reglas                        []ReglaLiquidacionPropuesta `json:"reglas"`
}
type ReglaLiquidacionPropuesta struct {
	Referencia    string `json:"referencia"`
	Tipo          string `json:"tipo"`
	Concepto      string `json:"concepto"`
	Grupo         int    `json:"grupo"`
	TopeCentimos  int64  `json:"tope_centimos"`
	CentimosPorKM int64  `json:"centimos_por_km"`
}
type RevisionLineaLiquidacion struct {
	Indice                      int    `json:"indice"`
	ReglaRef                    string `json:"regla_ref"`
	ReconocidoPropuestoCentimos int64  `json:"reconocido_propuesto_centimos"`
	MotivoCodigo                string `json:"motivo_codigo"`
}
type LineaLiquidacionPropuesta struct {
	Indice                      int    `json:"indice"`
	ReglaRef                    string `json:"regla_ref"`
	OriginalCentimos            int64  `json:"original_centimos"`
	ReconocidoPropuestoCentimos int64  `json:"reconocido_propuesto_centimos"`
	RechazadoCentimos           int64  `json:"rechazado_centimos"`
	MotivoCodigo                string `json:"motivo_codigo"`
}
type TotalesLiquidacionPropuesta struct {
	OriginalCentimos            int64 `json:"original_centimos"`
	ReconocidoPropuestoCentimos int64 `json:"reconocido_propuesto_centimos"`
	RechazadoCentimos           int64 `json:"rechazado_centimos"`
}

// InstantaneaLiquidacionPropuesta es una copia serializable, nunca una liquidación registrada.
type InstantaneaLiquidacionPropuesta struct {
	Esquema         string                       `json:"esquema"`
	Procedencia     string                       `json:"procedencia"`
	Liquidable      bool                         `json:"liquidable"`
	ComisionRef     string                       `json:"comision_ref"`
	ComisionVersion int64                        `json:"comision_version"`
	DocumentoSHA256 string                       `json:"documento_sha256"`
	CatalogoRef     string                       `json:"catalogo_ref"`
	CatalogoVersion string                       `json:"catalogo_version"`
	CatalogoSHA256  string                       `json:"catalogo_sha256"`
	Catalogo        CatalogoLiquidacionPropuesta `json:"catalogo"`
	Documento       DocumentoComision            `json:"documento"`
	Lineas          []LineaLiquidacionPropuesta  `json:"lineas"`
	Totales         TotalesLiquidacionPropuesta  `json:"totales"`
	SnapshotSHA256  string                       `json:"snapshot_sha256,omitempty"`
}

// PreparacionLiquidacion mantiene datos privados y entrega copias profundas.
// Ninguna huella prueba aprobación, publicación, autenticidad o custodia.
type PreparacionLiquidacion struct {
	instantanea InstantaneaLiquidacionPropuesta
}

func (p *PreparacionLiquidacion) Instantanea() InstantaneaLiquidacionPropuesta {
	s := p.instantanea
	s.Catalogo = clonarCatalogoLiquidacion(s.Catalogo)
	s.Documento = clonarDocumentoLiquidacion(s.Documento)
	s.Lineas = append([]LineaLiquidacionPropuesta{}, s.Lineas...)
	return s
}
func (p *PreparacionLiquidacion) Lineas() []LineaLiquidacionPropuesta { return p.Instantanea().Lineas }
func (p *PreparacionLiquidacion) Documento() DocumentoComision        { return p.Instantanea().Documento }
func (p *PreparacionLiquidacion) Catalogo() CatalogoLiquidacionPropuesta {
	return p.Instantanea().Catalogo
}

// HuellaDatosLiquidacion define JSON tipado de Go, sin espacios ni salto final.
// El orden de arrays es significativo; no es RFC8785 ni jsonb::text.
func HuellaDatosLiquidacion(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", ErrPreparacionLiquidacion
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func PrepararLiquidacion(ref string, version int64, documentoSHA string, d DocumentoComision, c CatalogoLiquidacionPropuesta, revisiones []RevisionLineaLiquidacion) (*PreparacionLiquidacion, error) {
	if !referenciaComision.MatchString(ref) || version < 1 || !huellaJustificante.MatchString(documentoSHA) || len(d.Lineas) < 1 || len(d.Lineas) > 102 || len(d.TramosAceptados) > 62 || len(revisiones) != len(d.Lineas) || d.GrupoDieta < 1 || d.GrupoDieta > 3 || validarCatalogoLiquidacion(c) != nil || d.VersionTarifaAceptada != c.VersionTarifaRef {
		return nil, ErrPreparacionLiquidacion
	}
	h, err := HuellaDatosLiquidacion(d)
	if err != nil || h != documentoSHA {
		return nil, ErrPreparacionLiquidacion
	}
	reglas := map[string]ReglaLiquidacionPropuesta{}
	for _, r := range c.Reglas {
		reglas[r.Referencia] = r
	}
	indices := map[int]bool{}
	lineas := make([]LineaLiquidacionPropuesta, len(d.Lineas))
	var totales TotalesLiquidacionPropuesta
	var manutencion, km, otros int64
	var kilometrosEscalados int64
	tramos := make([]int, 0)
	rutas := 0
	otrosLineas := 0
	for _, v := range revisiones {
		if v.Indice < 0 || v.Indice >= len(d.Lineas) || indices[v.Indice] {
			return nil, ErrPreparacionLiquidacion
		}
		indices[v.Indice] = true
		l := d.Lineas[v.Indice]
		r, ok := reglas[v.ReglaRef]
		if !ok || l.ImporteCentimos < 0 || v.ReconocidoPropuestoCentimos < 0 || v.ReconocidoPropuestoCentimos > l.ImporteCentimos || l.Tipo != r.Tipo {
			return nil, ErrPreparacionLiquidacion
		}
		if v.ReconocidoPropuestoCentimos < l.ImporteCentimos {
			if !codigoMotivoLiquidacion.MatchString(v.MotivoCodigo) {
				return nil, ErrPreparacionLiquidacion
			}
		} else if v.MotivoCodigo != "" {
			return nil, ErrPreparacionLiquidacion
		}
		limite := r.TopeCentimos
		switch l.Tipo {
		case "dieta":
			if l.Concepto != "manutencion" || r.Concepto != l.Concepto || l.Grupo != int(d.GrupoDieta) || r.Grupo != l.Grupo || l.VersionTarifaRef != c.VersionTarifaRef || l.IndiceTramo == nil || *l.IndiceTramo < 0 || *l.IndiceTramo >= 62 || !fechaCivilValida(l.Fecha) {
				return nil, ErrPreparacionLiquidacion
			}
			if !sumarLiquidacion(&manutencion, l.ImporteCentimos) {
				return nil, ErrPreparacionLiquidacion
			}
		case "kilometraje":
			if !d.VehiculoPropio || l.VersionTarifaRef != c.VersionTarifaRef || !referenciaJustificante.MatchString(l.VersionGrafo) || !codigoRuta.MatchString(l.OrigenCodigo) || !codigoRuta.MatchString(l.DestinoCodigo) || l.RutaIndice < 1 || l.RutaIndice > 8 {
				return nil, ErrPreparacionLiquidacion
			}
			cantidad, err := decimal4(l.Kilometros)
			if err != nil || cantidad <= 0 || cantidad > 10000*10000-kilometrosEscalados || cantidad > (math.MaxInt64-5000)/r.CentimosPorKM {
				return nil, ErrPreparacionLiquidacion
			}
			// Cotejar la ruta importada con el contrato documental, sin consultar
			// el grafo ni recalcular el importe original con otra tarifa.
			base, err := decimal4(l.KilometrosBase)
			if err != nil || base <= 0 || !ajusteTextoValido(l.AjusteKilometros, l.MotivoAjuste) {
				return nil, ErrPreparacionLiquidacion
			}
			ajuste, _ := ajusteEscalado(l.AjusteKilometros)
			if base+ajuste != cantidad {
				return nil, ErrPreparacionLiquidacion
			}
			kilometrosEscalados += cantidad
			limite = (cantidad*r.CentimosPorKM + 5000) / 10000
			if !sumarLiquidacion(&km, l.ImporteCentimos) {
				return nil, ErrPreparacionLiquidacion
			}
		case ClaseOtroMedio, ClaseOtroGasto:
			otrosLineas++
			if otrosLineas > 32 || l.TipoGasto != r.Concepto || l.CatalogoVersion != c.CatalogoOtrosGastosVersionRef || l.JustificanteRef == nil || l.JustificanteSHA256 == nil {
				return nil, ErrPreparacionLiquidacion
			}
			gasto := OtroGastoDeclarado{Tipo: l.Tipo, Concepto: l.Concepto, ImporteCentimos: l.ImporteCentimos, JustificanteRef: *l.JustificanteRef, JustificanteSHA256: *l.JustificanteSHA256, TipoGasto: l.TipoGasto, CatalogoVersion: l.CatalogoVersion, Fecha: l.Fecha}
			if gasto.validarComun() != nil || !sumarLiquidacion(&otros, l.ImporteCentimos) {
				return nil, ErrPreparacionLiquidacion
			}
		default:
			return nil, ErrPreparacionLiquidacion
		}
		if v.ReconocidoPropuestoCentimos > limite {
			return nil, ErrPreparacionLiquidacion
		}
		rechazo := l.ImporteCentimos - v.ReconocidoPropuestoCentimos
		if !sumarLiquidacion(&totales.OriginalCentimos, l.ImporteCentimos) || !sumarLiquidacion(&totales.ReconocidoPropuestoCentimos, v.ReconocidoPropuestoCentimos) || !sumarLiquidacion(&totales.RechazadoCentimos, rechazo) {
			return nil, ErrPreparacionLiquidacion
		}
		lineas[v.Indice] = LineaLiquidacionPropuesta{v.Indice, v.ReglaRef, l.ImporteCentimos, v.ReconocidoPropuestoCentimos, rechazo, v.MotivoCodigo}
	}
	// Validar índices en orden documental, independientemente del orden de revisión.
	tramos = tramos[:0]
	rutas = 0
	for _, l := range d.Lineas {
		if l.Tipo == "dieta" {
			tramos = append(tramos, *l.IndiceTramo)
		}
		if l.Tipo == "kilometraje" {
			rutas++
			if l.RutaIndice != rutas {
				return nil, ErrPreparacionLiquidacion
			}
		}
	}
	if len(tramos) != len(d.TramosAceptados) || d.AlojamientoTopeCentimos != 0 || manutencion != d.ManutencionCentimos || km != d.KilometrajeCentimos || otros != d.OtrosCentimos || totales.OriginalCentimos != d.TotalOrientativoCentimos || totales.ReconocidoPropuestoCentimos > math.MaxInt64-totales.RechazadoCentimos || totales.ReconocidoPropuestoCentimos+totales.RechazadoCentimos != totales.OriginalCentimos {
		return nil, ErrPreparacionLiquidacion
	}
	for i, n := range tramos {
		if n != d.TramosAceptados[i] || (i > 0 && n <= tramos[i-1]) {
			return nil, ErrPreparacionLiquidacion
		}
	}
	catSHA, err := HuellaDatosLiquidacion(c)
	if err != nil {
		return nil, err
	}
	s := InstantaneaLiquidacionPropuesta{Esquema: EsquemaPreparacionLiquidacion, Procedencia: "propuesta_sin_registrar", ComisionRef: ref, ComisionVersion: version, DocumentoSHA256: documentoSHA, CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoSHA256: catSHA, Catalogo: clonarCatalogoLiquidacion(c), Documento: clonarDocumentoLiquidacion(d), Lineas: lineas, Totales: totales}
	s.SnapshotSHA256, err = HuellaDatosLiquidacion(s)
	if err != nil {
		return nil, err
	}
	return &PreparacionLiquidacion{s}, nil
}
func validarCatalogoLiquidacion(c CatalogoLiquidacionPropuesta) error {
	if !referenciaJustificante.MatchString(c.Referencia) || !referenciaJustificante.MatchString(c.Version) || !referenciaJustificante.MatchString(c.VersionTarifaRef) || c.PaisISO2 != "ES" || c.Ambito != "nacional_ordinario_sin_alojamiento" || c.Procedencia != "ejemplo_retirable_sin_aprobacion" || len(c.Reglas) < 1 || len(c.Reglas) > 32 || len(c.Fuentes) < 1 || len(c.Fuentes) > 8 {
		return ErrPreparacionLiquidacion
	}
	fuentes := map[string]bool{}
	for _, f := range c.Fuentes {
		if len(f) > 300 || !strings.HasPrefix(f, "https://www.boe.es/") || strings.ContainsAny(f, "\r\n\x00") || fuentes[f] {
			return ErrPreparacionLiquidacion
		}
		fuentes[f] = true
	}
	refs := map[string]bool{}
	claves := map[string]bool{}
	hayD5 := false
	for _, r := range c.Reglas {
		if !referenciaJustificante.MatchString(r.Referencia) || refs[r.Referencia] || r.TopeCentimos < 0 || r.CentimosPorKM < 0 {
			return ErrPreparacionLiquidacion
		}
		refs[r.Referencia] = true
		key := r.Tipo + ":" + r.Concepto + ":" + strconv.Itoa(r.Grupo)
		if claves[key] {
			return ErrPreparacionLiquidacion
		}
		claves[key] = true
		switch r.Tipo {
		case "dieta":
			if r.Concepto != "manutencion" || r.Grupo < 1 || r.Grupo > 3 || r.TopeCentimos < 1 || r.CentimosPorKM != 0 {
				return ErrPreparacionLiquidacion
			}
		case "kilometraje":
			if r.Concepto != "vehiculo_propio" || r.Grupo != 0 || r.TopeCentimos != 0 || r.CentimosPorKM < 1 || r.CentimosPorKM > 10000 {
				return ErrPreparacionLiquidacion
			}
		case ClaseOtroMedio, ClaseOtroGasto:
			hayD5 = true
			clase, ok := ClaseDeTipoOtroGasto(c.CatalogoOtrosGastosVersionRef, r.Concepto)
			if !ok || clase != r.Tipo || r.Grupo != 0 || r.TopeCentimos < 1 || r.CentimosPorKM != 0 {
				return ErrPreparacionLiquidacion
			}
		default:
			return ErrPreparacionLiquidacion
		}
	}
	if !hayD5 && c.CatalogoOtrosGastosVersionRef != "" {
		return ErrPreparacionLiquidacion
	}
	return nil
}
func sumarLiquidacion(total *int64, n int64) bool {
	if n < 0 || *total > math.MaxInt64-n {
		return false
	}
	*total += n
	return true
}
func clonarCatalogoLiquidacion(c CatalogoLiquidacionPropuesta) CatalogoLiquidacionPropuesta {
	c.Fuentes = append([]string{}, c.Fuentes...)
	c.Reglas = append([]ReglaLiquidacionPropuesta{}, c.Reglas...)
	return c
}
func clonarDocumentoLiquidacion(d DocumentoComision) DocumentoComision {
	d.TramosAceptados = slices.Clone(d.TramosAceptados)
	d.Lineas = slices.Clone(d.Lineas)
	for i := range d.Lineas {
		l := &d.Lineas[i]
		if l.IndiceTramo != nil {
			n := *l.IndiceTramo
			l.IndiceTramo = &n
		}
		if l.JustificanteRef != nil {
			s := *l.JustificanteRef
			l.JustificanteRef = &s
		}
		if l.JustificanteSHA256 != nil {
			s := *l.JustificanteSHA256
			l.JustificanteSHA256 = &s
		}
	}
	return d
}
