package ports

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	ErrConsultaCategoriasProfesionalesInvalida     = errors.New("personal: consulta de categorias profesionales invalida")
	ErrCatalogoCategoriasProfesionalesNoDisponible = errors.New("personal: catalogo de categorias profesionales no disponible")
)

const maximoCategoriasProfesionalesConsultables = 10_000

var (
	patronIDCatalogoProfesional     = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	patronClaveCategoriaProfesional = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	patronAreaCategoriaProfesional  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)
	patronHuellaCatalogoProfesional = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronRevisionFuenteProfesional = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
)

// ReferenciaCatalogoCategoriasProfesionales inmoviliza la instantanea exacta
// que debe consumir Personal. No existe resolucion implicita de «la ultima».
type ReferenciaCatalogoCategoriasProfesionales struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
}

func (r ReferenciaCatalogoCategoriasProfesionales) Validar() error {
	if !patronIDCatalogoProfesional.MatchString(r.CatalogoID) ||
		r.CatalogoVersion < 1 || !patronHuellaCatalogoProfesional.MatchString(r.CatalogoHuellaSHA256) {
		return ErrConsultaCategoriasProfesionalesInvalida
	}
	return nil
}

// CategoriaProfesionalConsultable contiene solo datos de negocio necesarios
// para Personal. Excluye rutas de origen, actores, aprobaciones y motivos.
type CategoriaProfesionalConsultable struct {
	Clave        string `json:"clave"`
	Etiqueta     string `json:"etiqueta"`
	Descripcion  string `json:"descripcion,omitempty"`
	Orden        int    `json:"orden"`
	Area         string `json:"area"`
	AreaEtiqueta string `json:"area_etiqueta"`
}

// FuenteCategoriasProfesionalesConsultable identifica el origen publicado sin
// revelar rutas, actores, aprobaciones ni otra metadata interna.
type FuenteCategoriasProfesionalesConsultable struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso"`
}

func (f FuenteCategoriasProfesionalesConsultable) Validar() error {
	if !patronRevisionFuenteProfesional.MatchString(f.Revision) || f.ActualizadaEn.IsZero() ||
		f.ActualizadaEn.Location() != time.UTC || f.ActualizadaEn.Nanosecond()%1000 != 0 ||
		(f.Aviso != "" && !textoCategoriaProfesionalValido(f.Aviso, 4096, false)) ||
		(f.Demostracion && !strings.Contains(strings.ToUpper(f.Aviso), "DEMOSTRACI")) {
		return ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	return nil
}

func (c CategoriaProfesionalConsultable) Validar() error {
	if !patronClaveCategoriaProfesional.MatchString(c.Clave) ||
		!textoCategoriaProfesionalValido(c.Etiqueta, 512, false) ||
		(c.Descripcion != "" && !textoCategoriaProfesionalValido(c.Descripcion, 8*1024, true)) ||
		c.Orden < 1 || !patronAreaCategoriaProfesional.MatchString(c.Area) ||
		!textoCategoriaProfesionalValido(c.AreaEtiqueta, 512, false) {
		return ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	return nil
}

// CatalogoCategoriasProfesionalesConsultable es la proyeccion minimizada de
// una unica version publicada del catalogo gobernado.
type CatalogoCategoriasProfesionalesConsultable struct {
	Referencia ReferenciaCatalogoCategoriasProfesionales `json:"referencia"`
	Fuente     FuenteCategoriasProfesionalesConsultable  `json:"fuente"`
	Categorias []CategoriaProfesionalConsultable         `json:"categorias"`
}

func (c CatalogoCategoriasProfesionalesConsultable) Validar() error {
	if err := c.Referencia.Validar(); err != nil || c.Fuente.Validar() != nil || len(c.Categorias) == 0 ||
		len(c.Categorias) > maximoCategoriasProfesionalesConsultables {
		return ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	vistas := make(map[string]struct{}, len(c.Categorias))
	for indice, categoria := range c.Categorias {
		if err := categoria.Validar(); err != nil {
			return err
		}
		if _, existe := vistas[categoria.Clave]; existe {
			return ErrCatalogoCategoriasProfesionalesNoDisponible
		}
		vistas[categoria.Clave] = struct{}{}
		if indice > 0 && categoriaProfesionalMenor(categoria, c.Categorias[indice-1]) {
			return ErrCatalogoCategoriasProfesionalesNoDisponible
		}
	}
	return nil
}

func (c CatalogoCategoriasProfesionalesConsultable) Clonar() CatalogoCategoriasProfesionalesConsultable {
	clon := c
	clon.Categorias = append([]CategoriaProfesionalConsultable(nil), c.Categorias...)
	return clon
}

func OrdenarCategoriasProfesionales(categorias []CategoriaProfesionalConsultable) {
	sort.Slice(categorias, func(i, j int) bool {
		return categoriaProfesionalMenor(categorias[i], categorias[j])
	})
}

func categoriaProfesionalMenor(a, b CategoriaProfesionalConsultable) bool {
	if a.Orden != b.Orden {
		return a.Orden < b.Orden
	}
	return a.Clave < b.Clave
}

func textoCategoriaProfesionalValido(valor string, maximo int, multilinea bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len([]rune(valor)) > maximo {
		return false
	}
	for _, caracter := range valor {
		if unicode.Is(unicode.Cf, caracter) ||
			(unicode.IsControl(caracter) && (!multilinea || (caracter != '\n' && caracter != '\r' && caracter != '\t'))) {
			return false
		}
	}
	return true
}

// ConsultaCategoriasProfesionales es el puerto de lectura compartible por los
// casos de uso de Personal. El instante procede siempre de la aplicacion.
type ConsultaCategoriasProfesionales interface {
	ObtenerVigentes(context.Context, time.Time) (CatalogoCategoriasProfesionalesConsultable, error)
}
