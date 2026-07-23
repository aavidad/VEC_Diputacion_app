package ports

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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
	ErrResultadoFuenteAnalisisNoConfiable = errors.New(
		"contratacion temporal: resultado de fuente de analisis no confiable",
	)
)

// SolicitudValidarRC contiene únicamente referencias y datos presupuestarios
// necesarios para el cotejo. La fuente no recibe identidad personal ni datos
// de sesión del cliente.
type SolicitudValidarRC struct {
	PeticionRef     string
	OrganizacionRef string
	ExpedienteRef   string
	Entrada         domain.VinculoEntradaRC
	Declaracion     domain.DeclaracionRC
	SolicitadaEn    time.Time
}

func (s SolicitudValidarRC) Validar() error {
	if !domain.ReferenciaOpacaValida(s.PeticionRef) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.Entrada.Validar() != nil || s.Declaracion.Validar() != nil ||
		!domain.InstanteUTCCanonico(s.SolicitadaEn) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

// ResultadoValidacionRC conserva la ligadura con la petición exacta. Una
// respuesta estructuralmente válida no acredita disponibilidad: cualquier
// error del conector invalida conjuntamente valor y resultado.
type ResultadoValidacionRC struct {
	PeticionRef string
	Validacion  domain.ValidacionRC
}

func (r ResultadoValidacionRC) ValidarPara(solicitud SolicitudValidarRC) error {
	if solicitud.Validar() != nil ||
		r.PeticionRef != solicitud.PeticionRef ||
		r.Validacion.Validar() != nil ||
		r.Validacion.EntradaRef != solicitud.Entrada.Referencia ||
		r.Validacion.HuellaEntradaSHA256 != solicitud.Entrada.HuellaSHA256 ||
		r.Validacion.ValidadaEn.Before(solicitud.SolicitadaEn) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

type FuentePresupuestaria interface {
	ValidarRC(context.Context, SolicitudValidarRC) (ResultadoValidacionRC, error)
}

// SolicitudCalcularCoste liga el cálculo a una petición opaca e irrepetible.
// Los catálogos publicados deciden qué modalidades y categorías existen; este
// contrato solo aplica invariantes técnicas.
type SolicitudCalcularCoste struct {
	PeticionRef     string
	OrganizacionRef string
	ExpedienteRef   string
	CategoriaRef    string
	GrupoSubgrupo   string
	ModalidadClave  domain.ClaveCatalogo
	CausaClave      domain.ClaveCatalogo
	Periodo         domain.PeriodoPrevisto
	Jornada         domain.JornadaDiezmilesimas
	SolicitadaEn    time.Time
}

func (s SolicitudCalcularCoste) Validar() error {
	if !domain.ReferenciaOpacaValida(s.PeticionRef) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.CategoriaRef) ||
		!domain.GrupoSubgrupoValido(s.GrupoSubgrupo) ||
		!s.ModalidadClave.Valida() || !s.CausaClave.Valida() ||
		s.Periodo.Validar() != nil || s.Jornada.Validar() != nil ||
		!domain.InstanteUTCCanonico(s.SolicitadaEn) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

type ResultadoCalculoCoste struct {
	PeticionRef   string
	ExpedienteRef string
	FuenteRef     string
	ReciboRef     string
	Importe       domain.Importe
	CalculadoEn   time.Time
}

func (r ResultadoCalculoCoste) ValidarPara(solicitud SolicitudCalcularCoste) error {
	if solicitud.Validar() != nil || r.PeticionRef != solicitud.PeticionRef ||
		r.ExpedienteRef != solicitud.ExpedienteRef ||
		!domain.ReferenciaOpacaValida(r.FuenteRef) ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		r.Importe.Validar(false) != nil ||
		!domain.InstanteUTCCanonico(r.CalculadoEn) ||
		r.CalculadoEn.Before(solicitud.SolicitadaEn) {
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

// ValidarRCConFuente aplica el fallo cerrado común a todos los adaptadores.
// En particular, una fuente caída no puede transformarse en «RC no requerida».
func ValidarRCConFuente(
	ctx context.Context,
	fuente FuentePresupuestaria,
	solicitud SolicitudValidarRC,
) (domain.ValidacionRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(fuente) ||
		solicitud.Validar() != nil {
		return domain.ValidacionRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.ValidacionRC{}, nuevoErrorFuenteAnalisis(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	resultado, err := fuente.ValidarRC(ctx, solicitud)
	if err != nil {
		return domain.ValidacionRC{}, nuevoErrorFuenteAnalisis(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.ValidacionRC{}, nuevoErrorFuenteAnalisis(
			ErrFuentePresupuestariaNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return domain.ValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return clonarValidacionRC(resultado.Validacion), nil
}

func CalcularCosteConFuente(
	ctx context.Context,
	calculador CalculadorCostePersonal,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(calculador) ||
		solicitud.Validar() != nil {
		return ResultadoCalculoCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ResultadoCalculoCoste{}, nuevoErrorFuenteAnalisis(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	resultado, err := calculador.CalcularCoste(ctx, solicitud)
	if err != nil {
		return ResultadoCalculoCoste{}, nuevoErrorFuenteAnalisis(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	if err := ctx.Err(); err != nil {
		return ResultadoCalculoCoste{}, nuevoErrorFuenteAnalisis(
			ErrCalculadorCosteNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return resultado, nil
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
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

// errorFuenteAnalisis conserva la causa para errors.Is sin exponer el texto
// del proveedor en API, logs o clientes.
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

func nuevoErrorFuenteAnalisis(publico, causa error) error {
	return errorFuenteAnalisis{publico: publico, causa: causa}
}
