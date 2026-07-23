package ports

import (
	"bytes"
	"context"
	"crypto/hmac"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	TiempoMaximoFuenteAnalisis            = 5 * time.Second
	VigenciaMaximaRespuestaFuenteAnalisis = 5 * time.Second
)

var (
	ErrPeticionFuenteAnalisisInvalida = errors.New(
		"contratacion temporal: peticion a fuente de analisis invalida",
	)
	ErrFuentePresupuestariaNoDisponible = errors.New(
		"contratacion temporal: fuente presupuestaria no disponible",
	)
	ErrCalculadorCosteNoDisponible = errors.New(
		"contratacion temporal: calculador de coste no disponible",
	)
	ErrInfraestructuraFuenteAnalisisNoDisponible = errors.New(
		"contratacion temporal: infraestructura de fuente de analisis no disponible",
	)
	ErrResultadoFuenteAnalisisNoConfiable = errors.New(
		"contratacion temporal: resultado de fuente de analisis no confiable",
	)
	ErrVerificacionFuenteAnalisisNoDisponible = errors.New(
		"contratacion temporal: verificacion de fuente de analisis no disponible",
	)
	ErrConsumoFuenteAnalisisNoDisponible = errors.New(
		"contratacion temporal: consumo de fuente de analisis no disponible",
	)
	ErrRespuestaFuenteAnalisisYaConsumida = errors.New(
		"contratacion temporal: respuesta de fuente de analisis ya consumida con otros datos",
	)
)

type FuentePresupuestaria interface {
	PresentadorAutoridadFuenteAnalisis
	ValidarRC(context.Context, SolicitudValidarRC) (ResultadoValidacionRC, error)
}

type CalculadorCostePersonal interface {
	PresentadorAutoridadFuenteAnalisis
	CalcularCoste(
		context.Context,
		SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error)
}

func ValidarRCConFuente(
	ctx context.Context,
	fuente FuentePresupuestaria,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicaciones VerificadorPublicacionMotivoFuenteAnalisis,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudValidarRC,
) (domain.ValidacionRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(fuente) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(publicaciones) ||
		dependenciaNulaFuenteAnalisis(consumidor) ||
		dependenciaNulaFuenteAnalisis(reloj) || solicitud.Validar() != nil {
		return domain.ValidacionRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, TiempoMaximoFuenteAnalisis)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	datosSolicitud, errDatosSolicitud := solicitud.Datos()
	materialPeticion := materialDesafioSolicitudFuenteAnalisis(
		solicitud.datosCanonicos(),
		datosSolicitud.HuellaPeticionHMAC,
	)
	if errDatosSolicitud != nil ||
		datosSolicitud.OrganizacionRef != confianza.organizacionRef ||
		len(materialPeticion) == 0 {
		return domain.ValidacionRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	identidadFuente, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion, fuente, confianza, materialPeticion,
		RolFuentePresupuestaria, reloj.Ahora(),
	)
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	identidadVerificador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion, verificador, confianza, materialPeticion,
		RolVerificadorRespuesta, reloj.Ahora(),
	)
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	identidadPublicador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion, publicaciones, confianza, materialPeticion,
		RolPublicadorCatalogo, reloj.Ahora(),
	)
	if err != nil || !autoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
		identidadPublicador,
	) {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := fuente.ValidarRC(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	if errFuente != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrFuentePresupuestariaNoDisponible,
			errFuente,
		)
	}
	recibidaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	if errDatosResultado != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			identidadFuente.autoridadRef {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	confirmacion, err := verificarRespuestaFuenteAnalisis(
		operacion,
		verificador,
		identidadVerificador,
		resultado.solicitudVerificacion(),
		reloj,
	)
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	confirmacionMotivo, err := verificarMotivoResultadoRC(
		operacion,
		publicaciones,
		identidadPublicador,
		resultado,
		reloj,
	)
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	orden, err := nuevaOrdenConsumoResultadoRC(
		solicitud,
		resultado,
		confirmacion,
		confirmacionMotivo,
	)
	if err != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	antesConsumo := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrConsumoFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	if confirmacion.ValidarPara(
		resultado.solicitudVerificacion(),
		antesConsumo,
	) != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	recibo, errConsumo := consumidor.ConsumirRespuestaFuenteAnalisis(operacion, orden)
	if errConsumo != nil {
		if errContexto := operacion.Err(); errContexto != nil {
			return domain.ValidacionRC{}, errorDisponibilidadFuente(
				ErrConsumoFuenteAnalisisNoDisponible,
				errContexto,
			)
		}
		if errors.Is(errConsumo, ErrRespuestaFuenteAnalisisYaConsumida) {
			return domain.ValidacionRC{}, ErrRespuestaFuenteAnalisisYaConsumida
		}
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrConsumoFuenteAnalisisNoDisponible,
			errConsumo,
		)
	}
	if recibo.ValidarPara(orden) != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	datos, _ := resultado.Datos()
	validacion, err := materializarMotivoValidacionRC(datos.Validacion, datos.Motivo)
	if err != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return clonarValidacionRC(validacion), nil
}

func CalcularCosteConFuente(
	ctx context.Context,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(calculador) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(consumidor) ||
		dependenciaNulaFuenteAnalisis(reloj) || solicitud.Validar() != nil {
		return ResultadoCalculoCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, TiempoMaximoFuenteAnalisis)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	datosSolicitud, errDatosSolicitud := solicitud.Datos()
	materialPeticion := materialDesafioSolicitudFuenteAnalisis(
		solicitud.datosCanonicos(),
		datosSolicitud.HuellaPeticionHMAC,
	)
	if errDatosSolicitud != nil ||
		datosSolicitud.OrganizacionRef != confianza.organizacionRef ||
		len(materialPeticion) == 0 {
		return ResultadoCalculoCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	identidadFuente, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion, calculador, confianza, materialPeticion,
		RolCalculadorCoste, reloj.Ahora(),
	)
	if err != nil {
		return ResultadoCalculoCoste{}, err
	}
	identidadVerificador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion, verificador, confianza, materialPeticion,
		RolVerificadorRespuesta, reloj.Ahora(),
	)
	if err != nil || !autoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
	) {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := calculador.CalcularCoste(operacion, solicitud)
	if err := operacion.Err(); err != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	if errFuente != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrCalculadorCosteNoDisponible,
			errFuente,
		)
	}
	recibidaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	if errDatosResultado != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			identidadFuente.autoridadRef {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	confirmacion, err := verificarRespuestaFuenteAnalisis(
		operacion,
		verificador,
		identidadVerificador,
		resultado.solicitudVerificacion(),
		reloj,
	)
	if err != nil {
		return ResultadoCalculoCoste{}, err
	}
	orden, err := nuevaOrdenConsumoResultadoCoste(solicitud, resultado, confirmacion)
	if err != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	antesConsumo := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrConsumoFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	if confirmacion.ValidarPara(
		resultado.solicitudVerificacion(),
		antesConsumo,
	) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	recibo, errConsumo := consumidor.ConsumirRespuestaFuenteAnalisis(operacion, orden)
	if errConsumo != nil {
		if errContexto := operacion.Err(); errContexto != nil {
			return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
				ErrConsumoFuenteAnalisisNoDisponible,
				errContexto,
			)
		}
		if errors.Is(errConsumo, ErrRespuestaFuenteAnalisisYaConsumida) {
			return ResultadoCalculoCoste{},
				ErrRespuestaFuenteAnalisisYaConsumida
		}
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrConsumoFuenteAnalisisNoDisponible,
			errConsumo,
		)
	}
	if recibo.ValidarPara(orden) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return resultado.clonar(), nil
}

func verificarRespuestaFuenteAnalisis(
	ctx context.Context,
	verificador VerificadorRespuestaFuenteAnalisis,
	identidadVerificador identidadAutoridadFuenteAnalisis,
	solicitud SolicitudVerificarRespuestaFuenteAnalisis,
	reloj RelojFuenteAnalisis,
) (ConfirmacionRespuestaFuenteAnalisis, error) {
	confirmacion, errVerificacion := verificador.VerificarRespuestaFuenteAnalisis(
		ctx,
		solicitud,
	)
	if err := ctx.Err(); err != nil {
		return ConfirmacionRespuestaFuenteAnalisis{}, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			err,
		)
	}
	if errVerificacion != nil {
		return ConfirmacionRespuestaFuenteAnalisis{}, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			errVerificacion,
		)
	}
	verificadaEn := reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return ConfirmacionRespuestaFuenteAnalisis{}, errorDisponibilidadFuente(
			ErrVerificacionFuenteAnalisisNoDisponible,
			err,
		)
	}
	datosConfirmacion, errDatos := confirmacion.Datos()
	if errDatos != nil ||
		datosConfirmacion.VerificadorRef != identidadVerificador.autoridadRef ||
		confirmacion.ValidarPara(solicitud, verificadaEn) != nil {
		return ConfirmacionRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return confirmacion, nil
}

func clonarValidacionRC(validacion domain.ValidacionRC) domain.ValidacionRC {
	if validacion.FechaRC != nil {
		fecha := *validacion.FechaRC
		validacion.FechaRC = &fecha
	}
	if validacion.Importe != nil {
		importe := *validacion.Importe
		validacion.Importe = &importe
	}
	return validacion
}

func dependenciaNulaFuenteAnalisis(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func materialDesafioSolicitudFuenteAnalisis(
	canonica []byte,
	sello string,
) []byte {
	if len(canonica) == 0 || !selloPeticionFuenteAnalisisValido(sello) {
		return nil
	}
	buffer := bytes.NewBuffer(make([]byte, 0, len(canonica)+len(sello)+4))
	_, _ = buffer.Write(canonica)
	escribirTextoAutoridad(buffer, sello)
	return buffer.Bytes()
}

func sellosPeticionFuenteAnalisisIguales(primero, segundo string) bool {
	return selloPeticionFuenteAnalisisValido(primero) &&
		selloPeticionFuenteAnalisisValido(segundo) &&
		hmac.Equal([]byte(primero), []byte(segundo))
}

func errorDisponibilidadFuente(publico, causa error) error {
	var contexto error
	switch {
	case errors.Is(causa, context.Canceled):
		contexto = context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		contexto = context.DeadlineExceeded
	}
	return errorFuenteAnalisis{publico: publico, contexto: contexto}
}

type errorFuenteAnalisis struct {
	publico  error
	contexto error
}

func (e errorFuenteAnalisis) Error() string {
	return e.publico.Error()
}

func (e errorFuenteAnalisis) Unwrap() []error {
	if e.contexto == nil {
		return []error{e.publico}
	}
	return []error{e.publico, e.contexto}
}
