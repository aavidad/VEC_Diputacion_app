package domain

import (
	"encoding/hex"
	"strings"
)

// AcuseConsumoDenominacionPersona transporta el eco del consumo común
// confirmado. Su estructura no acredita COMMIT ni firma: sólo la fuente
// autorizada puede comprobarlo contra el material original y su registro.
type AcuseConsumoDenominacionPersona struct {
	datos DatosAcuseConsumoDenominacionPersona
}

type DatosAcuseConsumoDenominacionPersona struct {
	ConsumoRef, AuditoriaRef, DecisionRef, CorrelacionRef, RecursoRef string
	PersonaRef, SobreSHA256                                           string
	Version                                                           uint64
}

func NuevoAcuseConsumoDenominacionPersona(d DatosAcuseConsumoDenominacionPersona) (AcuseConsumoDenominacionPersona, error) {
	if !ReferenciaPersonaDenominacionValida(d.PersonaRef) || d.Version == 0 || d.Version > 1<<53-1 || len(d.SobreSHA256) != 64 || strings.ToLower(d.SobreSHA256) != d.SobreSHA256 {
		return AcuseConsumoDenominacionPersona{}, ErrDenominacionPersonaInvalida
	}
	if _, err := hex.DecodeString(d.SobreSHA256); err != nil {
		return AcuseConsumoDenominacionPersona{}, ErrDenominacionPersonaInvalida
	}
	for _, ref := range []string{d.ConsumoRef, d.AuditoriaRef, d.DecisionRef, d.CorrelacionRef, d.RecursoRef} {
		if len(ref) < 8 || len(ref) > 128 {
			return AcuseConsumoDenominacionPersona{}, ErrDenominacionPersonaInvalida
		}
		for _, r := range ref {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
				return AcuseConsumoDenominacionPersona{}, ErrDenominacionPersonaInvalida
			}
		}
	}
	return AcuseConsumoDenominacionPersona{datos: d}, nil
}

func (a AcuseConsumoDenominacionPersona) Datos() (DatosAcuseConsumoDenominacionPersona, error) {
	if _, err := NuevoAcuseConsumoDenominacionPersona(a.datos); err != nil {
		return DatosAcuseConsumoDenominacionPersona{}, err
	}
	return a.datos, nil
}
