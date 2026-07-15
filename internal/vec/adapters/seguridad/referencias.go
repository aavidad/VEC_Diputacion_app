package seguridad

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"vec-diputacion-granada/internal/vec/ports"
)

// GeneradorReferenciasCriptograficas crea identificadores opacos sin incluir
// DNI, nombre, correo ni ninguna clave de negocio.
type GeneradorReferenciasCriptograficas struct{}

func (GeneradorReferenciasCriptograficas) NuevaReferenciaDecisionAutorizacion() (string, error) {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", fmt.Errorf("crear referencia de decision: %w", err)
	}
	return "decision:" + hex.EncodeToString(aleatorio), nil
}

var _ ports.GeneradorReferenciaDecisionAutorizacion = GeneradorReferenciasCriptograficas{}
