package resultadobolsa

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func ValidarResultadoBolsaRRHH(r ports.ResultadoBolsaRRHH) error {
	if r.Vinculos == nil || r.EmisionesVinculables == nil ||
		len(r.Vinculos) > 20 || uint64(len(r.Vinculos)) > r.TotalVinculos ||
		len(r.EmisionesVinculables) > 20 ||
		(r.SiguienteCursor != nil && (len(r.Vinculos) != 20 || !CursorResultadoBolsaRRHHValido(*r.SiguienteCursor))) ||
		(r.PersonasSolicitadas != nil && *r.PersonasSolicitadas == 0) {
		return ports.ErrResultadoBolsaRRHHNoConfiable
	}
	if r.SiguienteCursor != nil {
		ultimo := r.Vinculos[len(r.Vinculos)-1]
		esperado := ultimo.VinculadoEn.UTC().Format("2006-01-02T15:04:05.000000Z") + "#" + ultimo.LlamamientoRef
		if *r.SiguienteCursor != esperado {
			return ports.ErrResultadoBolsaRRHHNoConfiable
		}
	}
	vistos := make(map[string]struct{}, len(r.Vinculos))
	for _, v := range r.Vinculos {
		if !domain.ReferenciaOpacaValida(v.BolsaRef) || !domain.LlamamientoEmisionBolsaValido(v.LlamamientoRef) ||
			v.ReciboEmisionRef != "recibo:"+v.LlamamientoRef || !domain.ReferenciaOpacaValida(v.ReciboVinculoRef) ||
			!domain.InstanteUTCCanonico(v.VinculadoEn) || !domain.InstanteUTCCanonico(v.EmitidoEn) ||
			v.VinculadoEn.Before(v.EmitidoEn) ||
			len(v.Participaciones) == 0 || len(v.Participaciones) > 100 {
			return ports.ErrResultadoBolsaRRHHNoConfiable
		}
		if _, repetido := vistos[v.LlamamientoRef]; repetido {
			return ports.ErrResultadoBolsaRRHHNoConfiable
		}
		vistos[v.LlamamientoRef] = struct{}{}
		participaciones := make(map[string]struct{}, len(v.Participaciones))
		for _, p := range v.Participaciones {
			if !domain.ReferenciaOpacaValida(p.ParticipacionRef) {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
			if _, repetida := participaciones[p.ParticipacionRef]; repetida {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
			participaciones[p.ParticipacionRef] = struct{}{}
			if p.Respuesta != nil {
				if (*p.Respuesta != "acepta" && *p.Respuesta != "renuncia" && *p.Respuesta != "renuncia_justificada") ||
					p.Modo == nil || (*p.Modo != "firme" && *p.Modo != "propuesta_rrhh") ||
					p.ReciboRespuestaRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboRespuestaRef) ||
					p.RespondidaEn == nil || !domain.InstanteUTCCanonico(*p.RespondidaEn) {
					return ports.ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.Modo != nil || p.ReciboRespuestaRef != nil || p.RespondidaEn != nil || p.JustificanteRef != nil {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
			if p.JustificanteRef != nil && !domain.ReferenciaOpacaValida(*p.JustificanteRef) {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
			if p.ContactoResultado != nil {
				if p.ReciboContactoRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboContactoRef) ||
					p.ContactoEn == nil || !domain.InstanteUTCCanonico(*p.ContactoEn) {
					return ports.ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.ReciboContactoRef != nil || p.ContactoEn != nil {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
			if p.SituacionActual != nil {
				if p.ReciboSituacionRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboSituacionRef) ||
					p.SituacionDesde == nil || !domain.InstanteUTCCanonico(*p.SituacionDesde) {
					return ports.ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.ReciboSituacionRef != nil || p.SituacionDesde != nil {
				return ports.ErrResultadoBolsaRRHHNoConfiable
			}
		}
	}
	for _, e := range r.EmisionesVinculables {
		if !domain.ReferenciaOpacaValida(e.BolsaRef) || !domain.LlamamientoEmisionBolsaValido(e.LlamamientoRef) ||
			e.ReciboEmisionRef != "recibo:"+e.LlamamientoRef || e.ReferenciaVisible == "" ||
			!domain.InstanteUTCCanonico(e.EmitidoEn) {
			return ports.ErrResultadoBolsaRRHHNoConfiable
		}
	}
	return nil
}
