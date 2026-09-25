package ports

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// ErrPoliticaTransicionesNoInstalada: la base de datos no tiene todavía la
// migración 000032. Rige la tabla compilada, que es la que aplica esa base.
var ErrPoliticaTransicionesNoInstalada = errors.New("bolsa: politica de transiciones no instalada")

// PoliticaTransicionesVigente es la política que la base aplicará al próximo
// cambio de situación. Version 0 significa que la base no publica política y
// rige la compilada.
type PoliticaTransicionesVigente struct {
	Version     int64
	CatalogoRef string
	Politica    dominiobolsa.PoliticaTransicionesSituacion
}

// PublicacionPoliticaTransicionesSituacion traslada a la base la política que
// resulta del catálogo de reglas. La referencia y la huella identifican la
// versión exacta del catálogo.
type PublicacionPoliticaTransicionesSituacion struct {
	CatalogoRef    string
	CatalogoSHA256 string
	Politica       dominiobolsa.PoliticaTransicionesSituacion
}

// ConsultaPoliticaTransicionesSituacion lee la política vigente. Un fallo de
// lectura nunca se interpreta como una política más laxa.
type ConsultaPoliticaTransicionesSituacion interface {
	PoliticaTransicionesSituacion(ctx context.Context) (PoliticaTransicionesVigente, error)
}

// PublicadorPoliticaTransicionesSituacion publica una nueva versión si
// difiere de la vigente. La base vuelve a exigir las invariantes fijas.
type PublicadorPoliticaTransicionesSituacion interface {
	PublicarPoliticaTransicionesSituacion(ctx context.Context, p PublicacionPoliticaTransicionesSituacion) (PoliticaTransicionesVigente, error)
}
