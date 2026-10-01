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
	Concepto, Detalle, Inicial, Propuesto, Diferencia, Motivo, Regla string
	Numero                                                           int
}
type vista struct {
	Textos                                                                          Textos
	Tema, Estilos                                                                   template.CSS
	ComisionRef, Version, Grupo                                                     string
	Filas                                                                           []fila
	TotalInicial, TotalPropuesto, TotalDiferencia                                   string
	CatalogoRef, CatalogoVersion, Tarifa, DocumentoSHA, CatalogoSHA, PreparacionSHA string
	Fuentes                                                                         []string
}

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
	for _, l := range s.Lineas {
		if l.Indice < 0 || l.Indice >= len(s.Documento.Lineas) {
			return nil, ErrPreparacion
		}
		d := s.Documento.Lineas[l.Indice]
		concepto := t.Conceptos[d.Concepto]
		detalle := ""
		if d.Tipo == "kilometraje" {
			concepto = t.Tipos[d.Tipo]
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
		v.Filas = append(v.Filas, fila{concepto, detalle, moneda(l.OriginalCentimos, t.Formato), moneda(l.ReconocidoPropuestoCentimos, t.Formato), moneda(l.RechazadoCentimos, t.Formato), motivo, l.ReglaRef, l.Indice + 1})
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
