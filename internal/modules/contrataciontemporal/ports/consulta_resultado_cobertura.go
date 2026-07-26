package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	// AccionConsultarResultadoCobertura y FinalidadRecuperarResultadoCobertura
	// son autoridad compilada de aplicación. Ningún canal puede elegirlas.
	AccionConsultarResultadoCobertura AccionLecturaResultadoCobertura = "" +
		"contratacion_temporal.cobertura.resultado.consultar"
	FinalidadRecuperarResultadoCobertura FinalidadLecturaResultadoCobertura = "" +
		"recuperar_resultado_propio_decision_cobertura"

	redaccionSolicitudLecturaResultadoCobertura = "" +
		"[SOLICITUD-LECTURA-RESULTADO-COBERTURA-REDACTADA]"
)

var ErrSolicitudLecturaResultadoCoberturaInvalida = errors.New(
	"contratacion temporal: solicitud de lectura propia de resultado invalida",
)

type AccionLecturaResultadoCobertura string
type FinalidadLecturaResultadoCobertura string

// ContextoRecuperacionResultadoCobertura procede íntegramente de la frontera
// corporativa. El canal no declara organización, actor ni perfil.
type ContextoRecuperacionResultadoCobertura struct {
	bloqueoSerializacionConsultaResultadoCobertura
	solicitud       SolicitudResolverContextoAutorizacionAltaV3
	contexto        ContextoAutorizacionAltaV3
	organizacionRef string
}

func NuevoContextoRecuperacionResultadoCobertura(
	solicitud SolicitudResolverContextoAutorizacionAltaV3,
	contexto ContextoAutorizacionAltaV3,
	organizacionRef string,
	evaluadoEn time.Time,
) (ContextoRecuperacionResultadoCobertura, error) {
	resultado := ContextoRecuperacionResultadoCobertura{
		solicitud:       solicitud,
		contexto:        contexto,
		organizacionRef: organizacionRef,
	}
	if _, _, _, err := resultado.DatosEn(evaluadoEn); err != nil {
		return ContextoRecuperacionResultadoCobertura{},
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	return resultado, nil
}

func (c ContextoRecuperacionResultadoCobertura) DatosEn(
	evaluadoEn time.Time,
) (
	SolicitudResolverContextoAutorizacionAltaV3,
	ContextoAutorizacionAltaV3,
	string,
	error,
) {
	if c.solicitud.Validar() != nil ||
		c.contexto.ValidarPara(c.solicitud, evaluadoEn) != nil ||
		!domain.ReferenciaOpacaValida(c.organizacionRef) {
		return SolicitudResolverContextoAutorizacionAltaV3{},
			ContextoAutorizacionAltaV3{}, "",
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	contexto := c.contexto
	clon, err := contexto.Resultado.Clonar()
	if err != nil {
		return SolicitudResolverContextoAutorizacionAltaV3{},
			ContextoAutorizacionAltaV3{}, "",
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	contexto.Resultado = clon
	return c.solicitud, contexto, c.organizacionRef, nil
}

func (ContextoRecuperacionResultadoCobertura) String() string {
	return redaccionSolicitudLecturaResultadoCobertura
}
func (c ContextoRecuperacionResultadoCobertura) GoString() string {
	return c.String()
}
func (c ContextoRecuperacionResultadoCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ContextoRecuperacionResultadoCobertura) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// ResolutorContextoRecuperacionResultadoCobertura extrae la autoridad del
// contexto confiable de ejecución; no recibe DTO declarable por el cliente.
type ResolutorContextoRecuperacionResultadoCobertura interface {
	ResolverContextoRecuperacionResultadoCobertura(
		context.Context,
	) (ContextoRecuperacionResultadoCobertura, error)
}

// DatosSolicitudLecturaResultadoCobertura es la vista mínima para el PDP de
// lectura. Contexto ya fue resuelto por la frontera confiable; acción y
// finalidad nacen del constructor cerrado.
type DatosSolicitudLecturaResultadoCobertura struct {
	bloqueoSerializacionConsultaResultadoCobertura
	SolicitudContexto SolicitudResolverContextoAutorizacionAltaV3
	Contexto          ContextoAutorizacionAltaV3
	OrganizacionRef   string
	ExpedienteRef     string
	Accion            AccionLecturaResultadoCobertura
	Finalidad         FinalidadLecturaResultadoCobertura
	EvaluadaEn        time.Time
}

func (DatosSolicitudLecturaResultadoCobertura) String() string {
	return redaccionSolicitudLecturaResultadoCobertura
}
func (d DatosSolicitudLecturaResultadoCobertura) GoString() string { return d.String() }
func (d DatosSolicitudLecturaResultadoCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosSolicitudLecturaResultadoCobertura) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

// SolicitudLecturaResultadoCobertura es opaca para transportes y logs.
type SolicitudLecturaResultadoCobertura struct {
	bloqueoSerializacionConsultaResultadoCobertura
	datos *DatosSolicitudLecturaResultadoCobertura
}

func NuevaSolicitudLecturaResultadoCobertura(
	contextoRecuperacion ContextoRecuperacionResultadoCobertura,
	expedienteRef string,
	evaluadaEn time.Time,
) (SolicitudLecturaResultadoCobertura, error) {
	solicitudContexto, contexto, organizacionRef, err :=
		contextoRecuperacion.DatosEn(evaluadaEn)
	if err != nil {
		return SolicitudLecturaResultadoCobertura{},
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	solicitud := SolicitudLecturaResultadoCobertura{
		datos: &DatosSolicitudLecturaResultadoCobertura{
			SolicitudContexto: solicitudContexto,
			Contexto:          contexto,
			OrganizacionRef:   organizacionRef,
			ExpedienteRef:     expedienteRef,
			Accion:            AccionConsultarResultadoCobertura,
			Finalidad:         FinalidadRecuperarResultadoCobertura,
			EvaluadaEn:        evaluadaEn,
		},
	}
	if _, err := solicitud.Datos(); err != nil {
		return SolicitudLecturaResultadoCobertura{},
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	return solicitud, nil
}

func (s SolicitudLecturaResultadoCobertura) Datos() (
	DatosSolicitudLecturaResultadoCobertura,
	error,
) {
	if s.datos == nil ||
		s.datos.SolicitudContexto.Validar() != nil ||
		s.datos.Contexto.ValidarPara(
			s.datos.SolicitudContexto,
			s.datos.EvaluadaEn,
		) != nil ||
		!domain.ReferenciaOpacaValida(s.datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.datos.ExpedienteRef) ||
		s.datos.Accion != AccionConsultarResultadoCobertura ||
		s.datos.Finalidad != FinalidadRecuperarResultadoCobertura ||
		!domain.InstanteUTCCanonico(s.datos.EvaluadaEn) {
		return DatosSolicitudLecturaResultadoCobertura{},
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	datos := *s.datos
	resultado, err := datos.Contexto.Resultado.Clonar()
	if err != nil {
		return DatosSolicitudLecturaResultadoCobertura{},
			ErrSolicitudLecturaResultadoCoberturaInvalida
	}
	datos.Contexto.Resultado = resultado
	return datos, nil
}

func (SolicitudLecturaResultadoCobertura) String() string {
	return redaccionSolicitudLecturaResultadoCobertura
}
func (s SolicitudLecturaResultadoCobertura) GoString() string { return s.String() }
func (s SolicitudLecturaResultadoCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudLecturaResultadoCobertura) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type ResultadoAutorizacionLecturaResultadoCobertura uint8

const (
	AutorizacionLecturaResultadoCoberturaDenegada ResultadoAutorizacionLecturaResultadoCobertura = iota + 1
	AutorizacionLecturaResultadoCoberturaConcedida
)

// AutorizadorLecturaResultadoCobertura aplica lectura propia y registra su
// evidencia. No recibe ni devuelve una capacidad de efecto.
type AutorizadorLecturaResultadoCobertura interface {
	AutorizarLecturaResultadoCobertura(
		context.Context,
		SolicitudLecturaResultadoCobertura,
	) (ResultadoAutorizacionLecturaResultadoCobertura, error)
}
