package seguridad

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AutenticadorFuentesAnalisisConConfianza implementa la verificación
// criptográfica local en infraestructura. La aplicación crea el desafío,
// obtiene la presentación y aporta el instante autoritativo posterior.
type AutenticadorFuentesAnalisisConConfianza struct {
	confianza ports.ConfianzaAutoridadesFuenteAnalisis
}

func NuevoAutenticadorFuentesAnalisisConConfianza(
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
) (*AutenticadorFuentesAnalisisConConfianza, error) {
	if !domain.ReferenciaOpacaValida(confianza.OrganizacionRef()) ||
		!domain.ReferenciaOpacaValida(confianza.Audiencia()) {
		return nil, ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return &AutenticadorFuentesAnalisisConConfianza{
		confianza: confianza,
	}, nil
}

func (a *AutenticadorFuentesAnalisisConConfianza) OrganizacionAutoridadFuenteAnalisis() string {
	if a == nil {
		return ""
	}
	return a.confianza.OrganizacionRef()
}

func (a *AutenticadorFuentesAnalisisConConfianza) AudienciaAutoridadFuenteAnalisis() string {
	if a == nil {
		return ""
	}
	return a.confianza.Audiencia()
}

func (a *AutenticadorFuentesAnalisisConConfianza) VerificarPresentacionAutoridadFuenteAnalisis(
	presentacion ports.PresentacionAutoridadFuenteAnalisis,
	desafio ports.DesafioAutoridadFuenteAnalisis,
	rol ports.RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	if a == nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return a.confianza.VerificarPresentacion(
		presentacion,
		desafio,
		rol,
		comprobadaEn,
	)
}

var _ ports.VerificadorPresentacionesAutoridadFuenteAnalisis = (*AutenticadorFuentesAnalisisConConfianza)(nil)
