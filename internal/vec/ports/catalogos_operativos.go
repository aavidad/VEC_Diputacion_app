package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrOperacionCatalogoOperativoNoEncontrada = errors.New("vec: operacion de catalogo operativo no encontrada")
	ErrOperacionCatalogoOperativoEnConflicto  = errors.New("vec: operacion de catalogo operativo en conflicto")
	ErrReciboCatalogoOperativoInvalido        = errors.New("vec: recibo de catalogo operativo invalido")
)

// CabezaCatalogoOperativo identifica la publicación exacta sobre la que se
// prepara una nueva versión. El valor cero no significa una cabeza inicial.
type CabezaCatalogoOperativo struct {
	CatalogoID   string
	Version      int
	HuellaSHA256 string
	Estado       domain.EstadoCatalogoConfigurable
}

// ConsultaCabezaCatalogoOperativo resuelve una sola publicación en el
// presupuesto recibido. No lista versiones ni limita la historia a 64 filas.
// Una fuente ausente o una respuesta truncada nunca se interpreta como vacía.
type ConsultaCabezaCatalogoOperativo interface {
	ObtenerCabezaCatalogoOperativo(context.Context, string, LimitesConsultaCatalogosAcotada) (ResultadoConsultaCatalogoAcotado, error)
}

// ConfirmacionCatalogoOperativo procede de ServicioCatalogos. La clave
// semántica es independiente de la correlación de cada intento. El adaptador
// añade desde su configuración confiable la identidad de fuente y su versión;
// esos datos no se aceptan como autoridad del cliente. MaterialCanonico es
// una copia de los bytes semánticos exactos cuya SHA256 se declara; el adaptador
// conserva los bytes y coteja además su significado contra el efecto.
type ConfirmacionCatalogoOperativo struct {
	ClaveIdempotencia    string
	HuellaMaterialSHA256 string
	MaterialCanonico     []byte
	CabezaEsperada       CabezaCatalogoOperativo
	HuellaAnteriorSHA256 string
	Catalogo             domain.CatalogoConfigurable
	Auditoria            domain.AuditEntry
	Evento               domain.Event
	Autorizacion         EvidenciaUsoDecisionAutorizacion
}

// RecuperacionCatalogoOperativo exige una autorización nueva para la acción
// original y su recurso exacto. El repositorio coteja también actor original,
// clave y material antes de devolver cualquier recibo.
type RecuperacionCatalogoOperativo struct {
	ClaveIdempotencia    string
	HuellaMaterialSHA256 string
	MaterialCanonico     []byte
	CatalogoID           string
	Version              int
	Accion               string
	Autorizacion         EvidenciaUsoDecisionAutorizacion
}

type ReciboCatalogoOperativo struct {
	Referencia           string
	ClaveIdempotencia    string
	HuellaMaterialSHA256 string
	Accion               string
	CatalogoID           string
	Version              int
	HuellaSHA256         string
	Estado               domain.EstadoCatalogoConfigurable
	ActorRef             string
	AuditoriaRef         string
	OutboxRef            string
	ConfirmadoEn         time.Time
}

// Catalogo es la instantánea original del efecto, incluso en recuperación.
// Una publicación posterior no reescribe este resultado ni su recibo.
type ResultadoOperacionCatalogoOperativo struct {
	Catalogo   domain.CatalogoConfigurable
	Recibo     ReciboCatalogoOperativo
	Recuperada bool
}

// RepositorioCatalogosOperativos amplía el gobierno sin cambiar el contrato
// histórico de cuatro métodos. Cada confirmación revalida y consume autoridad
// en la transacción que fija versión, historia, auditoría, outbox y recibo.
// Primero recupera una clave existente con el mismo material y actor; después
// coteja la cabeza publicada y, al publicar, el borrador exacto. Sólo publicar
// cambia la cabeza operativa. Material distinto para una clave produce conflicto.
// La recuperación también revalida autoridad vigente y registra su acceso.
type RepositorioCatalogosOperativos interface {
	ConfirmarAltaBorradorCatalogoOperativo(context.Context, ConfirmacionCatalogoOperativo) (ResultadoOperacionCatalogoOperativo, error)
	ConfirmarPublicacionCatalogoOperativo(context.Context, ConfirmacionCatalogoOperativo) (ResultadoOperacionCatalogoOperativo, error)
	RecuperarOperacionCatalogoOperativo(context.Context, RecuperacionCatalogoOperativo) (ResultadoOperacionCatalogoOperativo, error)
}
