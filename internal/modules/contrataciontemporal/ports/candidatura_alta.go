package ports

import (
	"context"
	"crypto/hmac"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// SolicitudResolverCandidaturaAlta transporta exclusivamente sellos y
// referencias opacas. La candidatura propuesta no reserva una plaza, no crea
// expediente y no concede autoridad.
type SolicitudResolverCandidaturaAlta struct {
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasPeticionHMAC     ColeccionSellosHMAC
	OrganizacionRef         string
	ActorRef                string
	PerfilRef               string
	Propuesta               CandidaturaAlta
}

func (s SolicitudResolverCandidaturaAlta) Validar() error {
	datosAmbitos, datosHuellas, validas :=
		datosColeccionesHMACAltaAlineadas(
			s.AmbitosIdempotenciaHMAC,
			s.HuellasPeticionHMAC,
		)
	if !validas ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		s.Propuesta.Validar() != nil ||
		s.Propuesta.OrganizacionRef != s.OrganizacionRef ||
		s.Propuesta.ActorRef != s.ActorRef ||
		s.Propuesta.PerfilRef != s.PerfilRef ||
		!hmac.Equal(
			[]byte(s.Propuesta.AmbitoIdempotenciaHMAC),
			[]byte(datosAmbitos.Activo.Valor),
		) ||
		!hmac.Equal(
			[]byte(s.Propuesta.HuellaPeticionHMAC),
			[]byte(datosHuellas.Activo.Valor),
		) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// ValidarResultado admite el par activo o uno retenido de la misma
// generación. Esto permite recuperar tras rotación exactamente la
// candidatura técnica que se acuñó en el primer intento.
func (s SolicitudResolverCandidaturaAlta) ValidarResultado(
	candidatura CandidaturaAlta,
) error {
	if s.Validar() != nil ||
		candidatura.Validar() != nil ||
		candidatura.OrganizacionRef != s.OrganizacionRef ||
		candidatura.ActorRef != s.ActorRef ||
		candidatura.PerfilRef != s.PerfilRef ||
		!ColeccionesHMACAltaContienenPar(
			s.AmbitosIdempotenciaHMAC,
			s.HuellasPeticionHMAC,
			candidatura.AmbitoIdempotenciaHMAC,
			candidatura.HuellaPeticionHMAC,
		) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// ResolutorCandidaturaAlta estabiliza referencias antes de proyectar y
// autorizar el efecto. Su resultado es técnico y no autoritativo; la única
// creación administrativa sigue ocurriendo en TransaccionAltas.
type ResolutorCandidaturaAlta interface {
	ResolverCandidaturaAlta(
		context.Context,
		SolicitudResolverCandidaturaAlta,
	) (CandidaturaAlta, error)
}
