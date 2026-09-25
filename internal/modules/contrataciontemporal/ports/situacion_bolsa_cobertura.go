package ports

import (
	"context"
	"errors"
	"time"
)

// ErrSituacionBolsaCoberturaNoDisponible indica que Bolsa no ha podido
// responder. Nunca se interpreta como bolsa vigente, agotada ni sin bolsa.
var ErrSituacionBolsaCoberturaNoDisponible = errors.New(
	"contratacion temporal: situacion de bolsa para cobertura no disponible",
)

// SituacionBolsaCobertura es el resumen mínimo que Bolsa entrega a
// Contratación temporal para evaluar la vía de cobertura. Solo contiene
// recuentos y la fecha de constitución: ninguna persona ni posición.
type SituacionBolsaCobertura struct {
	// Existe es falso si no hay bolsa constituida y vigente para la categoría.
	Existe bool
	// BolsaRef es la referencia opaca de la bolsa más reciente de la categoría.
	BolsaRef string
	// ConstituidaEn es el instante de constitución de esa bolsa.
	ConstituidaEn time.Time
	// Integrantes cuenta a todas las personas de las bolsas de la categoría.
	Integrantes int
	// Disponibles cuenta a quienes pueden ser llamados hoy (art. 9 del
	// Reglamento de bolsas): situación «disponible» o disponibilidad ya
	// alcanzada.
	Disponibles int
}

// Validar comprueba la coherencia del resumen antes de evaluarlo.
func (s SituacionBolsaCobertura) Validar() error {
	if !s.Existe {
		if s.BolsaRef != "" || !s.ConstituidaEn.IsZero() || s.Integrantes != 0 || s.Disponibles != 0 {
			return ErrSituacionBolsaCoberturaNoDisponible
		}
		return nil
	}
	if s.BolsaRef == "" || len(s.BolsaRef) > 200 || s.ConstituidaEn.IsZero() ||
		s.Integrantes < 0 || s.Disponibles < 0 || s.Disponibles > s.Integrantes {
		return ErrSituacionBolsaCoberturaNoDisponible
	}
	return nil
}

// ConsultaSituacionBolsaCobertura es el puerto hacia Bolsa. La composición lo
// implementa con las lecturas propias de Bolsa; Contratación temporal no lee
// sus tablas.
type ConsultaSituacionBolsaCobertura interface {
	SituacionBolsaCobertura(ctx context.Context, categoriaRef string) (SituacionBolsaCobertura, error)
}
