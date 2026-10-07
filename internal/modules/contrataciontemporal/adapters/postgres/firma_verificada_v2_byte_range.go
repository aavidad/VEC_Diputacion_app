package postgres

import (
	"encoding/json"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// encoding/json convierte null a cero al decodificar uint64. El contrato SQL
// exige cuatro enteros explícitos, incluido el cero inicial del ByteRange.
type byteRangeFirmaSQL172 []uint64

func (b *byteRangeFirmaSQL172) UnmarshalJSON(datos []byte) error {
	var tokens []json.RawMessage
	if json.Unmarshal(datos, &tokens) != nil || len(tokens) != 4 {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	valores := make(byteRangeFirmaSQL172, 4)
	for i, token := range tokens {
		if len(token) == 0 {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		for _, c := range token {
			if c < '0' || c > '9' {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
		}
		if json.Unmarshal(token, &valores[i]) != nil {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	*b = valores
	return nil
}
