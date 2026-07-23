package ports

import (
	"context"
	"crypto/hmac"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const TiempoMaximoFuenteAnalisis = 15 * time.Second

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
)

type ResultadoValidacionRC struct {
	datos *DatosResultadoValidacionRC
}

type DatosResultadoValidacionRC struct {
	PeticionRef        string
	HuellaPeticionHMAC string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionExpediente  uint64
	Validacion         domain.ValidacionRC
	Motivo             MotivoFuenteAnalisis
}

func NuevoResultadoValidacionRC(
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
) (ResultadoValidacionRC, error) {
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return ResultadoValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado := ResultadoValidacionRC{datos: &DatosResultadoValidacionRC{
		PeticionRef:        datosSolicitud.PeticionRef,
		HuellaPeticionHMAC: datosSolicitud.HuellaPeticionHMAC,
		OrganizacionRef:    datosSolicitud.OrganizacionRef,
		ExpedienteRef:      datosSolicitud.ExpedienteRef,
		VersionExpediente:  datosSolicitud.VersionExpediente,
		Validacion:         clonarValidacionRC(validacion),
		Motivo:             motivo.clonar(),
	}}
	if resultado.ValidarPara(solicitud, validacion.ValidadaEn) != nil {
		return ResultadoValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return resultado, nil
}

func (r ResultadoValidacionRC) Datos() (DatosResultadoValidacionRC, error) {
	if r.datos == nil {
		return DatosResultadoValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	copia := *r.datos
	copia.Validacion = clonarValidacionRC(copia.Validacion)
	copia.Motivo = copia.Motivo.clonar()
	return copia, nil
}

func (r ResultadoValidacionRC) ValidarPara(
	solicitud SolicitudValidarRC,
	finalizadaEn time.Time,
) error {
	s, errSolicitud := solicitud.Datos()
	datos, errResultado := r.Datos()
	if errSolicitud != nil || errResultado != nil ||
		!instanteFuenteAnalisisCanonico(finalizadaEn) ||
		datos.PeticionRef != s.PeticionRef ||
		!sellosPeticionFuenteAnalisisIguales(
			datos.HuellaPeticionHMAC,
			s.HuellaPeticionHMAC,
		) ||
		datos.OrganizacionRef != s.OrganizacionRef ||
		datos.ExpedienteRef != s.ExpedienteRef ||
		datos.VersionExpediente != s.VersionExpediente ||
		datos.Validacion.EntradaRef != s.Entrada.Referencia ||
		datos.Validacion.HuellaEntradaSHA256 != s.Entrada.HuellaSHA256 ||
		datos.Validacion.ValidadaEn.Before(s.SolicitadaEn) ||
		datos.Validacion.ValidadaEn.After(finalizadaEn) ||
		datos.Validacion.Motivo != "" {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	materializada, err := materializarMotivoValidacionRC(
		datos.Validacion,
		datos.Motivo,
	)
	if err != nil || materializada.Validar() != nil ||
		!importeFuenteAnalisisValidoOpcional(materializada.Importe) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

type FuentePresupuestaria interface {
	ValidarRC(context.Context, SolicitudValidarRC) (ResultadoValidacionRC, error)
}

type ResultadoCalculoCoste struct {
	datos *DatosResultadoCalculoCoste
}

type DatosResultadoCalculoCoste struct {
	PeticionRef        string
	HuellaPeticionHMAC string
	OrganizacionRef    string
	ExpedienteRef      string
	VersionExpediente  uint64
	FuenteRef          string
	ReciboRef          string
	Importe            domain.Importe
	CalculadoEn        time.Time
}

func NuevoResultadoCalculoCoste(
	solicitud SolicitudCalcularCoste,
	fuenteRef string,
	reciboRef string,
	importe domain.Importe,
	calculadoEn time.Time,
) (ResultadoCalculoCoste, error) {
	s, err := solicitud.Datos()
	if err != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado := ResultadoCalculoCoste{datos: &DatosResultadoCalculoCoste{
		PeticionRef: s.PeticionRef, HuellaPeticionHMAC: s.HuellaPeticionHMAC,
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente: s.VersionExpediente, FuenteRef: fuenteRef,
		ReciboRef: reciboRef, Importe: importe, CalculadoEn: calculadoEn,
	}}
	if resultado.validarEstructuraPara(solicitud) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return resultado, nil
}

func (r ResultadoCalculoCoste) Datos() (DatosResultadoCalculoCoste, error) {
	if r.datos == nil {
		return DatosResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return *r.datos, nil
}

func (r ResultadoCalculoCoste) validarEstructuraPara(
	solicitud SolicitudCalcularCoste,
) error {
	s, errSolicitud := solicitud.Datos()
	datos, errResultado := r.Datos()
	if errSolicitud != nil || errResultado != nil ||
		datos.PeticionRef != s.PeticionRef ||
		!sellosPeticionFuenteAnalisisIguales(
			datos.HuellaPeticionHMAC,
			s.HuellaPeticionHMAC,
		) ||
		datos.OrganizacionRef != s.OrganizacionRef ||
		datos.ExpedienteRef != s.ExpedienteRef ||
		datos.VersionExpediente != s.VersionExpediente ||
		!domain.ReferenciaOpacaValida(datos.FuenteRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRef) ||
		!importeFuenteAnalisisValido(datos.Importe) ||
		!instanteFuenteAnalisisCanonico(datos.CalculadoEn) ||
		datos.CalculadoEn.Before(s.SolicitadaEn) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (r ResultadoCalculoCoste) ValidarPara(
	solicitud SolicitudCalcularCoste,
	finalizadaEn time.Time,
) error {
	datos, err := r.Datos()
	if r.validarEstructuraPara(solicitud) != nil || err != nil ||
		!instanteFuenteAnalisisCanonico(finalizadaEn) ||
		datos.CalculadoEn.After(finalizadaEn) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

type CalculadorCostePersonal interface {
	CalcularCoste(
		context.Context,
		SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error)
}

func ValidarRCConFuente(
	ctx context.Context,
	fuente FuentePresupuestaria,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudValidarRC,
) (domain.ValidacionRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(fuente) ||
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
	finalizadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return domain.ValidacionRC{}, errorDisponibilidadFuente(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud, finalizadaEn) != nil {
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
	reloj RelojFuenteAnalisis,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(calculador) ||
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
	finalizadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ResultadoCalculoCoste{}, errorDisponibilidadFuente(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud, finalizadaEn) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return resultado.clonar(), nil
}

func (r ResultadoCalculoCoste) clonar() ResultadoCalculoCoste {
	if r.datos == nil {
		return ResultadoCalculoCoste{}
	}
	datos := *r.datos
	return ResultadoCalculoCoste{datos: &datos}
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

func sellosPeticionFuenteAnalisisIguales(primero, segundo string) bool {
	return selloPeticionFuenteAnalisisValido(primero) &&
		selloPeticionFuenteAnalisisValido(segundo) &&
		hmac.Equal([]byte(primero), []byte(segundo))
}

func errorDisponibilidadFuente(publico, causa error) error {
	return errorFuenteAnalisis{publico: publico, causa: causa}
}

type errorFuenteAnalisis struct {
	publico error
	causa   error
}

func (e errorFuenteAnalisis) Error() string {
	return e.publico.Error()
}

func (e errorFuenteAnalisis) Unwrap() []error {
	return []error{e.publico, e.causa}
}
