package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var (
	ErrConsultaConvocatoriasInvalida = errors.New("bolsa: consulta publica de convocatorias invalida")
	ErrConvocatoriaNoEncontrada      = errors.New("bolsa: convocatoria publica no encontrada")
	ErrFuenteConvocatoriasInvalida   = errors.New("bolsa: fuente publica de convocatorias invalida")
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
	Fuente        MetadatosFuenteConvocatorias
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
