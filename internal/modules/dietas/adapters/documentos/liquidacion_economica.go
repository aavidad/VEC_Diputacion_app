package documentos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	dietasdomain "vec-diputacion-granada/internal/modules/dietas/domain"
	vecdocx "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	vecpdf "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrDocumentoLiquidacionNoDisponible = errors.New("documento_liquidacion_no_disponible")

const maximoDocumentoLiquidacion = 4 << 20

// TextosLiquidacionDocumento procede del catálogo de datos del idioma activo.
type TextosLiquidacionDocumento struct {
	Esquema string            `json:"esquema"`
	Idioma  string            `json:"idioma"`
	Titulo  string            `json:"titulo"`
	Estado  string            `json:"estado"`
	Campos  map[string]string `json:"campos"`
	Tipos   map[string]string `json:"tipos"`
	Formato string            `json:"formato"`
}

var camposDocumentoLiquidacion = []string{
	"numero", "comision", "version", "fecha", "titular", "unidad", "linea", "regla", "original", "reconocido", "rechazado", "motivo", "total", "catalogo", "publicacion", "norma", "documento_huella", "catalogo_huella", "preparacion_huella", "liquidacion_huella", "recibo", "auditoria", "evento", "sin_motivo",
	"fecha_gasto", "ruta", "kilometros", "justificante", "justificante_huella", "tipo_gasto",
	"calculado", "diferencia", "total_calculado", "total_diferencia",
}

func CargarTextosLiquidacionDocumento(r io.Reader) (TextosLiquidacionDocumento, error) {
	var t TextosLiquidacionDocumento
	b, err := io.ReadAll(io.LimitReader(r, 16385))
	if err != nil || len(b) > 16384 {
		return t, ErrDocumentoLiquidacionNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&t) != nil || d.Decode(new(any)) != io.EOF || t.Esquema != "dietas-liquidacion-documento-v1" || (t.Idioma != "es" && t.Idioma != "en") || t.Titulo == "" || t.Estado == "" || t.Formato != "EUR" || len(t.Campos) != len(camposDocumentoLiquidacion) || len(t.Tipos) != 4 {
		return TextosLiquidacionDocumento{}, ErrDocumentoLiquidacionNoDisponible
	}
	for _, k := range camposDocumentoLiquidacion {
		if !textoDocumento(t.Campos[k]) {
			return TextosLiquidacionDocumento{}, ErrDocumentoLiquidacionNoDisponible
		}
	}
	for _, k := range []string{"dieta", "kilometraje", "otro_medio", "otro_gasto"} {
		if !textoDocumento(t.Tipos[k]) {
			return TextosLiquidacionDocumento{}, ErrDocumentoLiquidacionNoDisponible
		}
	}
	if !textoDocumento(t.Titulo) || !textoDocumento(t.Estado) {
		return TextosLiquidacionDocumento{}, ErrDocumentoLiquidacionNoDisponible
	}
	return t, nil
}

func textoDocumento(s string) bool {
	if len(s) < 1 || len(s) > 160 || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if r < ' ' || r == 0x7f {
			return false
		}
	}
	return true
}

// RenderizarLiquidacionEconomica sólo admite una instantánea durable que
// supera la reconstrucción de sus dos huellas; no lee reglas vigentes.
func RenderizarLiquidacionEconomica(ctx context.Context, s dietasdomain.InstantaneaLiquidacionEconomica, formato string, t TextosLiquidacionDocumento) ([]byte, string, error) {
	if ctx == nil || (formato != "pdf" && formato != "docx") || (t.Idioma != "es" && t.Idioma != "en") {
		return nil, "", ErrDocumentoLiquidacionNoDisponible
	}
	if _, err := dietasdomain.RecuperarLiquidacionEconomica(s); err != nil {
		return nil, "", ErrDocumentoLiquidacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	c, e := s.Constancia, s.Economia
	normasRegla := make(map[string]string, len(e.Catalogo.Reglas))
	for _, regla := range e.Catalogo.Reglas {
		normasRegla[regla.Referencia] = regla.NormaRef
	}
	p := []string{t.Estado,
		campo(t, "numero", c.NumeroDocumento), campo(t, "comision", e.ComisionRef), campo(t, "version", strconv.FormatUint(c.VersionResultante, 10)),
		campo(t, "fecha", c.Fecha), campo(t, "titular", c.TitularPersonaRef), campo(t, "unidad", c.UnidadRef),
	}
	for i, l := range e.Lineas {
		declarada := e.Documento.Lineas[i]
		tipo := t.Tipos[declarada.Tipo]
		if declarada.Tipo == "otro_gasto" || declarada.Tipo == "otro_medio" {
			tipo += " · " + declarada.Concepto
		}
		motivo := l.MotivoCodigo
		if motivo == "" {
			motivo = t.Campos["sin_motivo"]
		}
		p = append(p, campo(t, "linea", strconv.Itoa(l.Indice+1)+" · "+tipo),
			campo(t, "regla", l.ReglaRef), campo(t, "original", importe(l.OriginalCentimos, t.Idioma)),
			campo(t, "calculado", importe(l.CalculadoPublicadoCentimos, t.Idioma)),
			campo(t, "diferencia", importe(l.DiferenciaCentimos, t.Idioma)),
			campo(t, "reconocido", importe(l.ReconocidoPropuestoCentimos, t.Idioma)),
			campo(t, "rechazado", importe(l.RechazadoCentimos, t.Idioma)), campo(t, "motivo", motivo))
		p = append(p, campo(t, "norma", normasRegla[l.ReglaRef]))
		if declarada.Fecha != "" {
			p = append(p, campo(t, "fecha_gasto", declarada.Fecha))
		}
		if declarada.OrigenCodigo != "" {
			p = append(p, campo(t, "ruta", declarada.OrigenCodigo+" – "+declarada.DestinoCodigo))
		}
		if declarada.Kilometros != "" {
			p = append(p, campo(t, "kilometros", declarada.Kilometros))
		}
		if declarada.JustificanteRef != nil {
			p = append(p, campo(t, "justificante", *declarada.JustificanteRef))
		}
		if declarada.JustificanteSHA256 != nil {
			p = append(p, campo(t, "justificante_huella", *declarada.JustificanteSHA256))
		}
		if declarada.TipoGasto != "" {
			p = append(p, campo(t, "tipo_gasto", declarada.TipoGasto))
		}
	}
	p = append(p, campo(t, "total_calculado", importe(e.Totales.CalculadoPublicadoCentimos, t.Idioma)),
		campo(t, "total_diferencia", importe(e.Totales.DiferenciaCentimos, t.Idioma)),
		campo(t, "total", importe(e.Totales.ReconocidoPropuestoCentimos, t.Idioma)),
		campo(t, "catalogo", e.CatalogoRef+" / "+e.CatalogoVersion), campo(t, "publicacion", e.Catalogo.PublicacionRef))
	for _, n := range e.Catalogo.Normas {
		p = append(p, campo(t, "norma", n.Referencia+" · "+n.Articulo+" · "+n.URL))
	}
	p = append(p, campo(t, "documento_huella", e.DocumentoSHA256), campo(t, "catalogo_huella", e.CatalogoSHA256),
		campo(t, "preparacion_huella", e.SnapshotSHA256), campo(t, "liquidacion_huella", s.SnapshotSHA256),
		campo(t, "recibo", c.ReciboRef), campo(t, "auditoria", c.AuditoriaRef), campo(t, "evento", c.EventoRef))
	contenido := vecdomain.ContenidoDocumento{Titulo: t.Titulo, Parrafos: p}
	var salida []byte
	var err error
	var mime string
	if formato == "pdf" {
		idioma := "es-ES"
		if t.Idioma == "en" {
			idioma = "en-GB"
		}
		r := vecpdf.Renderizador{Idioma: idioma, Membrete: true}
		salida, err = r.Renderizar(ctx, contenido)
		if err == nil {
			err = r.ValidarSalida(ctx, salida)
		}
		mime = "application/pdf"
	} else {
		r := vecdocx.Renderizador{Membrete: true}
		salida, err = r.Renderizar(ctx, contenido)
		if err == nil {
			err = r.ValidarSalida(ctx, salida)
		}
		mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if err != nil || len(salida) == 0 || len(salida) > maximoDocumentoLiquidacion {
		return nil, "", ErrDocumentoLiquidacionNoDisponible
	}
	return salida, mime, nil
}

func campo(t TextosLiquidacionDocumento, k, v string) string {
	return fmt.Sprintf("%s: %s", t.Campos[k], v)
}

func importe(centimos int64, idioma string) string {
	signo := ""
	if centimos < 0 {
		signo = "-"
		centimos = -centimos
	}
	euros := strconv.FormatInt(centimos/100, 10)
	decimal := "."
	miles := ","
	if idioma == "es" {
		decimal, miles = ",", "."
	}
	var b strings.Builder
	for i, r := range euros {
		if i > 0 && (len(euros)-i)%3 == 0 {
			b.WriteString(miles)
		}
		b.WriteRune(r)
	}
	return signo + b.String() + decimal + fmt.Sprintf("%02d", centimos%100) + " €"
}
