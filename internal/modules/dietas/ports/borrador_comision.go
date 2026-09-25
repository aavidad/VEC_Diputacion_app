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
	ErrVersionComisionConflicto  = errors.New("dietas: version de comision en conflicto")
	ErrDocumentoNoDisponible     = errors.New("dietas: documento no disponible")
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
	OperacionCrearBorrador      OperacionBorrador = "crear_borrador_propio"
	OperacionConsultarBorrador  OperacionBorrador = "consultar_borrador_propio"
	OperacionEditarBorrador     OperacionBorrador = "editar_borrador_propio"
	OperacionBorrarBorrador     OperacionBorrador = "borrar_borrador_propio"
	OperacionEnviarBorrador     OperacionBorrador = "enviar_borrador_propio"
	OperacionConsultarDocumento OperacionBorrador = "consultar_documento_propio"
)

type SolicitudOperacionBorrador struct {
	Operacion   OperacionBorrador
	Crear       SolicitudCrearBorradorPropio
	Consulta    ConsultaBorradoresPropios
	Referencia  string
	RelacionRef string
	Editar      SolicitudEditarComisionPropia
	Mutacion    SolicitudMutacionComisionPropia
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

type SolicitudEditarComisionPropia struct {
	Referencia            string
	ClaveIdempotencia     string
	VersionEsperada       uint64
	RelacionRef           string
	FechaInicio           string
	FechaFin              string
	HoraInicio            string
	HoraFin               string
	Motivo                string
	CodigosRuta           []string
	VehiculoPropio        bool
	Rutas                 []domain.RutaDeclaradaComision
	TramosAceptados       []int
	VersionTarifaAceptada string
	Asignacion            *AsignacionDietasAcreditada
	Otros                 []domain.OtroGastoDeclarado
	Calculo               *domain.CalculoComision
	Documento             *domain.DocumentoComision
}

type SolicitudMutacionComisionPropia struct {
	Referencia        string
	ClaveIdempotencia string
	VersionEsperada   uint64
	RelacionRef       string
	Asignacion        *AsignacionDietasAcreditada
}

type AsignacionDietasAcreditada struct {
	AsignacionRef            string    `json:"asignacion_ref"`
	RelacionRef              string    `json:"relacion_ref"`
	PersonaRef               string    `json:"persona_ref"`
	UnidadRef                string    `json:"unidad_ref"`
	CentroRef                string    `json:"centro_ref"`
	AdministrativoPersonaRef string    `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string    `json:"responsable_persona_ref"`
	GrupoDieta               int16     `json:"grupo_dieta"`
	VigenteDesde             string    `json:"vigente_desde"`
	Version                  int64     `json:"version"`
	ReciboRef                string    `json:"recibo_ref"`
	DecisionRef              string    `json:"decision_ref"`
	EfectoRef                string    `json:"efecto_ref"`
	ConsumoHuellaSHA256      string    `json:"consumo_huella_sha256"`
	AuditoriaRef             string    `json:"auditoria_ref"`
	RegistradaEn             time.Time `json:"registrada_en"`
}

type ProveedorAsignacionParaEnvio interface {
	ConsultarAsignacionParaEnvio(context.Context, IdentidadEfectivaBorrador) (AsignacionDietasAcreditada, error)
}

type ReciboBorradorComision struct {
	Referencia        string    `json:"referencia"`
	Version           uint64    `json:"version"`
	RegistradoEn      time.Time `json:"registrado_en"`
	Repeticion        bool      `json:"repeticion"`
	ReglaRef          string    `json:"regla_ref,omitempty"`
	ReglaHuellaSHA256 string    `json:"regla_huella_sha256,omitempty"`
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

// La capacidad v2 queda separada para no convertir los adaptadores v1 ya
// instalados en implementaciones parciales de operaciones nuevas.
type RepositorioMutacionComision interface {
	EditarPropio(context.Context, IdentidadEfectivaBorrador, SolicitudEditarComisionPropia) (ResultadoBorradorComision, error)
	BorrarPropio(context.Context, IdentidadEfectivaBorrador, SolicitudMutacionComisionPropia) (ResultadoBorradorComision, error)
	EnviarPropio(context.Context, IdentidadEfectivaBorrador, SolicitudMutacionComisionPropia) (ResultadoBorradorComision, error)
}

type RecuperadorEdicionComision interface {
	RecuperarEdicionPorClave(context.Context, IdentidadEfectivaBorrador, SolicitudEditarComisionPropia) (ResultadoBorradorComision, bool, error)
}

type RepositorioConsultaDocumento interface {
	ObtenerDocumentoPropio(context.Context, IdentidadEfectivaBorrador, string) (ResultadoBorradorComision, error)
	ListarDocumentosPropios(context.Context, IdentidadEfectivaBorrador, ConsultaBorradoresPropios) (PaginaBorradoresPropios, error)
}
