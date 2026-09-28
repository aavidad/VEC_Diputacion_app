package postgres

import "vec-diputacion-granada/internal/modules/bolsa/ports"

// La etiqueta es una proyección revocable. El selector inmutable conserva
// código, versión y huella aunque esa etiqueta deje de ser publicable.
func selectorCausaParticipacionRecuperada(codigo *string, version *int64, huella, etiqueta *string, motivo string) (*ports.SelectorCausaParticipacion, bool) {
	if codigo == nil && version == nil && huella == nil && etiqueta == nil {
		return nil, true
	}
	if codigo == nil || version == nil || huella == nil {
		return nil, false
	}
	selector := ports.SelectorCausaParticipacion{Codigo: *codigo, Version: *version, HuellaSHA256: *huella}
	if selector.Validar() != nil || selector.Codigo != motivo {
		return nil, false
	}
	return &selector, true
}
