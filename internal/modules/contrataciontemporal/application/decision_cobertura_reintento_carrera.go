package application

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"time"
)

// Política de repetición de una confirmación de decisión de cobertura que la
// base revirtió por una carrera de serialización. Es la misma forma que la
// política común de VEC (espera exponencial con tope y aleatoria dentro del
// tramo), sin depender del adaptador de base de datos.
const (
	intentosMaximosCarreraConfirmacionCobertura = 30
	esperaInicialCarreraConfirmacionCobertura   = 4 * time.Millisecond
	esperaMaximaCarreraConfirmacionCobertura    = 200 * time.Millisecond
)

// esperarReintentoCarreraConfirmacionCobertura espera antes del intento n+1 y
// devuelve false si el contexto vence o se cancela mientras tanto.
func esperarReintentoCarreraConfirmacionCobertura(ctx context.Context, n int) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	tope := esperaMaximaCarreraConfirmacionCobertura
	if n < 16 {
		if exponencial := esperaInicialCarreraConfirmacionCobertura << n; exponencial < tope {
			tope = exponencial
		}
	}
	// La aleatoriedad solo reparte los reintentos; si falla, se espera el tope.
	espera := tope
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		espera = time.Duration(binary.LittleEndian.Uint64(b[:]) % uint64(tope))
	}
	t := time.NewTimer(time.Millisecond + espera)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
