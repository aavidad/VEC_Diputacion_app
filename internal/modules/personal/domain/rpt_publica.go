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
	patronHuellaRPTPublica        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type CategoriaRPTPublica struct {
	Clave        string   `json:"clave"`
	Denominacion string   `json:"denominacion"`
	Grupos       []string `json:"grupos"`
	Escalas      []string `json:"escalas"`
	Puestos      int      `json:"puestos"`
	Dotacion     int      `json:"dotacion"`
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
	Categorias []CategoriaRPTPublica `json:"categorias"`
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
func (c CatalogoRPTPublica) Validar() error {
	if c.Esquema != "vec.catalogo.rpt.v1" || !textoRPTPublico(c.Fuente.Documento, 1024) || !textoRPTPublico(c.Fuente.Importacion, 256) || !textoRPTPublico(c.Fuente.GeneradoEn, 64) || !textoRPTPublico(c.Fuente.Aviso, 4096) || !patronHuellaRPTPublica.MatchString(c.Fuente.HuellaSHA256) || len(c.Categorias) == 0 || len(c.Categorias) > 10000 {
		return ErrRPTPublicaNoDisponible
	}
	vistas := map[string]struct{}{}
	for _, categoria := range c.Categorias {
		if categoria.Validar() != nil {
			return ErrRPTPublicaNoDisponible
		}
		if _, ok := vistas[categoria.Clave]; ok {
			return ErrRPTPublicaNoDisponible
		}
		vistas[categoria.Clave] = struct{}{}
	}
	return nil
}
func (c CatalogoRPTPublica) Clonar() CatalogoRPTPublica {
	salida := c
	salida.Categorias = make([]CategoriaRPTPublica, len(c.Categorias))
	for i, categoria := range c.Categorias {
		salida.Categorias[i] = categoria.Clonar()
	}
	return salida
}

// Clonar conserva listas vacías como arrays JSON, nunca como null.
func (c CategoriaRPTPublica) Clonar() CategoriaRPTPublica {
	salida := c
	salida.Grupos = make([]string, len(c.Grupos))
	copy(salida.Grupos, c.Grupos)
	salida.Escalas = make([]string, len(c.Escalas))
	copy(salida.Escalas, c.Escalas)
	return salida
}
func OrdenarCategoriasRPTPublica(categorias []CategoriaRPTPublica) {
	sort.Slice(categorias, func(i, j int) bool {
		return categorias[i].Denominacion < categorias[j].Denominacion || categorias[i].Denominacion == categorias[j].Denominacion && categorias[i].Clave < categorias[j].Clave
	})
}
func textoRPTPublico(valor string, maximo int) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len([]rune(valor)) <= maximo && !strings.ContainsAny(valor, "\x00\r\n\t")
}
