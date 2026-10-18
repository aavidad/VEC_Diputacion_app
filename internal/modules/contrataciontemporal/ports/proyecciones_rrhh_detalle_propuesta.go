package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// PropuestaFormalizacionRRHH muestra el recibo confirmado de la propuesta del
// hijo. La autoridad y el recibo se cotejan en la lectura nominal de SQL.
type PropuestaFormalizacionRRHH struct {
	PropuestaRef string    `json:"propuesta_ref"`
	ReciboRef    string    `json:"recibo_ref"`
	Version      uint64    `json:"version"`
	ConfirmadaEn time.Time `json:"confirmada_en"`
}

func (p PropuestaFormalizacionRRHH) validarPara(d DetalleExpedienteRRHH) error {
	if d.InstruccionCompartida == nil || d.Resumen.Version < 7 || len(d.Hitos) < 2 ||
		!domain.ReferenciaOpacaValida(p.PropuestaRef) ||
		!domain.ReferenciaOpacaValida(p.ReciboRef) || p.PropuestaRef == p.ReciboRef ||
		p.Version != 7 || !domain.InstanteUTCCanonico(p.ConfirmadaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	h := d.Hitos[1]
	if h.Secuencia != 7 || h.VersionExpediente != 7 ||
		h.AccionClave != "registrar_propuesta_formalizacion" ||
		h.FaseOrigen != domain.FaseFiscalizacion || h.EstadoOrigen != domain.EstadoEnCurso ||
		h.FaseDestino != domain.FaseNombramiento || h.EstadoDestino != domain.EstadoEnCurso ||
		!h.RealizadaEn.Equal(p.ConfirmadaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func clonarPropuestaFormalizacionRRHH(p *PropuestaFormalizacionRRHH) *PropuestaFormalizacionRRHH {
	if p == nil {
		return nil
	}
	clon := *p
	return &clon
}
