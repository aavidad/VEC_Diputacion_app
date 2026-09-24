package ports

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrTeletrabajoNoAutorizado        = errors.New("cronos teletrabajo no autorizado en este periodo")
	ErrContinuidadMarcajeNoConfirmada = errors.New("cronos continuidad del marcaje remoto no confirmada")
	ErrMovimientoRemotoNoPermitido    = errors.New("cronos movimiento remoto no permitido por secuencia")
	ErrMarcajeRemotoNoEncontrado      = errors.New("cronos marcaje remoto no encontrado")
)

// PeriodoTeletrabajo sólo se expone para la persona propia con consulta autorizada.
// HastaUTC es exclusivo. No incluye referencias de empleado ni motivos privados.
type PeriodoTeletrabajo struct {
	DesdeUTC time.Time `json:"desde"`
	HastaUTC time.Time `json:"hasta"`
}

type DisponibilidadMarcajeRemoto struct {
	Autorizado            bool                `json:"autorizado"`
	ContinuidadConfirmada bool                `json:"continuidad_confirmada"`
	MovimientosPermitidos []domain.PunchKind  `json:"movimientos_permitidos"`
	Periodo               *PeriodoTeletrabajo `json:"periodo,omitempty"`
	Motivo                string              `json:"motivo"`
}

// ConsultaAutorizacionTeletrabajo debe exigir concesión positiva de lectura
// propia y devolver sólo la vigencia autorizada en el instante indicado.
// Un fallo del PDP o de la fuente devuelve error; nunca equivale a false.
type ConsultaAutorizacionTeletrabajo interface {
	ConsultarTeletrabajoPropio(context.Context, vecdomain.ContextoActor, string, time.Time) (domain.PeriodoTeletrabajo, bool, error)
}

// RepositorioMarcajesRemotos revalida teletrabajo autorizado por empleado e
// instante y continuidad bajo la misma transacción que consume V3, registra el
// marcaje, el recibo, la auditoría y el outbox. La consulta de continuidad
// concilia operación incierta y último marcaje bajo lock por empleado/periodo;
// claveOperacion vacía significa disponibilidad y una clave no vacía significa
// intento de escritura. Un TTL o la hora del navegador no son confirmación.
// Recuperar exige V3 propia de LECTURA y auditoría. Sólo puede devolver
// ErrMarcajeRemotoNoEncontrado tras tomar el mismo lock clave+empleado usado
// por POST, esperar toda transacción anterior y confirmar ausencia definitiva.
// Ante resultado incierto devuelve ErrDependenciaNoDisponible, nunca 404 débil.
// Si esa atomicidad no está disponible, debe fallar cerrado.
type RepositorioMarcajesRemotos interface {
	ConfirmarContinuidadMarcajeRemoto(context.Context, vecdomain.ContextoActor, string, domain.PeriodoTeletrabajo, string) (EstadoSecuenciaMarcajeRemoto, error)
	RegistrarOriginalRemotoAutorizado(context.Context, domain.MarcajeOriginal, domain.MaterialAutorizacionMarcajePropio, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboMarcajePropio, error)
	RecuperarOriginalRemotoAutorizado(context.Context, domain.MaterialRecuperacionMarcajeRemoto, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboMarcajePropio, error)
}

// EstadoSecuenciaMarcajeRemoto sólo procede de un repositorio durable nominal.
// Continuidad y movimientos se calculan bajo lock del empleado y periodo.
type EstadoSecuenciaMarcajeRemoto struct {
	ContinuidadConfirmada bool
	MovimientosPermitidos []domain.PunchKind
}

type CasoUsoMarcajesRemotos interface {
	ConsultarDisponibilidadMarcajeRemoto(context.Context, ContextoMarcajePropio) (DisponibilidadMarcajeRemoto, error)
	RegistrarMarcajeRemoto(context.Context, ContextoMarcajePropio, SolicitudMarcajePropio) (ReciboMarcajePropio, error)
	RecuperarReciboMarcajeRemoto(context.Context, ContextoRecuperacionMarcajeRemoto, SolicitudMarcajePropio) (ReciboMarcajePropio, error)
}

// ProveedorMaterialRecuperacionMarcajeRemoto usa una acción V3 de LECTURA
// propia, independiente de la concesión para registrar un nuevo marcaje.
type ProveedorMaterialRecuperacionMarcajeRemoto interface {
	ProveerMaterialRecuperacionMarcajeRemoto(context.Context, domain.MaterialRecuperacionMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenLecturaMarcajeRemoto struct {
	actor     vecdomain.ContextoActor
	proveedor ProveedorMaterialRecuperacionMarcajeRemoto
}

func NuevaOrdenLecturaMarcajeRemoto(actor vecdomain.ContextoActor, proveedor ProveedorMaterialRecuperacionMarcajeRemoto) (OrdenLecturaMarcajeRemoto, error) {
	if actor.Validar() != nil || dependenciaRemotaNula(proveedor) {
		return OrdenLecturaMarcajeRemoto{}, ErrDependenciaNoDisponible
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenLecturaMarcajeRemoto{}, ErrDependenciaNoDisponible
	}
	return OrdenLecturaMarcajeRemoto{actor: copia, proveedor: proveedor}, nil
}

func (o OrdenLecturaMarcajeRemoto) ContextoActor() (vecdomain.ContextoActor, error) {
	if dependenciaRemotaNula(o.proveedor) || o.actor.Validar() != nil {
		return vecdomain.ContextoActor{}, ErrDependenciaNoDisponible
	}
	return o.actor.Clonar()
}

func (o OrdenLecturaMarcajeRemoto) ProveedorMaterial() ProveedorMaterialRecuperacionMarcajeRemoto {
	if dependenciaRemotaNula(o.proveedor) {
		return nil
	}
	return o.proveedor
}

func dependenciaRemotaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

type ContextoRecuperacionMarcajeRemoto struct {
	CanalAcreditado domain.AcreditacionCanalMarcaje
	OrdenLectura    OrdenLecturaMarcajeRemoto
}
