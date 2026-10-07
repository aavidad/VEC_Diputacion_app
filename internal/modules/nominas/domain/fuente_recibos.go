package domain

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrFuenteRecibosInvalida = errors.New("nominas: consulta o pagina de fuente invalida")

const LimitePaginaRecibos = 100

// ConsultaFuenteRecibos recibe persona y relación resueltas por sus consumidores.
// EntidadPagadoraRef es un filtro opcional: la fuente aporta la entidad de cada
// descriptor. Este contrato no acredita identidad, relación ni autorización.
// VersionFuente, CoberturaRef y TotalEsperado quedan vacíos solo en la primera
// consulta; el consumidor los fija con la respuesta para recorrer la misma vista.
type ConsultaFuenteRecibos struct {
	PersonaRef         string
	RelacionRef        string
	EntidadPagadoraRef string
	PeriodoNomina      string
	VersionFuente      string
	CoberturaRef       string
	TotalEsperado      int
	Cursor             string
	Limite             int
}

// DescriptorRecibo identifica el documento original sin transportar contenido
// económico, localización de descarga ni una afirmación de firma o custodia.
type DescriptorRecibo struct {
	PersonaRef         string
	RelacionRef        string
	EntidadPagadoraRef string
	PeriodoNomina      string
	OrigenRef          string
	ReciboRef          string
	VersionRecibo      string
	CustodioRef        string
}

// PaginaFuenteRecibos liga total y continuación al filtro, versión y cobertura
// de esta respuesta. La fuente debe mantener estable esa vista al paginar.
type PaginaFuenteRecibos struct {
	Consulta        ConsultaFuenteRecibos
	VersionFuente   string
	CoberturaRef    string
	Total           int
	Recibos         []DescriptorRecibo
	CursorSiguiente string
}

func ValidarConsultaFuenteRecibos(c ConsultaFuenteRecibos) error {
	if !referenciaFuenteValida(c.PersonaRef, false) || !referenciaFuenteValida(c.RelacionRef, false) ||
		!referenciaFuenteValida(c.EntidadPagadoraRef, true) || !referenciaFuenteValida(c.PeriodoNomina, false) ||
		!referenciaFuenteValida(c.VersionFuente, true) || !referenciaFuenteValida(c.CoberturaRef, true) ||
		!referenciaFuenteValida(c.Cursor, true) || c.Limite < 1 || c.Limite > LimitePaginaRecibos ||
		(c.VersionFuente == "") != (c.CoberturaRef == "") ||
		(c.Cursor == "" && c.TotalEsperado != 0) ||
		(c.Cursor != "" && (c.VersionFuente == "" || c.TotalEsperado < 1)) {
		return ErrFuenteRecibosInvalida
	}
	return nil
}

// ValidarPaginaFuenteRecibos se aplica antes de usar o revelar un resultado.
// No transforma referencias del proveedor ni decide si el actor puede leerlas.
func ValidarPaginaFuenteRecibos(c ConsultaFuenteRecibos, p PaginaFuenteRecibos) error {
	if ValidarConsultaFuenteRecibos(c) != nil || p.Consulta != c ||
		!referenciaFuenteValida(p.VersionFuente, false) || !referenciaFuenteValida(p.CoberturaRef, false) ||
		c.VersionFuente != "" && p.VersionFuente != c.VersionFuente ||
		c.CoberturaRef != "" && p.CoberturaRef != c.CoberturaRef ||
		!referenciaFuenteValida(p.CursorSiguiente, true) || p.CursorSiguiente != "" && p.CursorSiguiente == c.Cursor ||
		p.Total < 0 || c.Cursor != "" && p.Total != c.TotalEsperado ||
		len(p.Recibos) > c.Limite || len(p.Recibos) > p.Total ||
		(len(p.Recibos) == 0 && (p.Total != 0 || p.CursorSiguiente != "")) ||
		(p.Total == 0 && c.Cursor != "") ||
		(c.Cursor == "" && p.Total > len(p.Recibos) && p.CursorSiguiente == "") ||
		(c.Cursor == "" && p.Total == len(p.Recibos) && p.CursorSiguiente != "") {
		return ErrFuenteRecibosInvalida
	}
	vistos := make(map[string]struct{}, len(p.Recibos))
	for _, r := range p.Recibos {
		if r.PersonaRef != c.PersonaRef || r.RelacionRef != c.RelacionRef ||
			!referenciaFuenteValida(r.EntidadPagadoraRef, false) ||
			c.EntidadPagadoraRef != "" && r.EntidadPagadoraRef != c.EntidadPagadoraRef ||
			r.PeriodoNomina != c.PeriodoNomina ||
			!referenciaFuenteValida(r.OrigenRef, false) || !referenciaFuenteValida(r.ReciboRef, false) ||
			!referenciaFuenteValida(r.VersionRecibo, false) || !referenciaFuenteValida(r.CustodioRef, false) {
			return ErrFuenteRecibosInvalida
		}
		// Una versión sustituida es otra entrada histórica. Un mismo original y
		// versión no puede aparecer dos veces en la página.
		clave := r.OrigenRef + "\x00" + r.ReciboRef + "\x00" + r.VersionRecibo
		if _, existe := vistos[clave]; existe {
			return ErrFuenteRecibosInvalida
		}
		vistos[clave] = struct{}{}
	}
	return nil
}

// Solo se valida la forma; procedencia y opacidad semántica son obligaciones del
// adaptador admitido. No se recortan ni normalizan referencias originales.
// Se excluyen rutas, URL y caracteres de consulta para no transportar enlaces.
func referenciaFuenteValida(s string, opcional bool) bool {
	if s == "" {
		return opcional
	}
	if len(s) > 256 || !utf8.ValidString(s) {
		return false
	}
	minusculas := strings.ToLower(s)
	for _, esquema := range []string{"http:", "https:", "ftp:", "file:", "data:", "blob:", "javascript:", "mailto:"} {
		if strings.HasPrefix(minusculas, esquema) {
			return false
		}
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
		switch r {
		case '/', '\\', '?', '#', '&', '=', '%', '@':
			return false
		}
	}
	return true
}
