package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/provision/domain"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

type PeticionProceso struct {
	Proceso   domain.ProcesoProvision   `json:"proceso"`
	Solicitud domain.SolicitudProvision `json:"solicitud"`
}

type SimuladorProceso interface {
	SimularProceso(PeticionProceso) (domain.ResultadoProceso, error)
}

// Puertos reservados para integración futura: este corte no los conecta al
// simulador ni convierte referencias enviadas por el cliente en autorizaciones.
type ConsultaEmpleadoProceso struct {
	EmpleadoRef           string
	FechaCorte            b.FechaCivil
	ContextoAutorizadoRef string
}

type FuentePersonalProceso interface {
	ObtenerInstantanea(context.Context, ConsultaEmpleadoProceso) (domain.InstantaneaEmpleado, error)
}

type ConsultaRPTProceso struct {
	PuestoRef             string
	Version               string
	ContextoAutorizadoRef string
}

// La fuente falla si no puede devolver esa versión exacta; nunca sustituye
// una caída por un puesto vacío ni presupone que existe una vacante oficial.
type FuenteRPTProceso interface {
	ObtenerPuesto(context.Context, ConsultaRPTProceso) (domain.PuestoOfertado, error)
}

type ConsultaAutorizacionProceso struct {
	ActorContextoRef string
	Accion           string
	RecursoRef       string
	AmbitoRef        string
	Finalidad        string
	Campos           []string
	Obligaciones     []string
	VersionEsperada  string
}

// El adaptador futuro usa la autoridad central de VEC, sin otro PDP local.
type AutorizadorProceso interface {
	Autorizar(context.Context, ConsultaAutorizacionProceso) (ConcesionProceso, error)
}

type ConcesionProceso struct{ Referencia string }

type EscrituraBorradorProceso struct {
	Concesion       ConcesionProceso
	IdempotenciaRef string
	VersionEsperada string
	CorrelacionRef  string
	Proceso         domain.ProcesoProvision
	Solicitud       domain.SolicitudProvision
}

type ConfirmacionBorradorProceso struct {
	Version   string
	ReciboRef string
}

// Una implementación durable deberá revalidar y consumir la concesión central
// en la misma transacción que versión, idempotencia semántica, historia,
// auditoría y outbox. Este puerto carece de implementación en la simulación.
type RepositorioBorradorProceso interface {
	GuardarAutorizado(context.Context, EscrituraBorradorProceso) (ConfirmacionBorradorProceso, error)
}
