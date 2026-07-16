// Package catalogosvec adapta catalogos configurables gobernados por el nucleo
// a las proyecciones publicas minimizadas del modulo Bolsa.
package catalogosvec

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	_ puertosbolsa.ConsultaCategoriasPublicas = (*ConsultaCategorias)(nil)

	patronIDCatalogo  = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	patronClaveKebab  = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	patronAtributoAPI = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)
	patronRevision    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
)

// ConsultaCategorias fija el ID y la version al construirse. Cambiar de
// version es una decision de configuracion explicita; nunca se consulta "la
// ultima" de forma implicita.
type ConsultaCategorias struct {
	fuente     fuenteCatalogosPublicos
	catalogoID string
	version    int
}

type fuenteCatalogosPublicos interface {
	puertosvec.ConsultaCatalogosConfigurables
	puertosvec.ConsultaMetadatosFuenteCatalogos
}

func NuevaConsultaCategorias(
	fuente fuenteCatalogosPublicos,
	catalogoID string,
	version int,
) (*ConsultaCategorias, error) {
	catalogoID = strings.TrimSpace(catalogoID)
	if fuente == nil || !patronIDCatalogo.MatchString(catalogoID) || version < 1 {
		return nil, puertosbolsa.ErrConsultaCategoriasInvalida
	}
	return &ConsultaCategorias{fuente: fuente, catalogoID: catalogoID, version: version}, nil
}

func (c *ConsultaCategorias) ObtenerPublicadas(
	ctx context.Context,
	instante time.Time,
) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	if ctx == nil || c == nil || c.fuente == nil || instante.IsZero() {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrConsultaCategoriasInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, err
	}
	instante = instante.UTC().Truncate(time.Microsecond)
	catalogo, err := c.fuente.ObtenerCatalogo(ctx, c.catalogoID, c.version)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return puertosbolsa.CatalogoCategoriasPublicas{}, err
		}
		return puertosbolsa.CatalogoCategoriasPublicas{}, errors.Join(puertosbolsa.ErrCatalogoCategoriasNoDisponible, err)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, err
	}
	metadatos, err := c.fuente.ObtenerMetadatosFuenteCatalogos(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return puertosbolsa.CatalogoCategoriasPublicas{}, err
		}
		return puertosbolsa.CatalogoCategoriasPublicas{}, errors.Join(puertosbolsa.ErrCatalogoCategoriasNoDisponible, err)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil || canonico.ID != c.catalogoID || canonico.Version != c.version ||
		canonico.Estado != dominiovec.EstadoCatalogoPublicado || instante.Before(canonico.PublicadoEn.UTC()) {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil || validarMetadatosFuente(metadatos) != nil {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}

	resultado := puertosbolsa.CatalogoCategoriasPublicas{
		ID:           canonico.ID,
		Version:      canonico.Version,
		HuellaSHA256: huella,
		Fuente: puertosbolsa.MetadatosFuenteCategorias{
			Revision:      metadatos.Revision,
			ActualizadaEn: metadatos.ActualizadaEn,
			Demostracion:  metadatos.Demostracion,
			Aviso:         metadatos.Aviso,
		},
		Categorias: make([]puertosbolsa.CategoriaPublica, 0, len(canonico.Entradas)),
	}
	for _, entrada := range canonico.Entradas {
		if err := ctx.Err(); err != nil {
			return puertosbolsa.CatalogoCategoriasPublicas{}, err
		}
		if !entrada.VigenteEn(instante) || entrada.Atributos["publicable"] != "si" {
			continue
		}
		categoria, err := proyectarCategoria(entrada, canonico.Version)
		if err != nil {
			return puertosbolsa.CatalogoCategoriasPublicas{}, err
		}
		resultado.Categorias = append(resultado.Categorias, categoria)
	}
	if len(resultado.Categorias) == 0 {
		return puertosbolsa.CatalogoCategoriasPublicas{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	sort.Slice(resultado.Categorias, func(i, j int) bool {
		if resultado.Categorias[i].Orden != resultado.Categorias[j].Orden {
			return resultado.Categorias[i].Orden < resultado.Categorias[j].Orden
		}
		return resultado.Categorias[i].Clave < resultado.Categorias[j].Clave
	})
	return resultado, nil
}

func validarMetadatosFuente(metadatos puertosvec.MetadatosFuenteCatalogos) error {
	if !patronRevision.MatchString(metadatos.Revision) || metadatos.ActualizadaEn.IsZero() ||
		metadatos.ActualizadaEn.Location() != time.UTC || metadatos.ActualizadaEn.Nanosecond()%1000 != 0 ||
		(metadatos.Demostracion && !textoPublicoValido(metadatos.Aviso, 500)) ||
		(!metadatos.Demostracion && metadatos.Aviso != "" && !textoPublicoValido(metadatos.Aviso, 500)) {
		return puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	return nil
}

func textoPublicoValido(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len([]rune(valor)) > maximo {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.Is(unicode.Cf, caracter) {
			return false
		}
	}
	return true
}

func proyectarCategoria(entrada dominiovec.EntradaCatalogoConfigurable, version int) (puertosbolsa.CategoriaPublica, error) {
	area := entrada.Atributos["area"]
	areaEtiqueta := entrada.Atributos["area_etiqueta"]
	semantica := entrada.Atributos["semantica"]
	suscribible := entrada.Atributos["suscribible"]
	if !patronClaveKebab.MatchString(entrada.Clave) || !patronAtributoAPI.MatchString(area) ||
		!textoPublicoValido(areaEtiqueta, 120) || !patronAtributoAPI.MatchString(semantica) ||
		(suscribible != "si" && suscribible != "no") {
		return puertosbolsa.CategoriaPublica{}, puertosbolsa.ErrCatalogoCategoriasNoDisponible
	}
	return puertosbolsa.CategoriaPublica{
		Clave:        entrada.Clave,
		Version:      version,
		Etiqueta:     entrada.Etiqueta,
		Descripcion:  entrada.Descripcion,
		Semantica:    semantica,
		Orden:        entrada.Orden,
		Area:         area,
		AreaEtiqueta: areaEtiqueta,
		Suscribible:  suscribible == "si",
	}, nil
}
