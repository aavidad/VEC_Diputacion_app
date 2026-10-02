package informeliquidacion

import (
	"bytes"
	_ "embed"
	"errors"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

var ErrPreparacion = errors.New("dietas_informe_preparacion_invalida")
var ErrTema = errors.New("dietas_informe_tema_invalido")

// Configuracion recibe únicamente CSS de la autoridad visual común, leído
// por la composición local. No se carga ningún recurso desde el documento.
type Configuracion struct{ TemaCSS string }
type Renderizador struct{ tema template.CSS }

//go:embed informe.gohtml
var plantilla string

//go:embed informe.css
var estilos string

var cargaCSS = regexp.MustCompile(`(?i)(url\s*\(|@import|\\)`)
var informe = template.Must(template.New("informe").Parse(plantilla))

func Nuevo(c Configuracion) (*Renderizador, error) {
	if len(c.TemaCSS) == 0 || len(c.TemaCSS) > 65536 || strings.ContainsAny(c.TemaCSS, "<>\x00") || cargaCSS.MatchString(c.TemaCSS) {
		return nil, ErrTema
	}
	return &Renderizador{tema: template.CSS(c.TemaCSS)}, nil
}

type fila struct {
	Concepto, Detalle, Inicial, Propuesto, Diferencia, Motivo, Regla          string
	Descripcion, Fecha, JustificanteRef, JustificanteSHA, CatalogoOtros, Tope string
	Numero                                                                    int
	D5, Kilometraje                                                           bool
	OrigenCodigo, DestinoCodigo, KilometrosBase, KilometrosFinales            string
	AjusteKilometros, MotivoAjuste                                            string
	RutaNumero                                                                int
}
type vista struct {
	Textos                                                                          Textos
	Tema, Estilos                                                                   template.CSS
	ComisionRef, Version, Grupo                                                     string
	Filas                                                                           []fila
	Resumen                                                                         []fila
	TotalInicial, TotalPropuesto, TotalDiferencia                                   string
	CatalogoRef, CatalogoVersion, Tarifa, DocumentoSHA, CatalogoSHA, PreparacionSHA string
	Fuentes                                                                         []string
	TieneD5                                                                         bool
}

var familiasInforme = [...]string{"manutencion", "kilometraje", domain.ClaseOtroMedio, domain.ClaseOtroGasto}

// Renderizar consume los importes y totales decididos por el dominio opaco.
// Sólo transforma presentación; no concede aprobación ni registra una liquidación.
func (r *Renderizador) Renderizar(p *domain.PreparacionLiquidacion, t Textos) ([]byte, error) {
	if r == nil || r.tema == "" {
		return nil, ErrTema
	}
	if p == nil {
		return nil, ErrPreparacion
	}
	if err := t.validar(); err != nil {
		return nil, err
	}
	s := p.Instantanea()
	if s.Esquema != domain.EsquemaPreparacionLiquidacion || s.Liquidable || s.SnapshotSHA256 == "" || len(s.Lineas) == 0 || len(s.Lineas) != len(s.Documento.Lineas) {
		return nil, ErrPreparacion
	}
	v := vista{Textos: t, Tema: r.tema, Estilos: template.CSS(estilos), ComisionRef: s.ComisionRef, Version: strconv.FormatInt(s.ComisionVersion, 10), Grupo: strconv.Itoa(int(s.Documento.GrupoDieta)), CatalogoRef: s.CatalogoRef, CatalogoVersion: s.CatalogoVersion, Tarifa: s.Catalogo.VersionTarifaRef, DocumentoSHA: s.DocumentoSHA256, CatalogoSHA: s.CatalogoSHA256, PreparacionSHA: s.SnapshotSHA256, Fuentes: s.Catalogo.Fuentes}
	reglas := make(map[string]domain.ReglaLiquidacionPropuesta, len(s.Catalogo.Reglas))
	for _, regla := range s.Catalogo.Reglas {
		reglas[regla.Referencia] = regla
	}
	subtotales := make(map[string]*domain.TotalesLiquidacionPropuesta, len(familiasInforme))
	for _, codigo := range familiasInforme {
		subtotales[codigo] = &domain.TotalesLiquidacionPropuesta{}
	}
	for _, l := range s.Lineas {
		if l.Indice < 0 || l.Indice >= len(s.Documento.Lineas) {
			return nil, ErrPreparacion
		}
		d := s.Documento.Lineas[l.Indice]
		familia := d.Tipo
		if d.Tipo == "dieta" {
			familia = d.Concepto
		}
		subtotal, ok := subtotales[familia]
		if !ok {
			return nil, ErrPreparacion
		}
		// El dominio ya validó importes no negativos y totales sin desbordamiento.
		// Agrupar esos céntimos no aplica tarifas ni vuelve a decidir la propuesta.
		subtotal.OriginalCentimos += l.OriginalCentimos
		subtotal.ReconocidoPropuestoCentimos += l.ReconocidoPropuestoCentimos
		subtotal.RechazadoCentimos += l.RechazadoCentimos
		concepto := t.Conceptos[d.Concepto]
		detalle := ""
		if d.Tipo == "kilometraje" {
			concepto = t.Tipos[d.Tipo]
		} else if d.Tipo == domain.ClaseOtroMedio || d.Tipo == domain.ClaseOtroGasto {
			concepto = t.TiposGasto[d.TipoGasto]
			if concepto == "" {
				return nil, ErrTextos
			}
			if d.CatalogoVersion == "" || d.JustificanteRef == nil || d.JustificanteSHA256 == nil || *d.JustificanteRef == "" || *d.JustificanteSHA256 == "" || s.Catalogo.CatalogoOtrosGastosVersionRef != d.CatalogoVersion {
				return nil, ErrPreparacion
			}
			for _, rotulo := range []string{t.Rotulos.FechaGasto, t.Rotulos.DescripcionDeclarada, t.Rotulos.JustificanteRef, t.Rotulos.JustificanteHuella, t.Rotulos.CatalogoOtrosGastos, t.Rotulos.TopeLinea, t.Rotulos.JustificanteLimite} {
				if rotulo == "" {
					return nil, ErrTextos
				}
			}
		} else if d.Tipo != "dieta" {
			return nil, ErrPreparacion
		}
		if concepto == "" {
			return nil, ErrTextos
		}
		if d.Fecha != "" {
			fecha, err := time.Parse("2006-01-02", d.Fecha)
			if err != nil {
				return nil, ErrPreparacion
			}
			detalle = fecha.Format(t.Formato.Fecha)
		}
		motivo := t.Rotulos.SinReduccion
		if l.MotivoCodigo != "" {
			motivo = t.Motivos[l.MotivoCodigo]
			if motivo == "" {
				return nil, ErrTextos
			}
		}
		f := fila{Concepto: concepto, Detalle: detalle, Inicial: moneda(l.OriginalCentimos, t.Formato), Propuesto: moneda(l.ReconocidoPropuestoCentimos, t.Formato), Diferencia: moneda(l.RechazadoCentimos, t.Formato), Motivo: motivo, Regla: l.ReglaRef, Numero: l.Indice + 1}
		if d.Tipo == "kilometraje" {
			for _, rotulo := range []string{t.Rotulos.Ruta, t.Rotulos.OrigenCodigo, t.Rotulos.DestinoCodigo, t.Rotulos.KilometrosBase, t.Rotulos.KilometrosFinales, t.Rotulos.AjusteKilometros, t.Rotulos.MotivoAjuste, t.Rotulos.NoConsta} {
				if !texto(rotulo) {
					return nil, ErrTextos
				}
			}
			f.Kilometraje = true
			f.RutaNumero = d.RutaIndice
			f.OrigenCodigo = d.OrigenCodigo
			f.DestinoCodigo = d.DestinoCodigo
			f.KilometrosBase = distancia(d.KilometrosBase, t)
			f.KilometrosFinales = distancia(d.Kilometros, t)
			f.AjusteKilometros = distancia(d.AjusteKilometros, t)
			f.MotivoAjuste = d.MotivoAjuste
			if f.MotivoAjuste == "" {
				f.MotivoAjuste = t.Rotulos.NoConsta
			}
		}
		if d.Tipo == domain.ClaseOtroMedio || d.Tipo == domain.ClaseOtroGasto {
			regla, ok := reglas[l.ReglaRef]
			if !ok || regla.Tipo != d.Tipo || regla.Concepto != d.TipoGasto || regla.TopeCentimos <= 0 {
				return nil, ErrPreparacion
			}
			f.D5 = true
			v.TieneD5 = true
			f.Fecha = detalle
			f.Detalle = ""
			f.Descripcion = d.Concepto
			f.JustificanteRef = *d.JustificanteRef
			f.JustificanteSHA = *d.JustificanteSHA256
			f.CatalogoOtros = d.CatalogoVersion
			f.Tope = moneda(regla.TopeCentimos, t.Formato)
		}
		v.Filas = append(v.Filas, f)
	}
	var comprobacion domain.TotalesLiquidacionPropuesta
	for _, codigo := range familiasInforme {
		subtotal := subtotales[codigo]
		v.Resumen = append(v.Resumen, fila{Concepto: t.Familias[codigo], Inicial: moneda(subtotal.OriginalCentimos, t.Formato), Propuesto: moneda(subtotal.ReconocidoPropuestoCentimos, t.Formato), Diferencia: moneda(subtotal.RechazadoCentimos, t.Formato)})
		comprobacion.OriginalCentimos += subtotal.OriginalCentimos
		comprobacion.ReconocidoPropuestoCentimos += subtotal.ReconocidoPropuestoCentimos
		comprobacion.RechazadoCentimos += subtotal.RechazadoCentimos
	}
	if comprobacion != s.Totales {
		return nil, ErrPreparacion
	}
	v.TotalInicial = moneda(s.Totales.OriginalCentimos, t.Formato)
	v.TotalPropuesto = moneda(s.Totales.ReconocidoPropuestoCentimos, t.Formato)
	v.TotalDiferencia = moneda(s.Totales.RechazadoCentimos, t.Formato)
	var b bytes.Buffer
	if err := informe.Execute(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// moneda conserva céntimos enteros incluso para los extremos de int64.
func moneda(n int64, f Formato) string {
	negativo := n < 0
	digitos := strconv.FormatInt(n, 10)
	if negativo {
		digitos = digitos[1:]
	}
	if len(digitos) < 3 {
		digitos = strings.Repeat("0", 3-len(digitos)) + digitos
	}
	entero := digitos[:len(digitos)-2]
	for i := len(entero) - 3; i > 0; i -= 3 {
		entero = entero[:i] + f.Agrupacion + entero[i:]
	}
	decimal := digitos[len(digitos)-2:]
	if negativo {
		entero = "-" + entero
	}
	return entero + f.Decimal + decimal + "\u00a0" + f.Moneda
}

// distancia localiza el decimal declarado sin calcular ni completar datos ausentes.
func distancia(s string, t Textos) string {
	if s == "" {
		return t.Rotulos.NoConsta
	}
	if !distanciaDecimal.MatchString(s) {
		return s
	}
	entero, decimal, _ := strings.Cut(s, ".")
	negativo := strings.HasPrefix(entero, "-")
	if negativo {
		entero = entero[1:]
	}
	for i := len(entero) - 3; i > 0; i -= 3 {
		entero = entero[:i] + t.Formato.Agrupacion + entero[i:]
	}
	if negativo {
		entero = "-" + entero
	}
	return entero + t.Formato.Decimal + decimal
}

var distanciaDecimal = regexp.MustCompile(`^-?[0-9]+\.[0-9]{4}$`)
