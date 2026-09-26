package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// MaximoPropuestasExpediente acota la historia de propuestas que se publica.
const MaximoPropuestasExpediente = 50

// EstadoPropuestaExpediente es una propuesta de nombramiento del expediente
// (CT128): la vigente o una anterior sustituida por su no incorporación.
// Solo referencias opacas y claves del catálogo; nunca datos de la persona.
type EstadoPropuestaExpediente struct {
	Orden             uint32
	VersionResultante uint64
	ConfirmadaEn      time.Time
	ReciboRef         string
	Vigente           bool
	Sustitucion       *SustitucionPropuestaExpediente
}

// SustitucionPropuestaExpediente: la no incorporación que dejó sin efecto la
// propuesta.
type SustitucionPropuestaExpediente struct {
	NoIncorporacionReciboRef string
	MotivoClave              string
	RegistradaEn             time.Time
}

// PropuestasExpedienteValidas: órdenes 1..n, versiones crecientes, solo la
// última vigente y cada anterior con su sustitución.
func PropuestasExpedienteValidas(p []EstadoPropuestaExpediente) bool {
	if len(p) > MaximoPropuestasExpediente {
		return false
	}
	for i, e := range p {
		ultima := i == len(p)-1
		if e.Orden != uint32(i+1) || e.VersionResultante == 0 || e.VersionResultante > MaximoEnteroSeguroIntegracionBolsa ||
			(i > 0 && e.VersionResultante <= p[i-1].VersionResultante) ||
			!domain.InstanteUTCCanonico(e.ConfirmadaEn) || !domain.ReferenciaOpacaValida(e.ReciboRef) ||
			e.Vigente != ultima || (e.Sustitucion == nil) != ultima {
			return false
		}
		if s := e.Sustitucion; s != nil && (!domain.ReferenciaOpacaValida(s.NoIncorporacionReciboRef) ||
			!domain.ClaveCatalogo(s.MotivoClave).Valida() || !domain.InstanteUTCCanonico(s.RegistradaEn)) {
			return false
		}
	}
	return true
}
