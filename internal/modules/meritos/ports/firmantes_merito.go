package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
)

var (
	ErrPlanFirmantesNoDisponible = errors.New("meritos.error.plan_firmantes_no_disponible")
	ErrPlanFirmantesConflicto    = errors.New("meritos.error.plan_firmantes_conflicto")
)

const (
	AccionPrepararPlanFirmantes  = "meritos.firmantes.plan.preparar"
	AccionCambiarPlanFirmantes   = "meritos.firmantes.plan.cambiar_contenido"
	AccionSustituirPlanFirmantes = "meritos.firmantes.plan.sustituir"
)

// SolicitudAutoridadFirmantes incluye una huella canónica de TODO lo pedido,
// calculada en aplicación. El adaptador obtiene actor/perfil del contexto
// acreditado, exige permiso positivo RRHH y revalida antes del efecto.
type SolicitudAutoridadFirmantes struct {
	Accion                string
	HechoRef              string
	DecisionRef           string
	VersionRosterEsperada uint64
	HuellaSolicitudSHA256 string
}

type AutoridadFirmantes struct {
	Accion                string
	ActorRef              string
	PerfilRef             string
	HechoRef              string
	DecisionRef           string
	VersionRosterEsperada uint64
	HuellaSolicitudSHA256 string
	PermisoRef            string
	VigenteHasta          time.Time
}

type AutorizadorFirmantesMerito interface {
	AutorizarPreparacionFirmantes(context.Context, SolicitudAutoridadFirmantes) (AutoridadFirmantes, error)
}

// FuenteDecisionFirmantes coteja el hecho RUM y la versión de decisión que
// edita RRHH, incluido motivo, puntos y cada evidencia exacta. Nunca toma
// el material enviado por HTTP como fuente autoritativa.
type FuenteDecisionFirmantes interface {
	ObtenerMaterialExactoAutorizado(context.Context, AutoridadFirmantes, domain.MaterialDecisionFirmantes) (domain.MaterialDecisionFirmantes, error)
}

// ComprobacionDesignaciones solo puede emitirla la autoridad de nombramientos
// y bases tras cotejar cada persona/rol/órgano/versión/fecha y el efecto
// permitido para la convocatoria. El catálogo de roles nunca concede acceso.
type ComprobacionDesignaciones struct {
	HuellaRosterSHA256 string
	FuenteRef          string
	FuenteVersion      uint64
	CompetenciaRef     string
	VigenteHasta       time.Time
}

type FuenteDesignacionesFirmantes interface {
	ComprobarDesignacionesExactas(context.Context, domain.PlanFirmantesMerito) (ComprobacionDesignaciones, error)
}

// RepositorioPlanFirmantes debe exigir CAS e idempotencia en la misma
// transacción que historia, auditoría y recibo. GuardarVersion vuelve a
// comprobar actor+clave+huella antes del CAS para cerrar la carrera entre
// RecuperarConfirmadaAutorizada y la escritura. Una recuperación equivalente
// devuelve el plan/recibo originales y no añade firmas ni otra versión.
// No existe adaptador productivo en este corte; inyectar nil falla cerrado.
type RepositorioPlanFirmantes interface {
	// RecuperarConfirmadaAutorizada revalida/consume una autoridad fresca antes
	// de leer por actor+clave y por hecho/decisión exactos. Devuelve la preimagen original ligada al recibo;
	// una clave usada con otra huella nunca se trata como ausencia.
	RecuperarConfirmadaAutorizada(context.Context, AutoridadFirmantes, string) (RecuperacionPlanFirmantes, bool, error)
	ObtenerActualAutorizado(context.Context, AutoridadFirmantes, string) (domain.PlanFirmantesMerito, error)
	GuardarVersion(context.Context, OrdenGuardarPlanFirmantes) (ResultadoGuardarPlanFirmantes, error)
}

type RecuperacionPlanFirmantes struct {
	Accion                string
	ActorRef              string
	ClaveIdempotencia     string
	HuellaSolicitudSHA256 string
	VersionRosterEsperada uint64
	Resultado             ResultadoGuardarPlanFirmantes
}

type OrdenGuardarPlanFirmantes struct {
	Plan                  domain.PlanFirmantesMerito
	VersionRosterEsperada uint64
	HuellaAnteriorSHA256  string
	ClaveIdempotencia     string
	Autoridad             AutoridadFirmantes
	Designaciones         ComprobacionDesignaciones
}

type ResultadoGuardarPlanFirmantes struct {
	Plan      domain.PlanFirmantesMerito
	ReciboRef string
	Repetida  bool
}
