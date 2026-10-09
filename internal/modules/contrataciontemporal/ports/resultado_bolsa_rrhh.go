package ports

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrResultadoBolsaRRHHNoConfiable = errors.New("contratacion temporal: resultado Bolsa no confiable")
var patronLlamamientoRealBolsaCT = regexp.MustCompile(`^llamamiento:[0-9a-f]{64}$`)

// Respuesta y contacto están ligados al llamamiento y a la participación.
// SituacionActual es global de Bolsa y nunca equivale a una respuesta firme.
type ParticipacionResultadoBolsaRRHH struct {
	ParticipacionRef   string     `json:"participacion_ref"`
	Respuesta          *string    `json:"respuesta"`
	Modo               *string    `json:"modo"`
	ReciboRespuestaRef *string    `json:"recibo_respuesta_ref"`
	RespondidaEn       *time.Time `json:"respondida_en"`
	JustificanteRef    *string    `json:"justificante_ref"`
	ContactoResultado  *string    `json:"contacto_resultado"`
	ReciboContactoRef  *string    `json:"recibo_contacto_ref"`
	ContactoEn         *time.Time `json:"contacto_en"`
	SituacionActual    *string    `json:"situacion_actual"`
	ReciboSituacionRef *string    `json:"recibo_situacion_ref"`
	SituacionDesde     *time.Time `json:"situacion_desde"`
}

type VinculoResultadoBolsaRRHH struct {
	BolsaRef         string                            `json:"bolsa_ref"`
	LlamamientoRef   string                            `json:"llamamiento_ref"`
	ReciboEmisionRef string                            `json:"recibo_emision_ref"`
	ReciboVinculoRef string                            `json:"recibo_vinculo_ref"`
	VinculadoEn      time.Time                         `json:"vinculado_en"`
	EmitidoEn        time.Time                         `json:"emitido_en"`
	Participaciones  []ParticipacionResultadoBolsaRRHH `json:"participaciones"`
}

type EmisionVinculableBolsaRRHH struct {
	BolsaRef          string    `json:"bolsa_ref"`
	LlamamientoRef    string    `json:"llamamiento_ref"`
	ReciboEmisionRef  string    `json:"recibo_emision_ref"`
	ReferenciaVisible string    `json:"referencia_visible"`
	EmitidoEn         time.Time `json:"emitido_en"`
}

type ResultadoBolsaRRHH struct {
	Vinculos             []VinculoResultadoBolsaRRHH  `json:"vinculos"`
	EmisionesVinculables []EmisionVinculableBolsaRRHH `json:"emisiones_vinculables"`
	SiguienteCursor      *string                      `json:"siguiente_cursor"`
}

func ResultadoBolsaRRHHDesdeSQL(raw []byte) (*ResultadoBolsaRRHH, error) {
	if len(raw) == 0 || len(raw) > 512*1024 {
		return nil, ErrResultadoBolsaRRHHNoConfiable
	}
	var salida ResultadoBolsaRRHH
	if json.Unmarshal(raw, &salida) != nil {
		return nil, ErrResultadoBolsaRRHHNoConfiable
	}
	for i := range salida.Vinculos {
		v := &salida.Vinculos[i]
		v.VinculadoEn, v.EmitidoEn = v.VinculadoEn.UTC(), v.EmitidoEn.UTC()
		for j := range v.Participaciones {
			p := &v.Participaciones[j]
			for _, instante := range []*time.Time{p.RespondidaEn, p.ContactoEn, p.SituacionDesde} {
				if instante != nil {
					*instante = instante.UTC()
				}
			}
		}
	}
	for i := range salida.EmisionesVinculables {
		salida.EmisionesVinculables[i].EmitidoEn = salida.EmisionesVinculables[i].EmitidoEn.UTC()
	}
	if salida.Validar() != nil {
		return nil, ErrResultadoBolsaRRHHNoConfiable
	}
	return &salida, nil
}

func (r ResultadoBolsaRRHH) Validar() error {
	if r.Vinculos == nil || r.EmisionesVinculables == nil ||
		len(r.Vinculos) > 100 || len(r.EmisionesVinculables) > 20 || r.SiguienteCursor != nil {
		return ErrResultadoBolsaRRHHNoConfiable
	}
	vistos := make(map[string]struct{}, len(r.Vinculos))
	for _, v := range r.Vinculos {
		if !domain.ReferenciaOpacaValida(v.BolsaRef) || !patronLlamamientoRealBolsaCT.MatchString(v.LlamamientoRef) ||
			v.ReciboEmisionRef != "recibo:"+v.LlamamientoRef || !domain.ReferenciaOpacaValida(v.ReciboVinculoRef) ||
			!domain.InstanteUTCCanonico(v.VinculadoEn) || !domain.InstanteUTCCanonico(v.EmitidoEn) ||
			v.VinculadoEn.Before(v.EmitidoEn) ||
			len(v.Participaciones) == 0 || len(v.Participaciones) > 100 {
			return ErrResultadoBolsaRRHHNoConfiable
		}
		if _, repetido := vistos[v.LlamamientoRef]; repetido {
			return ErrResultadoBolsaRRHHNoConfiable
		}
		vistos[v.LlamamientoRef] = struct{}{}
		participaciones := make(map[string]struct{}, len(v.Participaciones))
		for _, p := range v.Participaciones {
			if !domain.ReferenciaOpacaValida(p.ParticipacionRef) {
				return ErrResultadoBolsaRRHHNoConfiable
			}
			if _, repetida := participaciones[p.ParticipacionRef]; repetida {
				return ErrResultadoBolsaRRHHNoConfiable
			}
			participaciones[p.ParticipacionRef] = struct{}{}
			if p.Respuesta != nil {
				if (*p.Respuesta != "acepta" && *p.Respuesta != "renuncia" && *p.Respuesta != "renuncia_justificada") ||
					p.Modo == nil || (*p.Modo != "firme" && *p.Modo != "propuesta_rrhh") ||
					p.ReciboRespuestaRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboRespuestaRef) ||
					p.RespondidaEn == nil || !domain.InstanteUTCCanonico(*p.RespondidaEn) {
					return ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.Modo != nil || p.ReciboRespuestaRef != nil || p.RespondidaEn != nil || p.JustificanteRef != nil {
				return ErrResultadoBolsaRRHHNoConfiable
			}
			if p.JustificanteRef != nil && !domain.ReferenciaOpacaValida(*p.JustificanteRef) {
				return ErrResultadoBolsaRRHHNoConfiable
			}
			if p.ContactoResultado != nil {
				if p.ReciboContactoRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboContactoRef) ||
					p.ContactoEn == nil || !domain.InstanteUTCCanonico(*p.ContactoEn) {
					return ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.ReciboContactoRef != nil || p.ContactoEn != nil {
				return ErrResultadoBolsaRRHHNoConfiable
			}
			if p.SituacionActual != nil {
				if p.ReciboSituacionRef == nil || !domain.ReferenciaOpacaValida(*p.ReciboSituacionRef) ||
					p.SituacionDesde == nil || !domain.InstanteUTCCanonico(*p.SituacionDesde) {
					return ErrResultadoBolsaRRHHNoConfiable
				}
			} else if p.ReciboSituacionRef != nil || p.SituacionDesde != nil {
				return ErrResultadoBolsaRRHHNoConfiable
			}
		}
	}
	for _, e := range r.EmisionesVinculables {
		if !domain.ReferenciaOpacaValida(e.BolsaRef) || !patronLlamamientoRealBolsaCT.MatchString(e.LlamamientoRef) ||
			e.ReciboEmisionRef != "recibo:"+e.LlamamientoRef || e.ReferenciaVisible == "" ||
			!domain.InstanteUTCCanonico(e.EmitidoEn) {
			return ErrResultadoBolsaRRHHNoConfiable
		}
	}
	return nil
}
