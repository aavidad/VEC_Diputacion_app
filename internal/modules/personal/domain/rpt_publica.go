package domain

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrConsultaRPTPublicaInvalida = errors.New("personal: consulta RPT publica invalida")
	ErrRPTPublicaNoDisponible     = errors.New("personal: RPT publica no disponible")
	patronClaveRPTPublica         = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	patronCodigoPuestoRPTPublica  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{0,63}$`)
	patronHuellaRPTPublica        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// CategoriaRPTPublica y PuestoRPTPublico son proyecciones de la RPT publicada.
// No contienen ocupantes, personas, adscripciones efectivas ni jefaturas.
type CategoriaRPTPublica struct {
	Clave        string   `json:"clave"`
	Denominacion string   `json:"denominacion"`
	Grupos       []string `json:"grupos"`
	Escalas      []string `json:"escalas"`
	Puestos      int      `json:"puestos"`
	Dotacion     int      `json:"dotacion"`
}
type PuestoRPTPublico struct {
	Codigo                             string   `json:"codigo"`
	Denominacion                       string   `json:"denominacion"`
	CentroCodigo                       string   `json:"centro_codigo"`
	Centro                             string   `json:"centro"`
	Delegacion                         string   `json:"delegacion"`
	Grupos                             []string `json:"grupos"`
	Escala                             string   `json:"escala"`
	CategoriaClave                     string   `json:"categoria_clave"`
	NivelDestino                       int      `json:"nivel_destino"`
	ComplementoEspecificoAnualCentimos int      `json:"complemento_especifico_anual_centimos"`
	Dotacion                           int      `json:"dotacion"`
	Tipo                               string   `json:"tipo"`
	Provision                          string   `json:"provision"`
}
type ResumenRPTPublica struct {
	Puestos    int `json:"puestos"`
	Dotacion   int `json:"dotacion"`
	Categorias int `json:"categorias"`
	Centros    int `json:"centros"`
}
type FuenteRPTPublica struct {
	Documento    string `json:"documento"`
	Importacion  string `json:"importacion"`
	GeneradoEn   string `json:"generado_en"`
	Aviso        string `json:"aviso"`
	HuellaSHA256 string `json:"huella_sha256"`
}
type CatalogoRPTPublica struct {
	Esquema    string                `json:"esquema"`
	Fuente     FuenteRPTPublica      `json:"fuente"`
	Resumen    ResumenRPTPublica     `json:"resumen"`
	Categorias []CategoriaRPTPublica `json:"categorias"`
	Puestos    []PuestoRPTPublico    `json:"puestos"`
}

func (c CategoriaRPTPublica) Validar() error {
	if !patronClaveRPTPublica.MatchString(c.Clave) || !textoRPTPublico(c.Denominacion, 512) || c.Puestos < 0 || c.Dotacion < 0 || len(c.Grupos) == 0 {
		return ErrRPTPublicaNoDisponible
	}
	for _, lista := range [][]string{c.Grupos, c.Escalas} {
		for _, valor := range lista {
			if !textoRPTPublico(valor, 64) {
				return ErrRPTPublicaNoDisponible
			}
		}
	}
	return nil
}
func (p PuestoRPTPublico) Validar() error {
	if !patronCodigoPuestoRPTPublica.MatchString(p.Codigo) || !textoRPTPublico(p.Denominacion, 512) || !textoRPTPublico(p.CentroCodigo, 64) || !textoRPTPublico(p.Centro, 512) || !textoRPTPublico(p.Delegacion, 512) || p.NivelDestino < 0 || p.NivelDestino > 99 || p.ComplementoEspecificoAnualCentimos < 0 || p.Dotacion < 0 || !textoRPTPublico(p.Tipo, 64) || !textoRPTPublico(p.Provision, 128) {
		return ErrRPTPublicaNoDisponible
	}
	if p.Escala != "" && !textoRPTPublico(p.Escala, 64) {
		return ErrRPTPublicaNoDisponible
	}
	if p.CategoriaClave != "" && !patronClaveRPTPublica.MatchString(p.CategoriaClave) {
		return ErrRPTPublicaNoDisponible
	}
	for _, grupo := range p.Grupos {
		if !textoRPTPublico(grupo, 64) {
			return ErrRPTPublicaNoDisponible
		}
	}
	return nil
}
func (c CatalogoRPTPublica) Validar() error {
	if c.Esquema != "vec.catalogo.rpt.v1" || !textoRPTPublico(c.Fuente.Documento, 1024) || !textoRPTPublico(c.Fuente.Importacion, 256) || !textoRPTPublico(c.Fuente.GeneradoEn, 64) || !textoRPTPublico(c.Fuente.Aviso, 4096) || !patronHuellaRPTPublica.MatchString(c.Fuente.HuellaSHA256) || len(c.Categorias) == 0 || len(c.Categorias) > 10000 || len(c.Puestos) == 0 || len(c.Puestos) > 100000 {
		return ErrRPTPublicaNoDisponible
	}
	vistas, codigos, centros := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	dotacion := 0
	for _, categoria := range c.Categorias {
		if categoria.Validar() != nil {
			return ErrRPTPublicaNoDisponible
		}
		if _, ok := vistas[categoria.Clave]; ok {
			return ErrRPTPublicaNoDisponible
		}
		vistas[categoria.Clave] = struct{}{}
	}
	for _, puesto := range c.Puestos {
		if puesto.Validar() != nil {
			return ErrRPTPublicaNoDisponible
		}
		if _, ok := codigos[puesto.Codigo]; ok {
			return ErrRPTPublicaNoDisponible
		}
		codigos[puesto.Codigo] = struct{}{}
		centros[puesto.CentroCodigo] = struct{}{}
		dotacion += puesto.Dotacion
	}
	if c.Resumen.Puestos != len(c.Puestos) || c.Resumen.Dotacion != dotacion || c.Resumen.Categorias != len(c.Categorias) || c.Resumen.Centros != len(centros) {
		return ErrRPTPublicaNoDisponible
	}
	return nil
}
func (c CatalogoRPTPublica) Clonar() CatalogoRPTPublica {
	salida := c
	salida.Categorias = make([]CategoriaRPTPublica, len(c.Categorias))
	salida.Puestos = make([]PuestoRPTPublico, len(c.Puestos))
	for i, categoria := range c.Categorias {
		salida.Categorias[i] = categoria.Clonar()
	}
	for i, puesto := range c.Puestos {
		salida.Puestos[i] = puesto.Clonar()
	}
	return salida
}
func (c CategoriaRPTPublica) Clonar() CategoriaRPTPublica {
	salida := c
	salida.Grupos = copiarListaRPT(c.Grupos)
	salida.Escalas = copiarListaRPT(c.Escalas)
	return salida
}
func (p PuestoRPTPublico) Clonar() PuestoRPTPublico {
	salida := p
	salida.Grupos = copiarListaRPT(p.Grupos)
	return salida
}
func copiarListaRPT(valores []string) []string {
	salida := make([]string, len(valores))
	copy(salida, valores)
	return salida
}
func OrdenarCategoriasRPTPublica(categorias []CategoriaRPTPublica) {
	sort.Slice(categorias, func(i, j int) bool {
		return categorias[i].Denominacion < categorias[j].Denominacion || categorias[i].Denominacion == categorias[j].Denominacion && categorias[i].Clave < categorias[j].Clave
	})
}
func OrdenarPuestosRPTPublica(puestos []PuestoRPTPublico) {
	sort.Slice(puestos, func(i, j int) bool {
		return puestos[i].Denominacion < puestos[j].Denominacion || puestos[i].Denominacion == puestos[j].Denominacion && puestos[i].Codigo < puestos[j].Codigo
	})
}
func textoRPTPublico(valor string, maximo int) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len([]rune(valor)) <= maximo && !strings.ContainsAny(valor, "\x00\r\n\t")
}
