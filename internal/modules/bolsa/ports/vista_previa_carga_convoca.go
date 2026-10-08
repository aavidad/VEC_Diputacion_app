package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaVistaPreviaCargaConvocaV1 = "vec.bolsa.rrhh.carga_convoca.vista_previa.v1"

var ErrVistaPreviaCargaConvocaInvalida = errors.New("bolsa: vista previa CONVOCA invalida")
var ErrVistaPreviaCargaConvocaNoDisponible = errors.New("bolsa: vista previa CONVOCA no disponible")

type PaginaVistaPreviaCargaConvoca struct {
	Filtro         string
	Limite         int
	Desplazamiento int
}

// El navegador sólo aporta fichero, clave de categoría y página; identidad,
// motivo y correlación proceden de la frontera corporativa verificada.
type SolicitudVistaPreviaCargaConvoca struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	CategoriaRef       string
	NombreFichero      string
	Contenido          []byte
	Pagina             PaginaVistaPreviaCargaConvoca
}

// La orden sólo contiene referencias y el contexto de recurso que V3 selló.
// Ninguna fila del Excel cruza la frontera PostgreSQL de la vista previa.
type OrdenVistaPreviaCargaConvoca struct {
	ActaRef                 string
	ActorRef                string
	CategoriaRef            string
	HuellaFicheroSHA256     string
	ContextoRecursoCanonico []byte
}

type AcuseVistaPreviaCargaConvoca struct {
	DecisionRef          string
	ActaRef              string
	HuellaContextoSHA256 string
	AuditoriaRef         string
	ConsumidaEn          time.Time
}

// El adaptador PostgreSQL consume la decisión AD218 y devuelve el acuse
// confirmado en una transacción SERIALIZABLE; nunca recibe filas ni fichero.
type ConsumidorVistaPreviaCargaConvoca interface {
	ConsumirVistaPreviaCargaConvoca(context.Context, OrdenVistaPreviaCargaConvoca,
		puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (AcuseVistaPreviaCargaConvoca, error)
}
