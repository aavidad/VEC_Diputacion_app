// Package referencias genera referencias opacas de Selección con el
// generador criptográfico del sistema.
package referencias

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// Aleatorio crea 'sol_' seguido de 24 bytes aleatorios en base64url.
type Aleatorio struct{}

var _ ports.GeneradorReferencias = Aleatorio{}

func (Aleatorio) NuevaReferenciaSolicitud(ctx context.Context) (string, error) {
	if ctx == nil || ctx.Err() != nil {
		return "", ports.ErrNoDisponible
	}
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", ports.ErrNoDisponible
	}
	return "sol_" + base64.RawURLEncoding.EncodeToString(b[:]), nil
}
