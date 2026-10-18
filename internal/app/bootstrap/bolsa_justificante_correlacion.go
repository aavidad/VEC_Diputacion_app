package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// La clave de recuperación pertenece a persona, bolsa y petición. No depende
// de que el llamamiento siga abierto después de registrar la respuesta.
type correlacionJustificanteBolsa struct {
	base  docports.DerivadorCorrelacionSubidaLocal
	bolsa string
}

func (d correlacionJustificanteBolsa) CorrelacionSubida(ctx context.Context, persona, clave string) (string, error) {
	if d.base == nil || d.bolsa == "" || clave == "" || len(clave) > 256 {
		return "", docports.ErrSubidaLocalDenegada
	}
	canon, err := json.Marshal([]string{"vec.bolsa.justificante-renuncia.clave.v1", d.bolsa, clave})
	if err != nil {
		return "", docports.ErrSubidaLocalNoDisponible
	}
	suma := sha256.Sum256(canon)
	return d.base.CorrelacionSubida(ctx, persona, hex.EncodeToString(suma[:]))
}
