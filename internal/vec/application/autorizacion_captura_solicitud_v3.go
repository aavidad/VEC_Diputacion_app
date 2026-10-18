package application

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrCapturaEvaluacionSolicitudLigadaV3Invalida = errors.New("vec: captura de evaluacion ligada V3 invalida")

// CapturaEvaluacionSolicitudLigadaV3 conserva exclusivamente la instantanea
// evaluada por el PDP. No es una lectura de estado actual ni una concesion.
// Solo la emision nominal puede ligarla al material resultante.
type CapturaEvaluacionSolicitudLigadaV3 struct {
	bloqueoSerializacionCapturaV3
	datos *datosCapturaEvaluacionSolicitudLigadaV3
}

type datosCapturaEvaluacionSolicitudLigadaV3 struct {
	instantanea     domain.InstantaneaAutorizacion
	solicitudHuella string
	resultadoRef    string
	resultadoHuella string
	decisionHuella  string
	materialHuella  string
	audiencia       string
	validaHasta     time.Time
}

type bloqueoSerializacionCapturaV3 struct{}

func (bloqueoSerializacionCapturaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
}
func (*bloqueoSerializacionCapturaV3) UnmarshalJSON([]byte) error {
	return ErrCapturaEvaluacionSolicitudLigadaV3Invalida
}
func (bloqueoSerializacionCapturaV3) MarshalText() ([]byte, error) {
	return nil, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
}
func (*bloqueoSerializacionCapturaV3) UnmarshalText([]byte) error {
	return ErrCapturaEvaluacionSolicitudLigadaV3Invalida
}

func (CapturaEvaluacionSolicitudLigadaV3) String() string {
	return "vec: captura de evaluacion ligada V3"
}
func (c CapturaEvaluacionSolicitudLigadaV3) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (c CapturaEvaluacionSolicitudLigadaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func nuevaCapturaEvaluacionSolicitudLigadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
	decision domain.DecisionAutorizacionLigadaV3,
	instantanea domain.InstantaneaAutorizacion,
) (CapturaEvaluacionSolicitudLigadaV3, error) {
	concedida, _, err := decision.Resultado()
	huellaSolicitud, errSolicitud := domain.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	huellaDecision, errDecision := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil || errSolicitud != nil || errDecision != nil || !concedida ||
		decision.ValidarPara(solicitud) != nil || resultado.Validar() != nil || instantanea.Validar() != nil {
		return CapturaEvaluacionSolicitudLigadaV3{}, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	_, hasta, err := decision.VentanaValidez()
	if err != nil {
		return CapturaEvaluacionSolicitudLigadaV3{}, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	copia, err := clonarInstantaneaAutorizacionLigadaV3(instantanea)
	if err != nil {
		return CapturaEvaluacionSolicitudLigadaV3{}, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	return CapturaEvaluacionSolicitudLigadaV3{datos: &datosCapturaEvaluacionSolicitudLigadaV3{
		instantanea: copia, solicitudHuella: huellaSolicitud,
		resultadoRef: resultado.RegistroContextoRef, resultadoHuella: resultado.HuellaSHA256,
		decisionHuella: huellaDecision,
		validaHasta:    hasta,
	}}, nil
}

// LigarMaterial comprueba la preimagen emitida y crea otra captura cerrada.
// La captura de la aplicacion permanece inutilizable sin esta ligadura.
func (c CapturaEvaluacionSolicitudLigadaV3) LigarMaterial(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	audiencia string,
) (ports.CapturaEvaluacionSolicitudLigadaV3, error) {
	if c.datos == nil || c.datos.materialHuella != "" || audiencia == "" ||
		c.coincide(solicitud, resultado, decision, confirmacion) != nil {
		return nil, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil || !ports.MaterialAtestadoLigadoV3(
		solicitud, decision, confirmacion, resultado,
		datosSolicitud.ReferenciaMotivo, material, audiencia,
	) {
		return nil, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	huella, err := material.HuellaConjuntoSHA256()
	if err != nil {
		return nil, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	copia := *c.datos
	copia.materialHuella, copia.audiencia = huella, audiencia
	return CapturaEvaluacionSolicitudLigadaV3{datos: &copia}, nil
}

var _ ports.CapturaEvaluacionSolicitudLigadaV3 = CapturaEvaluacionSolicitudLigadaV3{}

// InstantaneaPara entrega una copia de la captura evaluada solo al portador
// de todos los objetos nominales de la emision. SQL sigue comprobando el
// estado actual y el consumo unico en la transaccion del efecto.
func (c CapturaEvaluacionSolicitudLigadaV3) InstantaneaPara(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	audiencia string,
	ahora time.Time,
) (domain.InstantaneaAutorizacion, error) {
	if c.datos == nil || c.datos.materialHuella == "" || audiencia != c.datos.audiencia ||
		c.coincide(solicitud, resultado, decision, confirmacion) != nil ||
		ahora.IsZero() || !ahora.Before(c.datos.validaHasta) ||
		!ahora.Before(material.ResumenCapacidad().ExpiraEn()) ||
		!confirmacion.DentroDeVentanaEn(ahora) {
		return domain.InstantaneaAutorizacion{}, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	huella, err := material.HuellaConjuntoSHA256()
	datosSolicitud, errSolicitud := solicitud.Datos()
	if err != nil || errSolicitud != nil || subtle.ConstantTimeCompare([]byte(huella), []byte(c.datos.materialHuella)) != 1 ||
		!ports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, resultado,
			datosSolicitud.ReferenciaMotivo, material, audiencia) {
		return domain.InstantaneaAutorizacion{}, ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	return clonarInstantaneaAutorizacionLigadaV3(c.datos.instantanea)
}

func (c CapturaEvaluacionSolicitudLigadaV3) coincide(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) error {
	if c.datos == nil || resultado.Validar() != nil || decision.ValidarPara(solicitud) != nil {
		return ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	huellaSolicitud, errSolicitud := domain.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	huellaDecision, errDecision := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	datosSolicitud, errSolicitudDatos := solicitud.Datos()
	if errSolicitud != nil || errDecision != nil || errConfirmacion != nil || errSolicitudDatos != nil {
		return errors.Join(ErrCapturaEvaluacionSolicitudLigadaV3Invalida,
			errSolicitud, errDecision, errConfirmacion, errSolicitudDatos)
	}
	orden, errOrden := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, datosSolicitud.ReferenciaMotivo, resultado,
	)
	if errOrden != nil {
		return errors.Join(ErrCapturaEvaluacionSolicitudLigadaV3Invalida, errOrden)
	}
	if confirmacion.ValidarPara(orden) != nil ||
		subtle.ConstantTimeCompare([]byte(huellaSolicitud), []byte(c.datos.solicitudHuella)) != 1 ||
		subtle.ConstantTimeCompare([]byte(huellaDecision), []byte(c.datos.decisionHuella)) != 1 ||
		resultado.RegistroContextoRef != c.datos.resultadoRef ||
		subtle.ConstantTimeCompare([]byte(resultado.HuellaSHA256), []byte(c.datos.resultadoHuella)) != 1 ||
		datosConfirmacion.DecisionHuellaSHA256 != c.datos.decisionHuella ||
		datosConfirmacion.ValidaHasta.After(c.datos.validaHasta) {
		return ErrCapturaEvaluacionSolicitudLigadaV3Invalida
	}
	return nil
}
