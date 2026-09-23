// Package ports define contratos neutros de Dietas.
package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrAccesoBorradorDenegado    = errors.New("dietas: acceso al borrador denegado")
	ErrRelacionAmbigua           = errors.New("dietas: relacion de servicio ambigua")
	ErrRelacionNoDisponible      = errors.New("dietas: relacion de servicio no disponible")
	ErrRelacionNoValida          = errors.New("dietas: relacion de servicio no valida")
	ErrComisionNoEncontrada      = errors.New("dietas: comision no encontrada")
	ErrConflictoIdempotencia     = errors.New("dietas: conflicto de idempotencia")
	ErrResultadoBorradorIncierto = errors.New("dietas: resultado de borrador incierto")
	ErrBorradorNoDisponible      = errors.New("dietas: borrador no disponible")
)

type RelacionServicioAcreditada struct {
	RelacionRef        string
	PersonaRef         string
	EmpleadoRef        string
	UnidadRef          string
	VigenteDesde       string
	VigenteHasta       string
	Version            int64
	ProcedenciaActoRef string
	FuenteRef          string
	FuenteVersion      int64
}

type RevalidacionRelacionPersonal struct {
	RelacionRef, PersonaRef, EmpleadoRef, UnidadRef string
	VigenteDesde, VigenteHasta                      string
	FechaReferencia                                 string
	Version                                         int64
	ProcedenciaActoRef, FuenteRef                   string
	FuenteVersion                                   int64
}

type AutorizacionBorradorDurable struct {
	Material     vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Accion       string
	RecursoRef   string
	Finalidad    string
	Revalidacion RevalidacionRelacionPersonal
}

type IdentidadEfectivaBorrador struct {
	Vinculo            vecdomain.VinculoAutenticacionActorV2
	ContextoRegistrado vecdomain.ResultadoContextoActorRegistradoV2
	Relacion           RelacionServicioAcreditada
	Autorizacion       AutorizacionBorradorDurable
}

type OperacionBorrador string

const (
	OperacionCrearBorrador     OperacionBorrador = "crear_borrador_propio"
	OperacionConsultarBorrador OperacionBorrador = "consultar_borrador_propio"
)

type SolicitudOperacionBorrador struct {
	Operacion   OperacionBorrador
	Crear       SolicitudCrearBorradorPropio
	Consulta    ConsultaBorradoresPropios
	Referencia  string
	RelacionRef string
}

type SolicitudCrearBorradorPropio struct {
	ClaveIdempotencia string
	FechaInicio       string
	FechaFin          string
	Motivo            string
	CodigosRuta       []string
	HoraInicio        string
	HoraFin           string
	Calculo           *domain.CalculoComision
	RelacionRef       string
}

type ReciboBorradorComision struct {
	Referencia   string    `json:"referencia"`
	Version      uint64    `json:"version"`
	RegistradoEn time.Time `json:"registrado_en"`
	Repeticion   bool      `json:"repeticion"`
}
type ResultadoBorradorComision struct {
	Comision domain.ComisionBorrador `json:"comision"`
	Recibo   ReciboBorradorComision  `json:"recibo"`
}
type ConsultaBorradoresPropios struct {
	Limite int
	Cursor string
}
type PaginaBorradoresPropios struct {
	Items           []ResultadoBorradorComision
	SiguienteCursor string
}

type ResolutorIdentidadEfectivaBorrador interface {
	ResolverIdentidadEfectivaBorrador(context.Context, SolicitudOperacionBorrador) (IdentidadEfectivaBorrador, error)
}
type RepositorioBorradorComision interface {
	CrearORecuperar(context.Context, IdentidadEfectivaBorrador, SolicitudCrearBorradorPropio) (ResultadoBorradorComision, error)
	ObtenerPropio(context.Context, IdentidadEfectivaBorrador, string) (ResultadoBorradorComision, error)
	ListarPropios(context.Context, IdentidadEfectivaBorrador, ConsultaBorradoresPropios) (PaginaBorradoresPropios, error)
}
