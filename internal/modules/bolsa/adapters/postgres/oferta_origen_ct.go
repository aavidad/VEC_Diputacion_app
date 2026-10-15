package postgres

import (
	"encoding/json"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// El corte que se incluye en el canon escalar firmado conserva exactamente
// los seis decimales pactados con PostgreSQL; no se normaliza a milisegundos.
func serializarOrigenOfertaCT(o *ports.OrigenOfertaCT) (any, error) {
	if o == nil {
		return nil, nil
	}
	type lecturaSQL struct {
		ports.OrigenLlamamientoCT
		Corte string `json:"corte"`
	}
	type origenSQL struct {
		ports.OrigenOfertaCT
		Lectura lecturaSQL `json:"lectura"`
	}
	return json.Marshal(origenSQL{OrigenOfertaCT: *o, Lectura: lecturaSQL{OrigenLlamamientoCT: o.Lectura, Corte: o.CorteCanonico()}})
}
