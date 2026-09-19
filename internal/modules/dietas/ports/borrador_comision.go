package ports

import (
	"context"
	"errors"
	"time"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Límites por operación. SQL impone además 192 KiB al JSONB persistido
// antes de aceptar la creación; cubre expansión decimal y espacios de JSONB.
// El listado devuelve hasta 20 resúmenes sin coordenadas, tramos ni trazado.
const (
	MaxBytesEntradaBorrador  = 64 << 10
	MaxBytesMaterialBorrador = 96 << 10
	MaxBytesReciboBorrador   = 4 << 10
	MaxBytesDetalleBorrador  = 256 << 10
	MaxBytesListadoBorrador  = 256 << 10
)

var (
	ErrAccesoBorradorDenegado       = errors.New("dietas: acceso al borrador denegado")
	ErrConflictoBorrador            = errors.New("dietas: conflicto de version o idempotencia")
	ErrBorradorNoEncontrado         = errors.New("dietas: borrador no encontrado")
	ErrBorradorNoDisponible         = errors.New("dietas: borrador no disponible")
	ErrResultadoBorradorIncierto    = errors.New("dietas: resultado del borrador indeterminado")
	ErrPoliticaBorradorNoDisponible = errors.New("dietas: politica del borrador no disponible")
)

type SolicitudCrearBorradorPropio struct {
	ContextoActor      vecdomain.ContextoActor
	ClaveOperacion     string
	VersionEsperada    uint64
	Borrador           domain.BorradorComision
	PoliticaReferencia string
	PoliticaVersion    string
}

type SolicitudPoliticaKilometraje struct {
	ContextoActor  vecdomain.ContextoActor
	Referencia     string
	Version        string
	Inicio         time.Time
	VehiculoPropio bool
}

// El proveedor gobierna vigencia y versión. Una selección del cliente nunca
// aporta una tarifa ni constituye autoridad para su uso.
type ProveedorPoliticaKilometrajeBorrador interface {
	ResolverPoliticaKilometraje(context.Context, SolicitudPoliticaKilometraje) (domain.PoliticaKilometraje, error)
}

type ConsultaBorradoresPropios struct {
	Limite  int
	Despues string
}

// BorradorConRecibo transporta sólo el resumen sin geometría en listados.
type BorradorConRecibo struct {
	Borrador domain.BorradorComision `json:"borrador"`
	Recibo   ReciboBorradorComision  `json:"recibo"`
}
type PaginaBorradoresPropios struct {
	Borradores []BorradorConRecibo `json:"borradores"`
	Siguiente  string              `json:"siguiente"`
}

type ReciboBorradorComision struct {
	ComisionRef    string    `json:"comision_ref"`
	ReciboRef      string    `json:"recibo_ref"`
	CorrelacionRef string    `json:"correlacion_ref"`
	Version        uint64    `json:"version"`
	Repeticion     bool      `json:"repeticion"`
	RegistradoEn   time.Time `json:"registrado_en"`
}

// UnidadTrabajoBorradorComision es la frontera durable del efecto. La emisión
// V3 puede ocurrir previamente; su consumo y la escritura de estado, historia,
// recibo, auditoría y outbox comparten transacción. Cada lectura y replay exige
// autorización nueva. No acepta permiso ni material de autorización de HTTP.
type UnidadTrabajoBorradorComision interface {
	CrearBorradorPropio(context.Context, SolicitudCrearBorradorPropio) (ReciboBorradorComision, error)
	RecuperarBorradorPropio(context.Context, vecdomain.ContextoActor, string) (domain.BorradorComision, ReciboBorradorComision, error)
	ListarBorradoresPropios(context.Context, vecdomain.ContextoActor, ConsultaBorradoresPropios) (PaginaBorradoresPropios, error)
}
