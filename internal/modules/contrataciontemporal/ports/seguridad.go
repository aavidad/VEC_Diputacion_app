// Package ports define las fronteras hexagonales del módulo de contratación
// temporal.
package ports

import (
	"context"
	"crypto/hmac"
	"errors"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionCrearSolicitud             = "contratacion_temporal.solicitud.crear"
	FinalidadCrearSolicitud          = "tramitar_necesidad_personal_temporal"
	ModuloContratacion               = "contratacion_temporal"
	TipoRecursoExpediente            = "expediente_contratacion_temporal"
	TipoRecursoCuadroRRHH            = "cuadro_rrhh_contratacion_temporal"
	AtributoHuellaPeticionHMACActiva = "huella_peticion_hmac_activa"
)

var (
	ErrContextoAutorizacionV3Invalido = errors.New(
		"contratacion temporal: contexto de autorizacion V3 invalido",
	)
	ErrAutorizacionDenegada = errors.New(
		"contratacion temporal: autorizacion denegada",
	)
	ErrMotivoAutorizacionNoDisponible = errors.New(
		"contratacion temporal: motivo de autorizacion no disponible",
	)
)

var patronSelloHMACSHA256 = regexp.MustCompile(
	`^hmac-sha256:[a-z][a-z0-9._/-]{1,95}:[a-f0-9]{64}$`,
)

// SelloHMACSHA256Valido comprueba el sobre versionado común. La referencia
// intermedia identifica el dominio y la generación de clave; nunca es la
// propia clave.
func SelloHMACSHA256Valido(valor string) bool {
	if !patronSelloHMACSHA256.MatchString(valor) {
		return false
	}
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[2] != strings.Repeat("0", 64)
}

func sellosHMACIguales(primero, segundo string) bool {
	return SelloHMACSHA256Valido(primero) &&
		SelloHMACSHA256Valido(segundo) &&
		hmac.Equal([]byte(primero), []byte(segundo))
}

// SolicitudResolverContextoAutorizacionAltaV3 solo transporta referencias
// opacas de una autenticación ya iniciada. No admite nombres, roles, permisos,
// cuenta, método, garantía ni atributos declarados por un cliente.
type SolicitudResolverContextoAutorizacionAltaV3 struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
}

func (s SolicitudResolverContextoAutorizacionAltaV3) Validar() error {
	if (dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: s.AutenticacionRef,
		SesionRef:        s.SesionRef,
	}).Validar() != nil ||
		!referenciaVECConPrefijoValida(s.PerfilRef, "prf_") {
		return ErrContextoAutorizacionV3Invalido
	}
	return nil
}

// ContextoAutorizacionAltaV3 transporta exclusivamente capacidades comunes
// de VEC. El envoltorio no concede acceso: el vínculo solo puede nacer de las
// autoridades de autenticación y contexto, y el PDP V3 vuelve a evaluarlo.
type ContextoAutorizacionAltaV3 struct {
	Vinculo   dominiovec.VinculoAutenticacionActorV2
	Resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (c ContextoAutorizacionAltaV3) ValidarPara(
	solicitud SolicitudResolverContextoAutorizacionAltaV3,
	instante time.Time,
) error {
	if solicitud.Validar() != nil || !domain.InstanteUTCCanonico(instante) ||
		c.Resultado.Validar() != nil ||
		c.Vinculo.ValidarPara(c.Resultado) != nil ||
		!c.Vinculo.VigenteEn(instante, c.Resultado) {
		return ErrContextoAutorizacionV3Invalido
	}
	datos, err := c.Vinculo.Datos()
	if err != nil || datos.AutenticacionRef != solicitud.AutenticacionRef ||
		datos.SesionRef != solicitud.SesionRef ||
		datos.PerfilActivoRef != solicitud.PerfilRef {
		return ErrContextoAutorizacionV3Invalido
	}
	return nil
}

// ResolutorContextoAutorizacionAltaV3 es un adaptador fino sobre
// CrearVinculoAutenticacionActorV2. Web, escritorio, CLI y MCP entregan las
// mismas referencias y no pueden inyectar la identidad resultante.
type ResolutorContextoAutorizacionAltaV3 interface {
	ResolverContextoAutorizacionAltaV3(
		context.Context,
		SolicitudResolverContextoAutorizacionAltaV3,
	) (ContextoAutorizacionAltaV3, error)
}

// SolicitudResolverMotivoAutorizacionAltaV3 permite que una entrada
// administrable resuelva la versión publicada exacta del catálogo. El texto
// funcional recibido nunca se usa directamente como clave de autorización.
type SolicitudResolverMotivoAutorizacionAltaV3 struct {
	OrganizacionRef string
	Flujo           domain.ReferenciaFlujo
	MotivoClave     domain.ClaveCatalogo
	Instante        time.Time
}

func (s SolicitudResolverMotivoAutorizacionAltaV3) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		s.Flujo.Validar() != nil || !s.MotivoClave.Valida() ||
		!domain.InstanteUTCCanonico(s.Instante) {
		return ErrMotivoAutorizacionNoDisponible
	}
	return nil
}

type ResolutorMotivoAutorizacionAltaV3 interface {
	ResolverMotivoAutorizacionAltaV3(
		context.Context,
		SolicitudResolverMotivoAutorizacionAltaV3,
	) (dominiovec.ReferenciaEntradaCatalogo, error)
}

func referenciaVECConPrefijoValida(valor, prefijo string) bool {
	const (
		longitudMinimaTokenVEC = 22
		longitudMaximaTokenVEC = 128
	)
	if !strings.HasPrefix(valor, prefijo) ||
		len(valor) < len(prefijo)+longitudMinimaTokenVEC ||
		len(valor) > len(prefijo)+longitudMaximaTokenVEC {
		return false
	}
	for _, caracter := range valor[len(prefijo):] {
		if (caracter < 'a' || caracter > 'z') &&
			(caracter < 'A' || caracter > 'Z') &&
			(caracter < '0' || caracter > '9') &&
			caracter != '_' && caracter != '-' {
			return false
		}
	}
	return true
}
