package informeperiodo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	app "vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ErrPDFNoDisponible indica que el renderer común no pudo producir un PDF válido.
var ErrPDFNoDisponible = errors.New("pdf_informe_dietas_no_disponible")

const maxFilasPDF = 500

// CatalogoPDF contiene los textos de la muestra PDF. Las plantillas usan
// marcadores {{clave}} que se sustituyen una sola vez; los datos nunca se
// interpretan como plantilla.
type CatalogoPDF struct {
	Esquema          string            `json:"esquema"`
	Referencia       string            `json:"referencia"`
	Version          string            `json:"version"`
	Idioma           string            `json:"idioma"`
	FormatoFecha     string            `json:"formato_fecha"`
	SeparadorDecimal string            `json:"separador_decimal"`
	Titulo           string            `json:"titulo"`
	Sintetico        string            `json:"sintetico"`
	Criterio         string            `json:"criterio"`
	CamposFecha      map[string]string `json:"campos_fecha"`
	Periodo          string            `json:"periodo"`
	PeriodoDesde     string            `json:"periodo_desde"`
	PeriodoHasta     string            `json:"periodo_hasta"`
	Resumen          string            `json:"resumen"`
	Concepto         string            `json:"concepto"`
	Conceptos        map[string]string `json:"conceptos"`
	Fila             string            `json:"fila"`
	Situaciones      map[string]string `json:"situaciones"`
	Importes         map[string]string `json:"importes"`
	Vacio            string            `json:"vacio"`
	Limite           string            `json:"limite"`
}

type plantillaPDF struct {
	texto      string
	marcadores []string
}

// plantillasPDF fija qué marcadores debe contener exactamente una vez cada texto.
func plantillasPDF(c CatalogoPDF) []plantillaPDF {
	return []plantillaPDF{
		{c.Titulo, nil}, {c.Sintetico, nil}, {c.Vacio, nil}, {c.Limite, nil},
		{c.Criterio, []string{"campo"}}, {c.Periodo, []string{"desde", "hasta"}},
		{c.PeriodoDesde, []string{"desde"}}, {c.PeriodoHasta, []string{"hasta"}},
		{c.Resumen, []string{"cuenta", "importe"}}, {c.Concepto, []string{"concepto", "importe"}},
		{c.Fila, []string{"referencia", "version", "persona", "unidad", "situacion", "fecha", "importe"}},
	}
}

// CargarCatalogoPDF valida el esquema cerrado del catálogo PDF.
func CargarCatalogoPDF(r io.Reader) (CatalogoPDF, error) {
	var c CatalogoPDF
	raw, err := io.ReadAll(io.LimitReader(r, limiteCatalogo+1))
	if err != nil || len(raw) > limiteCatalogo {
		return c, ErrCatalogoInvalido
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || dec.Decode(new(any)) != io.EOF {
		return CatalogoPDF{}, ErrCatalogoInvalido
	}
	version, err := strconv.Atoi(c.Version)
	idioma, errIdioma := language.Parse(c.Idioma)
	if err != nil || errIdioma != nil || version < 1 || strconv.Itoa(version) != c.Version ||
		idioma.String() != c.Idioma || c.Esquema != "dietas-informes-pdf-v1" || !textoPDF(c.Referencia, 512) ||
		!textoPDF(c.FormatoFecha, 32) || (c.SeparadorDecimal != "," && c.SeparadorDecimal != ".") ||
		len(c.CamposFecha) != len(app.CamposFecha) || !mapaTextosPDF(c.Conceptos, nil) ||
		!mapaTextosPDF(c.Situaciones, nil) || !mapaTextosPDF(c.Importes, []string{"valor"}) {
		return CatalogoPDF{}, ErrCatalogoInvalido
	}
	// El separador decimal debe casar con la agrupación que aplica el idioma.
	if !strings.Contains(message.NewPrinter(idioma).Sprintf("%.1f", 0.5), c.SeparadorDecimal) {
		return CatalogoPDF{}, ErrCatalogoInvalido
	}
	for _, clave := range app.CamposFecha {
		if !textoPDF(c.CamposFecha[clave], 128) {
			return CatalogoPDF{}, ErrCatalogoInvalido
		}
	}
	for _, p := range plantillasPDF(c) {
		if !textoPDF(p.texto, 2048) || !plantillaExacta(p.texto, p.marcadores) {
			return CatalogoPDF{}, ErrCatalogoInvalido
		}
	}
	return c, nil
}

// PrepararPDF compone el documento con el renderer común y valida su salida.
// Devuelve el PDF completo o un error, nunca bytes parciales.
func PrepararPDF(ctx context.Context, renderer vecports.RenderizadorDocumento, c CatalogoPDF, inf app.Informe) ([]byte, error) {
	if ctx == nil || nuloPDF(renderer) || renderer.Formato() != vecdomain.FormatoDocumentoPDF {
		return nil, ErrPDFNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	campo, ok := c.CamposFecha[inf.CampoFecha]
	plantillaImporte, okImporte := c.Importes[inf.Moneda]
	if !ok || !okImporte {
		return nil, ErrCatalogoInvalido
	}
	if len(inf.Filas) > maxFilasPDF || len(inf.ConceptosCentimos) != len(inf.Conceptos) || inf.TotalCentimos < 0 {
		return nil, app.ErrDatosInvalidos
	}
	for _, v := range inf.ConceptosCentimos {
		if v < 0 {
			return nil, app.ErrDatosInvalidos
		}
	}
	impresor := message.NewPrinter(language.MustParse(c.Idioma))
	importe := func(centimos int64) string {
		valor := impresor.Sprintf("%d", centimos/100) + c.SeparadorDecimal + strconv.FormatInt(100+centimos%100, 10)[1:]
		return strings.Replace(plantillaImporte, "{{valor}}", valor, 1)
	}
	parrafos := []string{c.Sintetico, sustituirPDF(c.Criterio, "campo", campo)}
	switch {
	case inf.Desde != nil && inf.Hasta != nil:
		parrafos = append(parrafos, sustituirPDF(c.Periodo, "desde", inf.Desde.Format(c.FormatoFecha), "hasta", inf.Hasta.Format(c.FormatoFecha)))
	case inf.Desde != nil:
		parrafos = append(parrafos, sustituirPDF(c.PeriodoDesde, "desde", inf.Desde.Format(c.FormatoFecha)))
	case inf.Hasta != nil:
		parrafos = append(parrafos, sustituirPDF(c.PeriodoHasta, "hasta", inf.Hasta.Format(c.FormatoFecha)))
	}
	parrafos = append(parrafos, sustituirPDF(c.Resumen, "cuenta", impresor.Sprintf("%d", len(inf.Filas)), "importe", importe(inf.TotalCentimos)))
	for i, clave := range inf.Conceptos {
		rotulo, ok := c.Conceptos[clave]
		if !ok {
			return nil, ErrCatalogoInvalido
		}
		parrafos = append(parrafos, sustituirPDF(c.Concepto, "concepto", rotulo, "importe", importe(inf.ConceptosCentimos[i])))
	}
	for _, f := range inf.Filas {
		situacion, ok := c.Situaciones[f.Situacion]
		if !ok {
			return nil, ErrCatalogoInvalido
		}
		if !textoPDF(f.Referencia, 256) || !textoPDF(f.Persona, 256) || !textoPDF(f.Unidad, 256) ||
			f.ImporteIncluidoCentimos < 0 || f.VersionComision < 1 {
			return nil, app.ErrDatosInvalidos
		}
		parrafos = append(parrafos, sustituirPDF(c.Fila, "referencia", f.Referencia,
			"version", strconv.Itoa(f.VersionComision), "persona", f.Persona, "unidad", f.Unidad,
			"situacion", situacion, "fecha", f.Fecha.Format(c.FormatoFecha), "importe", importe(f.ImporteIncluidoCentimos)))
	}
	if len(inf.Filas) == 0 {
		parrafos = append(parrafos, c.Vacio)
	}
	parrafos = append(parrafos, c.Limite)
	contenido, err := renderer.Renderizar(ctx, vecdomain.ContenidoDocumento{Titulo: c.Titulo, Parrafos: parrafos})
	if err != nil {
		return nil, err
	}
	if len(contenido) == 0 || len(contenido) > 4*1024*1024 {
		return nil, ErrPDFNoDisponible
	}
	contenido = append([]byte(nil), contenido...)
	if err := renderer.ValidarSalida(ctx, contenido); err != nil {
		return nil, err
	}
	if !bytes.Contains(contenido, []byte("/Lang ("+c.Idioma+")")) {
		return nil, ErrPDFNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return contenido, nil
}

// sustituirPDF reemplaza todos los marcadores en una sola pasada.
func sustituirPDF(plantilla string, pares ...string) string {
	for i := 0; i < len(pares); i += 2 {
		pares[i] = "{{" + pares[i] + "}}"
	}
	return strings.NewReplacer(pares...).Replace(plantilla)
}

func plantillaExacta(texto string, marcadores []string) bool {
	restante := texto
	for _, m := range marcadores {
		marca := "{{" + m + "}}"
		if strings.Count(restante, marca) != 1 {
			return false
		}
		restante = strings.ReplaceAll(restante, marca, "")
	}
	return !strings.Contains(restante, "{{") && !strings.Contains(restante, "}}")
}

func mapaTextosPDF(m map[string]string, marcadores []string) bool {
	if len(m) == 0 || len(m) > 20 {
		return false
	}
	for clave, v := range m {
		if clave == "" || !textoPDF(v, 256) || !plantillaExacta(v, marcadores) {
			return false
		}
	}
	return true
}

func textoPDF(v string, max int) bool {
	if strings.TrimSpace(v) == "" || len(v) > max || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func nuloPDF(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return r.IsNil()
	}
	return false
}
