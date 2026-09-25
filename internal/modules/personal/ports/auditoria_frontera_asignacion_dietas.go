package ports

import (
	"context"
	"errors"
	"regexp"
)

const (
	SuperficieFronteraAsignacionDietas   = "api.personal.asignaciones_dietas"
	RutaFronteraAsignacionesDietas       = "/api/vec/personal/asignaciones-dietas"
	RutaFronteraAsignacionDetalle        = "/api/vec/personal/asignaciones-dietas/detalle"
	RutaFronteraAsignacionGrupo          = "/api/vec/personal/asignaciones-dietas/grupo"
	RutaFronteraRelacionesDietas         = "/api/vec/personal/relaciones-dietas"
	MotivoFronteraPersonalAutenticacion  = "autenticacion_requerida"
	MotivoFronteraPersonalDenegado       = "acceso_denegado"
	MotivoFronteraPersonalDependencia    = "dependencia_no_disponible"
	MotivoFronteraPersonalPeticion       = "peticion_invalida"
	MotivoFronteraPersonalNoEncontrada   = "no_encontrada"
	MotivoFronteraPersonalMetodo         = "metodo_no_permitido"
	MotivoFronteraPersonalRepresentacion = "representacion_no_admitida"
	MotivoFronteraPersonalConflicto      = "conflicto"
)

var ErrAuditoriaFronteraAsignacionInvalida = errors.New("personal: auditoria frontera asignacion invalida")
var patronActorFronteraPersonal = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,512}$`)
var patronCorrelacionFronteraPersonal = regexp.MustCompile(`^corr_[0-9a-f]{32}$`)

type OrdenAuditoriaFronteraAsignacionDietas struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	Accion         string
	ActorRef       string
	RecursoRef     string
	EstadoHTTP     int
}

func (o OrdenAuditoriaFronteraAsignacionDietas) Validar() error {
	if o.CorrelacionRef != "corr_no_disponible" && !patronCorrelacionFronteraPersonal.MatchString(o.CorrelacionRef) {
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	estadoPorMotivo := map[string]int{
		MotivoFronteraPersonalPeticion:       400,
		MotivoFronteraPersonalAutenticacion:  401,
		MotivoFronteraPersonalDenegado:       403,
		MotivoFronteraPersonalNoEncontrada:   404,
		MotivoFronteraPersonalMetodo:         405,
		MotivoFronteraPersonalRepresentacion: 406,
		MotivoFronteraPersonalConflicto:      409,
		MotivoFronteraPersonalDependencia:    503,
	}
	if estadoPorMotivo[o.Motivo] != o.EstadoHTTP {
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	switch o.Ruta {
	case RutaFronteraAsignacionesDietas, RutaFronteraAsignacionDetalle, RutaFronteraAsignacionGrupo, RutaFronteraRelacionesDietas:
	default:
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	switch o.Accion {
	case "consultar", "registrar_inicial", "corregir", "grupo_corregir", "consultar_relaciones", "metodo_no_admitido":
	default:
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	if o.Accion != "metodo_no_admitido" {
		permitida := (o.Ruta == RutaFronteraAsignacionesDietas && o.Accion == "registrar_inicial") ||
			(o.Ruta == RutaFronteraAsignacionDetalle && (o.Accion == "consultar" || o.Accion == "corregir")) ||
			(o.Ruta == RutaFronteraAsignacionGrupo && o.Accion == "grupo_corregir") ||
			(o.Ruta == RutaFronteraRelacionesDietas && o.Accion == "consultar_relaciones")
		if !permitida {
			return ErrAuditoriaFronteraAsignacionInvalida
		}
	}
	if o.Motivo == MotivoFronteraPersonalAutenticacion && o.ActorRef != "" {
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	if o.ActorRef != "" && !patronActorFronteraPersonal.MatchString(o.ActorRef) {
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	if o.RecursoRef != "" && !ReferenciaRelacionValidaFrontera(o.RecursoRef) && !ReferenciaEmpleadoValidaFrontera(o.RecursoRef) {
		return ErrAuditoriaFronteraAsignacionInvalida
	}
	return nil
}

func ReferenciaRelacionValidaFrontera(v string) bool {
	return patronRelacionFronteraPersonal.MatchString(v)
}
func ReferenciaEmpleadoValidaFrontera(v string) bool {
	return patronEmpleadoFronteraPersonal.MatchString(v)
}

var patronRelacionFronteraPersonal = regexp.MustCompile(`^rel_[A-Za-z0-9_-]{22,128}$`)
var patronEmpleadoFronteraPersonal = regexp.MustCompile(`^emp_[A-Za-z0-9_-]{22,128}$`)

type RegistradorAuditoriaFronteraAsignacionDietas interface {
	RegistrarAuditoriaFronteraAsignacionDietas(context.Context, OrdenAuditoriaFronteraAsignacionDietas) error
}
