package calculoexperienciaoficial

import (
	"context"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type PerfilConfirmacionDuradera string

const (
	PerfilConfirmacionExternoOrdinario PerfilConfirmacionDuradera = "externo_ordinario"
	PerfilConfirmacionInternoAlto      PerfilConfirmacionDuradera = "interno_alto"
)

type DatosConfirmacionDuradera struct {
	bloqueoSerializacion
	Perfil                PerfilConfirmacionDuradera
	ReferenciaIntento     string
	Selector              puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo
	Fuente                puertosbolsa.FuenteExactaCalculoReglasBaremo
	Clave                 oficial.ClaveEfectoV1
	Intencion             oficial.IntencionResultadoV1
	Resultado             calculo.ResultadoExperienciaV1
	ResultadoCanonico     []byte
	HuellaResultadoSHA256 string
	AutorizacionLectura   puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	AutorizacionEscritura puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	CorrelacionLectura    dominiovec.ReferenciaCorrelacionAutorizacionV2
	CorrelacionEscritura  dominiovec.ReferenciaCorrelacionAutorizacionV2
	Motivo                dominiovec.ReferenciaEntradaCatalogo
	LecturaNoAntesDe      time.Time
	FuenteSolicitadaEn    time.Time
	EscrituraNoAntesDe    time.Time
	SolicitadaEn          time.Time
}

// SolicitudConfirmacionDuradera es opaca para impedir que una frontera de
// transporte reconstruya una escritura oficial sin pasar por el caso de uso.
type SolicitudConfirmacionDuradera struct {
	bloqueoSerializacion
	datos *DatosConfirmacionDuradera
}

func (s SolicitudConfirmacionDuradera) Datos() (DatosConfirmacionDuradera, error) {
	if s.validar() != nil {
		return DatosConfirmacionDuradera{}, ErrConfirmacionInvalida
	}
	datos := *s.datos
	datos.ResultadoCanonico = append([]byte(nil), s.datos.ResultadoCanonico...)
	return datos, nil
}

type DesenlaceConfirmacionDuradera string

const (
	ConfirmacionCreada      DesenlaceConfirmacionDuradera = "creada"
	ConfirmacionReutilizada DesenlaceConfirmacionDuradera = "reutilizada"
)

type datosResultadoConfirmacionDuradera struct {
	ReferenciaIntento      string
	Recibo                 oficial.ReciboV1
	IndiceEfectoHMACSHA256 string
	HuellaResultadoSHA256  string
	Desenlace              DesenlaceConfirmacionDuradera
}

type ResultadoConfirmacionDuradera struct {
	bloqueoSerializacion
	datos *datosResultadoConfirmacionDuradera
}

func NuevoResultadoConfirmacionDuradera(
	referenciaIntento string,
	recibo oficial.ReciboV1,
	indiceEfectoHMACSHA256 string,
	huellaResultadoSHA256 string,
	desenlace DesenlaceConfirmacionDuradera,
) (ResultadoConfirmacionDuradera, error) {
	resultado := ResultadoConfirmacionDuradera{datos: &datosResultadoConfirmacionDuradera{
		ReferenciaIntento: referenciaIntento, Recibo: recibo,
		IndiceEfectoHMACSHA256: indiceEfectoHMACSHA256,
		HuellaResultadoSHA256:  huellaResultadoSHA256, Desenlace: desenlace,
	}}
	if resultado.validarEstructura() != nil {
		return ResultadoConfirmacionDuradera{}, ErrReciboNoConfiable
	}
	return resultado, nil
}

type ConfirmadorDuradero interface {
	Confirmar(
		context.Context,
		SolicitudConfirmacionDuradera,
	) (ResultadoConfirmacionDuradera, error)
}

type DatosReconciliacionDuradera struct {
	bloqueoSerializacion
	ReferenciaIntento     string
	HuellaIntencionSHA256 string
}

// SolicitudReconciliacionDuradera no admite un DTO libre: se deriva de la
// correlacion V2 y la intencion oficial exactas que originaron el intento.
type SolicitudReconciliacionDuradera struct {
	bloqueoSerializacion
	datos *DatosReconciliacionDuradera
}

func NuevaSolicitudReconciliacionDuradera(
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	intencion oficial.IntencionResultadoV1,
) (SolicitudReconciliacionDuradera, error) {
	referencia, errReferencia := correlacion.ValorCanonico()
	huella, errHuella := intencion.HuellaSHA256()
	if errReferencia != nil || errHuella != nil {
		return SolicitudReconciliacionDuradera{}, ErrConfirmacionInvalida
	}
	return SolicitudReconciliacionDuradera{datos: &DatosReconciliacionDuradera{
		ReferenciaIntento: referencia, HuellaIntencionSHA256: huella,
	}}, nil
}

func (s SolicitudReconciliacionDuradera) Datos() (DatosReconciliacionDuradera, error) {
	if s.datos == nil || !referenciaIntentoValida(s.datos.ReferenciaIntento) ||
		!huellaSHA256Valida(s.datos.HuellaIntencionSHA256) {
		return DatosReconciliacionDuradera{}, ErrConfirmacionInvalida
	}
	return *s.datos, nil
}

type ReconciliadorDuradero interface {
	Reconciliar(
		context.Context,
		SolicitudReconciliacionDuradera,
	) (ResultadoConfirmacionDuradera, error)
}
