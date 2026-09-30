package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
)

// SeleccionHechosIncorporacionCT procede de la preparación confiable de CT.
// Identifica hechos exactos; no concede acceso ni acredita una reserva RPT.
type SeleccionHechosIncorporacionCT struct {
	OrganismoRef, PersonaRef, EmpleadoRef              string
	RelacionRef, OcupacionRef                          string
	UnidadRef, PuestoRef, PlazaRef                     string
	VersionEmpleado, VersionRelacion, VersionOcupacion int64
	Corte                                              domain.CorteEmpleadoB2
}

type SolicitudHechosIncorporacionCT struct {
	Seleccion SeleccionHechosIncorporacionCT
	Actor     core.ContextoActor
}

// HechosIncorporacionCT proyecta la relación y ocupación registradas en B2.
// Las fechas conservan [Desde,Hasta), incluido el fin abierto. La evidencia
// corresponde a la lectura nominal; no es un recibo de alta ni de CT75.
type HechosIncorporacionCT struct {
	Esquema                                  string
	Seleccion                                SeleccionHechosIncorporacionCT
	Relacion, Ocupacion                      domain.TrazaEmpleadoB2
	RegimenRef, ModalidadRef, ClaseOcupacion string
	FirmaOficial, EficaciaAdministrativa     bool
	Evidencia                                EvidenciaRegistroEmpleadoB2
}

type ConsultaHechosIncorporacionCT interface {
	ConsultarHechosIncorporacionCT(context.Context, SolicitudHechosIncorporacionCT) (HechosIncorporacionCT, error)
}

// FuenteFichaIncorporacionCT mantiene la lectura propietaria nominal de B2.
// Cada llamada debe autorizarse y auditarse, también al recuperar un plan.
type FuenteFichaIncorporacionCT interface {
	ConsultarFicha(context.Context, domain.SolicitudFichaEmpleadoB2) (ResultadoFichaEmpleadoB2, error)
}
