// Package catalogosvec adapta el catalogo configurable del nucleo a la
// consulta minimizada de categorias profesionales de Personal.
package catalogosvec

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	puertospersonal "vec-diputacion-granada/internal/modules/personal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	_ puertospersonal.ConsultaCategoriasProfesionales = (*ConsultaCategoriasProfesionales)(nil)

	patronClaveCategoria = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	patronAreaCategoria  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)
)

// ConsultaCategoriasProfesionales fija ID, version y huella en su
// construccion. Una seleccion distinta exige otra composicion explicita.
type ConsultaCategoriasProfesionales struct {
	consulta   fuenteCategoriasProfesionales
	referencia puertospersonal.ReferenciaCatalogoCategoriasProfesionales
}

type fuenteCategoriasProfesionales interface {
	puertosvec.ConsultaCatalogosConfigurables
	puertosvec.ConsultaMetadatosFuenteCatalogos
}

func NuevaConsultaCategoriasProfesionales(
	consulta fuenteCategoriasProfesionales,
	referencia puertospersonal.ReferenciaCatalogoCategoriasProfesionales,
) (*ConsultaCategoriasProfesionales, error) {
	if dependenciaCatalogosNula(consulta) || referencia.Validar() != nil {
		return nil, puertospersonal.ErrConsultaCategoriasProfesionalesInvalida
	}
	return &ConsultaCategoriasProfesionales{consulta: consulta, referencia: referencia}, nil
}

func (c *ConsultaCategoriasProfesionales) ObtenerVigentes(
	ctx context.Context,
	instante time.Time,
) (puertospersonal.CatalogoCategoriasProfesionalesConsultable, error) {
	if ctx == nil || c == nil || dependenciaCatalogosNula(c.consulta) ||
		c.referencia.Validar() != nil || instante.IsZero() {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, puertospersonal.ErrConsultaCategoriasProfesionalesInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	metadatos, err := c.consulta.ObtenerMetadatosFuenteCatalogos(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
		}
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, errors.Join(
			puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible, err,
		)
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	instante = instante.UTC().Truncate(time.Microsecond)
	catalogo, err := c.consulta.ObtenerCatalogo(ctx, c.referencia.CatalogoID, c.referencia.CatalogoVersion)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
		}
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, errors.Join(
			puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible, err,
		)
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil || canonico.ID != c.referencia.CatalogoID ||
		canonico.Version != c.referencia.CatalogoVersion ||
		canonico.Estado != dominiovec.EstadoCatalogoPublicado ||
		instante.Before(canonico.PublicadoEn.UTC()) {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil || !huellasCatalogoIguales(huella, c.referencia.CatalogoHuellaSHA256) {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}

	resultado := puertospersonal.CatalogoCategoriasProfesionalesConsultable{
		Referencia: c.referencia,
		Fuente: puertospersonal.FuenteCategoriasProfesionalesConsultable{
			Revision:      metadatos.Revision,
			ActualizadaEn: metadatos.ActualizadaEn,
			Demostracion:  metadatos.Demostracion,
			Aviso:         metadatos.Aviso,
		},
		Categorias: make([]puertospersonal.CategoriaProfesionalConsultable, 0, len(canonico.Entradas)),
	}
	for _, entrada := range canonico.Entradas {
		if err := ctx.Err(); err != nil {
			return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
		}
		if !entrada.VigenteEn(instante) {
			continue
		}
		categoria, err := proyectarCategoriaProfesional(entrada)
		if err != nil {
			return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
		}
		resultado.Categorias = append(resultado.Categorias, categoria)
	}
	puertospersonal.OrdenarCategoriasProfesionales(resultado.Categorias)
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	if err := resultado.Validar(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return puertospersonal.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	return resultado.Clonar(), nil
}

func proyectarCategoriaProfesional(
	entrada dominiovec.EntradaCatalogoConfigurable,
) (puertospersonal.CategoriaProfesionalConsultable, error) {
	area := entrada.Atributos["area"]
	areaEtiqueta := entrada.Atributos["area_etiqueta"]
	if !patronClaveCategoria.MatchString(entrada.Clave) || !patronAreaCategoria.MatchString(area) ||
		strings.TrimSpace(areaEtiqueta) == "" || areaEtiqueta != strings.TrimSpace(areaEtiqueta) {
		return puertospersonal.CategoriaProfesionalConsultable{}, puertospersonal.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	categoria := puertospersonal.CategoriaProfesionalConsultable{
		Clave:        entrada.Clave,
		Etiqueta:     entrada.Etiqueta,
		Descripcion:  entrada.Descripcion,
		Orden:        entrada.Orden,
		Area:         area,
		AreaEtiqueta: areaEtiqueta,
	}
	if err := categoria.Validar(); err != nil {
		return puertospersonal.CategoriaProfesionalConsultable{}, err
	}
	return categoria, nil
}

func huellasCatalogoIguales(a, b string) bool {
	bytesA, errA := hex.DecodeString(a)
	bytesB, errB := hex.DecodeString(b)
	return errA == nil && errB == nil && len(bytesA) == 32 && len(bytesB) == 32 &&
		subtle.ConstantTimeCompare(bytesA, bytesB) == 1
}

func dependenciaCatalogosNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
