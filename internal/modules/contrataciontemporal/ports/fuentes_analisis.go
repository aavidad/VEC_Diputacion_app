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
