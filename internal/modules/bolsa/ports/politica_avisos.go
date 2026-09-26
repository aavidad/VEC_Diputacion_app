package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

var ErrPoliticaAvisosNoDisponible = errors.New("bolsa: politica de avisos no disponible")

// PublicacionPoliticaAvisos traslada a la base los parámetros del catálogo
// de reglas (b16, b17 y b19) con la referencia y la huella de su versión.
type PublicacionPoliticaAvisos struct {
	CatalogoRef    string
	CatalogoSHA256 string
	Politica       dominiobolsa.PoliticaAvisosBolsa
}

// PublicadorPoliticaAvisos registra una versión nueva si difiere de la
// vigente y devuelve la versión que rige.
type PublicadorPoliticaAvisos interface {
	PublicarPoliticaAvisos(context.Context, PublicacionPoliticaAvisos) (int64, error)
}

// ConsultaMarcasParticipaciones devuelve, por participación, las marcas de
// una bolsa en un instante. Solo aparecen las participaciones con alguna.
type ConsultaMarcasParticipaciones interface {
	MarcasParticipaciones(ctx context.Context, bolsaRef string, corte time.Time) (map[string]dominiobolsa.MarcasParticipacion, error)
}
