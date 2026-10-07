// Package informesaldo convierte la proyección mínima en un documento neutral
// y usa el renderer común. No consulta personas ni concede exportaciones.
package informesaldo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type Catalogo struct {
	Referencia      string            `json:"referencia"`
	Version         string            `json:"version"`
	Idioma          string            `json:"idioma"`
	FormatoFecha    string            `json:"formato_fecha"`
	Titulo          string            `json:"titulo"`
	Periodo         string            `json:"periodo"`
	Previsto        string            `json:"previsto"`
	Trabajado       string            `json:"trabajado"`
	Saldo           string            `json:"saldo"`
	Duracion        string            `json:"duracion"`
	DuracionCorta   string            `json:"duracion_corta"`
	Desconocido     string            `json:"desconocido"`
	Estados         map[string]string `json:"estados"`
	Estado          string            `json:"estado"`
	Limite          string            `json:"limite"`
	Sintetico       string            `json:"sintetico"`
	NombreSintetico string            `json:"nombre_sintetico"`
}

type Preparador struct {
	renderer        vecports.RenderizadorDocumento
	catalogo        Catalogo
	catalogoVersion int64
	huella          string
	impresor        *message.Printer
}

func Nuevo(renderer vecports.RenderizadorDocumento, datos io.Reader) (*Preparador, error) {
	if dependenciaNula(renderer) || dependenciaNula(datos) || renderer.Formato() != vecdomain.FormatoDocumentoPDF {
		return nil, ports.ErrExportacionSaldoNoDisponible
	}
	raw, err := io.ReadAll(io.LimitReader(datos, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	var c Catalogo
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&c) != nil || decoder.Decode(new(any)) != io.EOF {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	version, err := strconv.ParseInt(c.Version, 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != c.Version {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	idioma, err := language.Parse(c.Idioma)
	if err != nil || idioma.String() != c.Idioma || !texto(c.Referencia, 512) || c.FormatoFecha == "" || len(c.FormatoFecha) > 32 {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	for _, valor := range []string{c.Titulo, c.Periodo, c.Previsto, c.Trabajado, c.Saldo, c.Duracion, c.DuracionCorta, c.Desconocido, c.Estado, c.Limite, c.Sintetico, c.NombreSintetico} {
		if !texto(valor, 2048) {
			return nil, ports.ErrExportacionSaldoInvalida
		}
	}
	if len(c.Estados) != 3 {
		return nil, ports.ErrExportacionSaldoInvalida
	}
	for _, estado := range []string{ports.EstadoSaldoDisponible, ports.EstadoSaldoNoDisponible, ports.EstadoSaldoIncompleto} {
		if !texto(c.Estados[estado], 256) {
			return nil, ports.ErrExportacionSaldoInvalida
		}
	}
	// Las sustituciones son cerradas; una etiqueta sin marcador perdería el dato.
	plantillas := []struct {
		valor  string
		claves []string
	}{
		{c.Periodo, []string{"desde", "hasta"}}, {c.Previsto, []string{"valor"}}, {c.Trabajado, []string{"valor"}}, {c.Saldo, []string{"valor"}}, {c.Duracion, []string{"horas", "minutos"}}, {c.DuracionCorta, []string{"minutos"}}, {c.Estado, []string{"valor"}}, {c.NombreSintetico, []string{"nombre"}},
	}
	for _, p := range plantillas {
		restante := p.valor
		for _, clave := range p.claves {
			marcador := "{{" + clave + "}}"
			if strings.Count(restante, marcador) != 1 {
				return nil, ports.ErrExportacionSaldoInvalida
			}
			restante = strings.ReplaceAll(restante, marcador, "")
		}
		if strings.Contains(restante, "{{") || strings.Contains(restante, "}}") {
			return nil, ports.ErrExportacionSaldoInvalida
		}
	}
	sum := sha256.Sum256(raw)
	return &Preparador{renderer: renderer, catalogo: c, catalogoVersion: version, huella: hex.EncodeToString(sum[:]), impresor: message.NewPrinter(idioma)}, nil
}

func (p *Preparador) PrepararInformeSaldo(ctx context.Context, s ports.SaldoExportable) (ports.DocumentoSaldoPreparado, error) {
	return p.preparar(ctx, s, "")
}

// PrepararEjemploSintetico produce una muestra marcada en el cuerpo del PDF.
// No es confirmación, recibo, exportación autorizada ni autoridad conectable.
func (p *Preparador) PrepararEjemploSintetico(ctx context.Context, s ports.SaldoExportable, nombre string) (ports.DocumentoSaldoPreparado, error) {
	if !texto(nombre, 128) {
		return ports.DocumentoSaldoPreparado{}, ports.ErrExportacionSaldoInvalida
	}
	return p.preparar(ctx, s, nombre)
}

func (p *Preparador) preparar(ctx context.Context, s ports.SaldoExportable, nombre string) (ports.DocumentoSaldoPreparado, error) {
	cero := ports.DocumentoSaldoPreparado{}
	if p == nil || ctx == nil || dependenciaNula(p.renderer) || p.impresor == nil || p.huella == "" {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	desde, hasta, err := validarSaldoExportable(s)
	if err != nil {
		return cero, err
	}
	r := s.Resumen
	c := p.catalogo
	parrafos := []string{}
	if nombre != "" {
		parrafos = append(parrafos, c.Sintetico, sustituir(c.NombreSintetico, "nombre", nombre))
	}
	periodo := sustituir(sustituir(c.Periodo, "desde", desde.Format(c.FormatoFecha)), "hasta", hasta.Format(c.FormatoFecha))
	trabajado := r.TrabajadosMinutos
	parrafos = append(parrafos, periodo, sustituir(c.Estado, "valor", c.Estados[r.Estado]), sustituir(c.Previsto, "valor", p.duracion(r.PrevistosMinutos)), sustituir(c.Trabajado, "valor", p.duracion(&trabajado)), sustituir(c.Saldo, "valor", p.duracion(r.SaldoMinutos)), c.Limite)
	contenido, err := p.renderer.Renderizar(ctx, vecdomain.ContenidoDocumento{Titulo: c.Titulo, Parrafos: parrafos})
	if err != nil {
		return cero, err
	}
	if len(contenido) > 2*1024*1024 {
		return cero, ports.ErrExportacionSaldoInvalida
	}
	contenido = append([]byte(nil), contenido...)
	if err := p.renderer.ValidarSalida(ctx, contenido); err != nil {
		return cero, err
	}
	// El renderer común aún no permite seleccionar idioma. Comprobar el idioma
	// real del PDF evita entregar una traducción con metadatos discordantes.
	if !bytes.Contains(contenido, []byte("/Lang ("+c.Idioma+")")) {
		return cero, ports.ErrExportacionSaldoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return ports.DocumentoSaldoPreparado{Contenido: contenido, CatalogoRef: c.Referencia, CatalogoVersion: p.catalogoVersion, CatalogoSHA256: p.huella}, nil
}

func (p *Preparador) duracion(minutos *int64) string {
	if minutos == nil {
		return p.catalogo.Desconocido
	}
	valor := *minutos
	signo := ""
	if valor < 0 {
		signo = "-"
		valor = -valor
	}
	if valor < 60 {
		return signo + sustituir(p.catalogo.DuracionCorta, "minutos", strconv.FormatInt(valor, 10))
	}
	return signo + sustituir(sustituir(p.catalogo.Duracion, "horas", p.impresor.Sprintf("%d", valor/60)), "minutos", strconv.FormatInt(valor%60, 10))
}
func sustituir(plantilla, clave, valor string) string {
	return strings.ReplaceAll(plantilla, "{{"+clave+"}}", valor)
}
func texto(v string, maximo int) bool {
	if strings.TrimSpace(v) == "" || !utf8.ValidString(v) || len(v) > maximo {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

var _ ports.PreparadorInformeSaldo = (*Preparador)(nil)

func dependenciaNula(v any) bool {
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

func validarSaldoExportable(s ports.SaldoExportable) (time.Time, time.Time, error) {
	desde, e1 := time.Parse("2006-01-02", s.Periodo.Desde)
	hasta, e2 := time.Parse("2006-01-02", s.Periodo.Hasta)
	if e1 != nil || e2 != nil || hasta.Before(desde) || hasta.After(desde.AddDate(1, 0, 0)) {
		return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
	}
	// Límite físico de un periodo, sin inventar jornadas o reglas laborales.
	maximo := int64(hasta.Sub(desde)/(24*time.Hour)+1) * 25 * 60
	r := s.Resumen
	if r.TrabajadosMinutos < 0 || r.TrabajadosMinutos > maximo || r.PrevistosMinutos != nil && (*r.PrevistosMinutos < 0 || *r.PrevistosMinutos > maximo) || r.SaldoMinutos != nil && (*r.SaldoMinutos < -maximo || *r.SaldoMinutos > maximo) {
		return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
	}
	switch r.Estado {
	case ports.EstadoSaldoDisponible:
		if r.PrevistosMinutos == nil || r.SaldoMinutos == nil {
			return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
		}
	case ports.EstadoSaldoNoDisponible:
		if r.PrevistosMinutos != nil || r.SaldoMinutos != nil {
			return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
		}
	case ports.EstadoSaldoIncompleto:
		if r.SaldoMinutos != nil {
			return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
		}
	default:
		return time.Time{}, time.Time{}, ports.ErrExportacionSaldoInvalida
	}
	return desde, hasta, nil
}
