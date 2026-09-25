package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	// ErrContratosParticipacionNoDisponible: ni el inbox ni la consulta
	// interpretan un fallo como ausencia de contratos.
	ErrContratosParticipacionNoDisponible = errors.New("bolsa: historico de contratos no disponible")
	// ErrEventoContratoDivergente: el mismo evento llegó con otro contenido.
	ErrEventoContratoDivergente = errors.New("bolsa: evento de contrato reentregado con otro contenido")
)

// ContratoParticipacion es una fila del histórico B13 de la ficha RRHH.
type ContratoParticipacion struct {
	EventoRef      string
	Tipo           string
	Inicio         *time.Time
	FinPrevisto    *time.Time
	ModalidadClave string
	CategoriaRef   string
	CausaClave     string
	ExpedienteRef  string
	LlamamientoRef string
	OcurridoEn     time.Time
}

// CursorContratosParticipacion es el último origen CT recibido por el inbox,
// por posición de publicación (la transacción CT que lo escribió).
type CursorContratosParticipacion struct {
	Posicion  int64
	OrigenRef string
}

// EventoContratoRecibido conserva el contenido exacto y su huella para que
// el inbox pueda distinguir una reentrega de un contenido divergente.
type EventoContratoRecibido struct {
	Evento         dominiobolsa.EventoContratoParticipacion
	Contenido      []byte
	HuellaSHA256   string
	OrigenCreadaEn time.Time
	OrigenPosicion int64
}

// ResultadoRegistroContrato: ParticipacionRef vacía si Bolsa no reconoce
// el llamamiento; el evento queda conservado igualmente.
type ResultadoRegistroContrato struct {
	Reutilizado      bool
	ParticipacionRef string
}

// BuzonContratosParticipacion es el inbox idempotente B13.
type BuzonContratosParticipacion interface {
	CursorContratos(context.Context) (CursorContratosParticipacion, bool, error)
	RegistrarContrato(context.Context, EventoContratoRecibido) (ResultadoRegistroContrato, error)
}

// RepositorioContratosParticipacion lee el histórico con el mismo material
// V3 que consume el historial de operaciones de la participación.
type RepositorioContratosParticipacion interface {
	ListarContratos(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ContratoParticipacion, error)
}
