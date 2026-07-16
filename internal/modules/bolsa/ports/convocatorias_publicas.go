package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var (
	ErrConsultaConvocatoriasInvalida  = errors.New("bolsa: consulta publica de convocatorias invalida")
	ErrConsultaCategoriasInvalida     = errors.New("bolsa: consulta publica de categorias invalida")
	ErrConvocatoriaNoEncontrada       = errors.New("bolsa: convocatoria publica no encontrada")
	ErrFuenteConvocatoriasInvalida    = errors.New("bolsa: fuente publica de convocatorias invalida")
	ErrCatalogoCategoriasNoDisponible = errors.New("bolsa: catalogo publico de categorias no disponible")
)

const (
	CatalogoTiposConvocatoria      = "tipos_convocatoria"
	CatalogoEstadosConvocatoria    = "estados_convocatoria"
	CatalogoCategoriasConvocatoria = "categorias_convocatoria"
	CatalogoTiposPlazo             = "tipos_plazo"
	CatalogoTiposDocumento         = "tipos_documento"
	CatalogoCategoriasAyuda        = "categorias_ayuda"
)

// FiltroConvocatoriasPublicas contiene únicamente filtros públicos. Instante
// es fijado por el servidor para que un cliente no pueda decidir qué plazo se
// considera vigente.
type FiltroConvocatoriasPublicas struct {
	Texto            string
	Tipo             string
	Categoria        string
	Estado           string
	SoloPlazoAbierto bool
	Instante         time.Time
	Limite           int
	Desplazamiento   int
}

// EntradaCatalogoPublico es configuración gobernada y versionada. Añadir un
// valor no exige recompilar el núcleo ni la interfaz.
type EntradaCatalogoPublico struct {
	Clave       string `json:"clave"`
	Version     int    `json:"version"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion,omitempty"`
	Semantica   string `json:"semantica"`
	Orden       int    `json:"orden"`
	Publicable  bool   `json:"publicable"`
}

type CatalogoPublico struct {
	Referencia string                   `json:"referencia"`
	Version    int                      `json:"version"`
	Entradas   []EntradaCatalogoPublico `json:"entradas"`
}

// CategoriaPublica es la proyeccion minimizada de una entrada del catalogo
// gobernado del nucleo. Los metadatos de procedencia, gobierno y aprobacion no
// forman parte de este contrato publico.
type CategoriaPublica struct {
	Clave        string
	Version      int
	Etiqueta     string
	Descripcion  string
	Semantica    string
	Orden        int
	Area         string
	AreaEtiqueta string
	Suscribible  bool
}

type MetadatosFuenteCategorias struct {
	Revision      string
	ActualizadaEn time.Time
	Demostracion  bool
	Aviso         string
}

// CatalogoCategoriasPublicas conserva la identidad y version exactas que
// resolvio el adaptador. Ningun consumidor selecciona implicitamente la ultima
// version disponible.
type CatalogoCategoriasPublicas struct {
	ID           string
	Version      int
	HuellaSHA256 string
	Fuente       MetadatosFuenteCategorias
	Categorias   []CategoriaPublica
}

// ConsultaCategoriasPublicas separa el catalogo profesional de la fuente de
// convocatorias. Su adaptador debe fijar ID y version al construirse y devolver
// solo entradas publicadas, vigentes y publicables para el instante indicado.
type ConsultaCategoriasPublicas interface {
	ObtenerPublicadas(context.Context, time.Time) (CatalogoCategoriasPublicas, error)
}

type ConteoCategoriaConvocatorias struct {
	NumeroConvocatorias  int
	NumeroPlazosAbiertos int
}

type MetadatosFuenteConvocatorias struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso,omitempty"`
}

type PaginaConvocatorias struct {
	Convocatorias []dominiobolsa.Convocatoria
	Total         int
	Catalogos     []CatalogoPublico
	// ConteosCategorias aplica todos los filtros salvo Categoria. Permite
	// construir facetas navegables sin que la opcion seleccionada oculte las
	// restantes y sin convertir la fuente en autoridad del catalogo.
	ConteosCategorias map[string]ConteoCategoriaConvocatorias
	Fuente            MetadatosFuenteConvocatorias
}

type DetalleConvocatoria struct {
	Convocatoria dominiobolsa.Convocatoria
	Catalogos    []CatalogoPublico
	Fuente       MetadatosFuenteConvocatorias
}

// ConsultaConvocatoriasPublicas es el puerto autoritativo de lectura del mismo
// agregado Convocatoria que consumen solicitudes y baremación. Un adaptador
// PostgreSQL u Oracle puede sustituir al fichero local sin alterar este contrato.
type ConsultaConvocatoriasPublicas interface {
	BuscarPublicadas(context.Context, FiltroConvocatoriasPublicas) (PaginaConvocatorias, error)
	ObtenerPublicada(context.Context, string) (DetalleConvocatoria, error)
}
