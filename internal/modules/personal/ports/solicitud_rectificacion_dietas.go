package ports

import (
	"context"
	"errors"
	"regexp"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrRectificacionDietasNoDisponible = errors.New("personal: rectificacion de dietas no disponible")
	ErrRectificacionDietasDenegada     = errors.New("personal: rectificacion de dietas denegada")
	ErrRectificacionDietasInvalida     = errors.New("personal: rectificacion de dietas invalida")
	ErrRectificacionDietasConflicto    = errors.New("personal: conflicto de rectificacion de dietas")
	ErrRectificacionDietasNoEncontrada = errors.New("personal: solicitud de rectificacion de dietas no encontrada")
)

const (
	AccionSolicitarRectificacionDietas    = "personal.asignacion_dietas.rectificacion.solicitar"
	AccionConsultarRectificacionDietas    = "personal.asignacion_dietas.rectificacion.propia.consultar"
	AccionResolverRectificacionDietas     = "personal.asignacion_dietas.rectificacion.resolver"
	AudienciaSolicitarRectificacionDietas = "vec_personal.asignacion_dietas.rectificacion.solicitar.v1"
	AudienciaConsultarRectificacionDietas = "vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1"
	AudienciaResolverRectificacionDietas  = "vec_personal.asignacion_dietas.rectificacion.resolver.v1"
)

type ProveedorAutorizacionRectificacionDietas interface {
	AutorizarRectificacionDietas(context.Context, personaldomain.MaterialRectificacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenRectificacionDietas struct {
	Material               personaldomain.MaterialRectificacionDietas
	Autorizacion           vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Correccion             personaldomain.MaterialAsignacionDietas
	AutorizacionCorreccion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoRectificacionDietas struct {
	SolicitudRef        string    `json:"solicitud_ref"`
	ReciboRef           string    `json:"recibo_ref"`
	Estado              string    `json:"estado"`
	RegistradaEn        time.Time `json:"registrada_en"`
	AsignacionRef       string    `json:"asignacion_ref"`
	VersionOrigen       int64     `json:"version_origen"`
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaAD3Ref     string    `json:"auditoria_ad3_ref"`
}

type RepositorioRectificacionDietas interface {
	EjecutarRectificacionDietas(context.Context, OrdenRectificacionDietas) (ResultadoRectificacionDietas, error)
}

const RutaSolicitudesRectificacionDietas = "/api/vec/personal/solicitudes-rectificacion-dietas"

type OrdenAuditoriaFronteraRectificacionDietas struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	Accion         string
	ActorRef       string
	RecursoRef     string
	EstadoHTTP     int
}

func (o OrdenAuditoriaFronteraRectificacionDietas) Validar() error {
	if o.CorrelacionRef != "corr_no_disponible" && !patronCorrelacionFronteraPersonal.MatchString(o.CorrelacionRef) {
		return ErrRectificacionDietasInvalida
	}
	if o.Motivo != MotivoFronteraPersonalAutenticacion && o.Motivo != MotivoFronteraPersonalDenegado && o.Motivo != MotivoFronteraPersonalDependencia {
		return ErrRectificacionDietasInvalida
	}
	if o.Ruta != RutaSolicitudesRectificacionDietas {
		return ErrRectificacionDietasInvalida
	}
	if o.Accion != "solicitar" && o.Accion != "consultar" && o.Accion != "consultar_competentes" && o.Accion != "confirmar" && o.Accion != "rechazar" && o.Accion != "metodo_no_admitido" {
		return ErrRectificacionDietasInvalida
	}
	if o.Motivo == MotivoFronteraPersonalAutenticacion && o.ActorRef != "" {
		return ErrRectificacionDietasInvalida
	}
	if o.ActorRef != "" && !patronActorFronteraPersonal.MatchString(o.ActorRef) {
		return ErrRectificacionDietasInvalida
	}
	if o.RecursoRef != "" && !ReferenciaFronteraRectificacionDietasValida(o.RecursoRef) {
		return ErrRectificacionDietasInvalida
	}
	if o.EstadoHTTP != 400 && o.EstadoHTTP != 401 && o.EstadoHTTP != 403 && o.EstadoHTTP != 404 && o.EstadoHTTP != 405 && o.EstadoHTTP != 406 && o.EstadoHTTP != 409 && o.EstadoHTTP != 503 {
		return ErrRectificacionDietasInvalida
	}
	if (o.Motivo == MotivoFronteraPersonalAutenticacion) != (o.EstadoHTTP == 401) ||
		(o.Motivo == MotivoFronteraPersonalDependencia) != (o.EstadoHTTP == 503) {
		return ErrRectificacionDietasInvalida
	}
	return nil
}

var patronSolicitudRectificacionFrontera = regexp.MustCompile(`^srd_[0-9a-f]{32}$`)

func ReferenciaFronteraRectificacionDietasValida(s string) bool {
	return personaldomain.ReferenciaRelacionValida(s) || patronSolicitudRectificacionFrontera.MatchString(s)
}

type RegistradorAuditoriaFronteraRectificacionDietas interface {
	RegistrarAuditoriaFronteraRectificacionDietas(context.Context, OrdenAuditoriaFronteraRectificacionDietas) error
}

const (
	AccionConsultarRectificacionesCompetentesDietas    = "personal.asignacion_dietas.rectificacion.competente.consultar"
	AudienciaConsultarRectificacionesCompetentesDietas = "vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1"
	FinalidadConsultarRectificacionesCompetentesDietas = "consultar_rectificaciones_dietas_competentes"
)

type ProveedorAutorizacionRectificacionesCompetentesDietas interface {
	AutorizarRectificacionesCompetentesDietas(context.Context, personaldomain.MaterialRectificacionesCompetentesDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenConsultaRectificacionesCompetentesDietas struct {
	Material     personaldomain.MaterialRectificacionesCompetentesDietas
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type AsignacionActualRectificacionDietas struct {
	AsignacionRef            string                    `json:"asignacion_ref"`
	CentroRef                string                    `json:"centro_ref"`
	AdministrativoPersonaRef string                    `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string                    `json:"responsable_persona_ref"`
	GrupoDieta               int16                     `json:"grupo_dieta"`
	VigenteDesde             personaldomain.FechaCivil `json:"vigente_desde"`
	Version                  int64                     `json:"version"`
}

type SolicitudCompetenteRectificacionDietas struct {
	SolicitudRef      string                              `json:"solicitud_ref"`
	Estado            string                              `json:"estado"`
	PersonaRef        string                              `json:"persona_ref"`
	EmpleadoRef       string                              `json:"empleado_ref"`
	RelacionRef       string                              `json:"relacion_ref"`
	UnidadRef         string                              `json:"unidad_ref"`
	AsignacionRef     string                              `json:"asignacion_ref"`
	VersionOrigen     int64                               `json:"version_origen"`
	FechaReferencia   personaldomain.FechaCivil           `json:"fecha_referencia"`
	CamposARevisar    []string                            `json:"campos_a_revisar"`
	MotivoRevision    string                              `json:"motivo_revision"`
	DetalleSolicitado string                              `json:"detalle_solicitado"`
	RegistradaEn      time.Time                           `json:"registrada_en"`
	AsignacionActual  AsignacionActualRectificacionDietas `json:"asignacion_actual"`
}

type ResultadoConsultaRectificacionesCompetentesDietas struct {
	ReciboRef           string                                   `json:"recibo_ref"`
	DecisionRef         string                                   `json:"decision_ref"`
	EfectoRef           string                                   `json:"efecto_ref"`
	ConsumoHuellaSHA256 string                                   `json:"consumo_huella_sha256"`
	AuditoriaAD3Ref     string                                   `json:"auditoria_ad3_ref"`
	ConsultadaEn        time.Time                                `json:"consultada_en"`
	Cardinalidad        int                                      `json:"cardinalidad"`
	Solicitudes         []SolicitudCompetenteRectificacionDietas `json:"solicitudes"`
}

type RepositorioRectificacionesCompetentesDietas interface {
	ConsultarRectificacionesCompetentesDietas(context.Context, OrdenConsultaRectificacionesCompetentesDietas) (ResultadoConsultaRectificacionesCompetentesDietas, error)
}
