package postgres

import (
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/resultadobolsa"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func ResultadoBolsaRRHHDesdeSQL(raw []byte) (*ports.ResultadoBolsaRRHH, error) {
	if len(raw) == 0 || len(raw) > 8*1024*1024 {
		return nil, ports.ErrResultadoBolsaRRHHNoConfiable
	}
	var salida ports.ResultadoBolsaRRHH
	if json.Unmarshal(raw, &salida) != nil {
		return nil, ports.ErrResultadoBolsaRRHHNoConfiable
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
	if resultadobolsa.ValidarResultadoBolsaRRHH(salida) != nil {
		return nil, ports.ErrResultadoBolsaRRHHNoConfiable
	}
	return &salida, nil
}
