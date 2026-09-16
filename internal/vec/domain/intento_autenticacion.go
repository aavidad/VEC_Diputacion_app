package domain

import (
	"errors"
	"time"
)

var ErrIntentoAutenticacionAdministracionInvalido = errors.New("vec: intento de autenticacion de administracion invalido")

type ResultadoIntentoAutenticacionAdministracionV1 string

const (
	ResultadoIntentoAutenticacionAdministracionPermitidoV1 ResultadoIntentoAutenticacionAdministracionV1 = "permitido"
	ResultadoIntentoAutenticacionAdministracionDenegadoV1  ResultadoIntentoAutenticacionAdministracionV1 = "denegado"
	ResultadoIntentoAutenticacionAdministracionErrorV1     ResultadoIntentoAutenticacionAdministracionV1 = "error"
)

type MotivoIntentoAutenticacionAdministracionV1 string

const (
	MotivoIntentoAutenticacionAdministracionPermisoConcedidoV1        MotivoIntentoAutenticacionAdministracionV1 = "permiso_concedido"
	MotivoIntentoAutenticacionAdministracionCertificadoAusenteV1      MotivoIntentoAutenticacionAdministracionV1 = "certificado_ausente"
	MotivoIntentoAutenticacionAdministracionCertificadoNoVerificadoV1 MotivoIntentoAutenticacionAdministracionV1 = "certificado_no_verificado"
	MotivoIntentoAutenticacionAdministracionCertificadoRevocadoV1     MotivoIntentoAutenticacionAdministracionV1 = "certificado_revocado"
	MotivoIntentoAutenticacionAdministracionSesionRevocadaV1          MotivoIntentoAutenticacionAdministracionV1 = "sesion_revocada"
	MotivoIntentoAutenticacionAdministracionPermisoDenegadoV1         MotivoIntentoAutenticacionAdministracionV1 = "permiso_denegado"
	MotivoIntentoAutenticacionAdministracionCanalNoValidoV1           MotivoIntentoAutenticacionAdministracionV1 = "canal_no_valido"
	MotivoIntentoAutenticacionAdministracionErrorInfraestructuraV1    MotivoIntentoAutenticacionAdministracionV1 = "error_infraestructura"
)

type EstadoActorIntentoAutenticacionV1 string

const (
	EstadoActorIntentoAutenticacionAcreditadoV1   EstadoActorIntentoAutenticacionV1 = "acreditado"
	EstadoActorIntentoAutenticacionNoAcreditadoV1 EstadoActorIntentoAutenticacionV1 = "no_acreditado"
)

// IntentoAutenticacionAdministracionV1 es el evento minimizado que la
// frontera entrega a la auditoria T13. IntentoRef reutiliza la clave HMAC de
// idempotencia ya generada por la frontera; CorrelacionRef usa el generador
// CSPRNG existente de autorizacion V2. Este contrato solo las valida y nunca
// crea identidad, HMAC ni material de autenticacion.
type IntentoAutenticacionAdministracionV1 struct {
	IntentoRef     string                                        `json:"intento_ref"`
	CorrelacionRef string                                        `json:"correlacion_ref"`
	InstanteUTC    time.Time                                     `json:"instante_utc"`
	Superficie     SuperficieAutenticacionActorV1                `json:"superficie"`
	Resultado      ResultadoIntentoAutenticacionAdministracionV1 `json:"resultado"`
	Motivo         MotivoIntentoAutenticacionAdministracionV1    `json:"motivo"`
	ActorEstado    EstadoActorIntentoAutenticacionV1             `json:"actor_estado"`
	HMACActor      string                                        `json:"hmac_actor"`
}

func (i IntentoAutenticacionAdministracionV1) Validar() error {
	if !esHuellaHMACSHA256(i.IntentoRef) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(i.CorrelacionRef) ||
		!instanteAutorizacionCanonico(i.InstanteUTC) ||
		i.Superficie != SuperficieAutenticacionAdministracionPrivilegiadaV1 {
		return ErrIntentoAutenticacionAdministracionInvalido
	}

	switch i.ActorEstado {
	case EstadoActorIntentoAutenticacionAcreditadoV1:
		if !esHuellaHMACSHA256(i.HMACActor) {
			return ErrIntentoAutenticacionAdministracionInvalido
		}
	case EstadoActorIntentoAutenticacionNoAcreditadoV1:
		if i.HMACActor != "" {
			return ErrIntentoAutenticacionAdministracionInvalido
		}
	default:
		return ErrIntentoAutenticacionAdministracionInvalido
	}

	switch i.Motivo {
	case MotivoIntentoAutenticacionAdministracionPermisoConcedidoV1:
		if i.Resultado == ResultadoIntentoAutenticacionAdministracionPermitidoV1 &&
			i.ActorEstado == EstadoActorIntentoAutenticacionAcreditadoV1 {
			return nil
		}
	case MotivoIntentoAutenticacionAdministracionCertificadoAusenteV1:
		if i.Resultado == ResultadoIntentoAutenticacionAdministracionDenegadoV1 &&
			i.ActorEstado == EstadoActorIntentoAutenticacionNoAcreditadoV1 {
			return nil
		}
	case MotivoIntentoAutenticacionAdministracionCertificadoNoVerificadoV1,
		MotivoIntentoAutenticacionAdministracionCertificadoRevocadoV1,
		MotivoIntentoAutenticacionAdministracionCanalNoValidoV1:
		if i.Resultado == ResultadoIntentoAutenticacionAdministracionDenegadoV1 &&
			i.ActorEstado == EstadoActorIntentoAutenticacionNoAcreditadoV1 {
			return nil
		}
	case MotivoIntentoAutenticacionAdministracionSesionRevocadaV1,
		MotivoIntentoAutenticacionAdministracionPermisoDenegadoV1:
		if i.Resultado == ResultadoIntentoAutenticacionAdministracionDenegadoV1 &&
			i.ActorEstado == EstadoActorIntentoAutenticacionAcreditadoV1 {
			return nil
		}
	case MotivoIntentoAutenticacionAdministracionErrorInfraestructuraV1:
		if i.Resultado == ResultadoIntentoAutenticacionAdministracionErrorV1 {
			return nil
		}
	}
	return ErrIntentoAutenticacionAdministracionInvalido
}
