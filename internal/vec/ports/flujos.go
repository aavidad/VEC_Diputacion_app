package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrDefinicionFlujoNoEncontrada      = errors.New("vec: definicion de flujo no encontrada")
	ErrVersionDefinicionFlujoYaExiste   = errors.New("vec: version de definicion de flujo ya existe")
	ErrRevisionDefinicionFlujoConflicto = errors.New("vec: revision de definicion de flujo en conflicto")
	ErrSecuenciaDefinicionFlujoInvalida = errors.New("vec: secuencia de definicion de flujo invalida")
	ErrInstanciaFlujoNoEncontrada       = errors.New("vec: instancia de flujo no encontrada")
	ErrInstanciaFlujoYaExiste           = errors.New("vec: instancia de flujo ya existe")
	ErrEntidadConInstanciaFlujo         = errors.New("vec: la entidad ya tiene una instancia para esta definicion")
	ErrRevisionInstanciaFlujoConflicto  = errors.New("vec: revision de instancia de flujo en conflicto")
	ErrDecisionReglaFlujoNoEncontrada   = errors.New("vec: decision de regla de flujo no encontrada")
	ErrDecisionReglaFlujoYaExiste       = errors.New("vec: decision de regla de flujo ya existe")
	ErrEvaluadorReglaFlujoNoDisponible  = errors.New("vec: evaluador de regla de flujo no disponible")
	ErrAprobacionFlujoNoVerificada      = errors.New("vec: aprobacion de flujo no verificada")
)

type ConsultaDefinicionesFlujo interface {
	ObtenerDefinicionFlujo(context.Context, string, int) (domain.DefinicionFlujo, error)
	ObtenerDefinicionFlujoPorReferencia(context.Context, string) (domain.DefinicionFlujo, error)
	ListarVersionesDefinicionFlujo(context.Context, string) ([]domain.DefinicionFlujo, error)
}

// RepositorioGobiernoFlujos confirma cada mutacion con su auditoria y evento
// en una misma transaccion. Una version publicada o retirada es inmutable.
type RepositorioGobiernoFlujos interface {
	ConfirmarAltaBorradorFlujo(context.Context, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarActualizacionBorradorFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaFlujo(context.Context, string, domain.DefinicionFlujo, domain.AuditEntry, domain.Event) error
}

type ConsultaInstanciasFlujo interface {
	ObtenerInstanciaFlujo(context.Context, string) (domain.InstanciaFlujo, error)
}

// RepositorioInstanciasFlujo aplica control optimista y confirma estado,
// auditoria y outbox de forma atomica. La decision de regla debe estar
// registrada previamente y se vuelve a cotejar al confirmar la transicion.
type RepositorioInstanciasFlujo interface {
	ConfirmarInicioInstanciaFlujo(context.Context, domain.InstanciaFlujo, domain.AuditEntry, domain.Event) error
	ConfirmarTransicionInstanciaFlujo(
		context.Context,
		string,
		domain.InstanciaFlujo,
		domain.CambioEstadoFlujo,
		domain.DecisionReglaFlujo,
		*domain.EvidenciaAprobacionFlujo,
		domain.AuditEntry,
		domain.Event,
	) error
}

type SolicitudEvaluarReglaFlujo struct {
	Definicion     domain.DefinicionFlujo
	Instancia      domain.InstanciaFlujo
	Transicion     domain.TransicionFlujoConfigurable
	ActorID        string
	Finalidad      string
	Motivo         string
	CorrelacionRef string
}

// EvaluadorReglasFlujo resuelve ReglaRef y obtiene los hechos mediante
// referencias internas. La solicitud no transporta payloads personales para
// evitar que estos terminen accidentalmente en trazas o colas.
type EvaluadorReglasFlujo interface {
	EvaluarReglaFlujo(context.Context, SolicitudEvaluarReglaFlujo) (domain.DecisionReglaFlujo, error)
}

// RegistroDecisionesReglaFlujo es de solo adicion. Incluso una denegacion se
// registra antes de devolverse para conservar la explicabilidad del intento.
type RegistroDecisionesReglaFlujo interface {
	RegistrarDecisionReglaFlujo(context.Context, domain.DecisionReglaFlujo, domain.AuditEntry, domain.Event) error
	ObtenerDecisionReglaFlujo(context.Context, string) (domain.DecisionReglaFlujo, error)
}

type SolicitudVerificarAprobacionFlujo struct {
	ReferenciaAprobacion string
	SolicitanteID        string
	Definicion           domain.DefinicionFlujo
	Instancia            domain.InstanciaFlujo
	Transicion           domain.TransicionFlujoConfigurable
	DecisionRegla        domain.DecisionReglaFlujo
	Finalidad            string
	CorrelacionRef       string
}

// VerificadorAprobacionesFlujo impide considerar suficiente una referencia
// escrita por el cliente. El adaptador consulta el registro de aprobaciones y
// devuelve la evidencia exacta, que el caso de uso vuelve a validar.
type VerificadorAprobacionesFlujo interface {
	VerificarAprobacionFlujo(context.Context, SolicitudVerificarAprobacionFlujo) (domain.EvidenciaAprobacionFlujo, error)
}

type GeneradorIDInstanciaFlujo interface {
	NuevoIDInstanciaFlujo() (string, error)
}
