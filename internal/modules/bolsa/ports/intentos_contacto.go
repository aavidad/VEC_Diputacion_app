package ports

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// ReglaIntentosContacto es una regla del catálogo que la ficha muestra junto
// al control de intentos, con su referencia exacta catálogo:versión:entrada.
type ReglaIntentosContacto struct {
	Clave, Etiqueta, Descripcion, Referencia string
	Ejemplo                                  bool
}

// PoliticaIntentosContacto traduce el catálogo de reglas de Bolsa. Sin
// catálogo responde configurada=false y el registro de contactos conserva su
// conducta sin control; un catálogo ilegible es un error, nunca una regla
// por defecto.
type PoliticaIntentosContacto interface {
	PoliticaIntentosTelefonicos(context.Context) (dominiobolsa.PoliticaIntentosTelefonicos, []ReglaIntentosContacto, bool, error)
}

// CalendarioDiasHabiles dice si la fecha civil del instante es hábil en la
// sede. Una indisponibilidad nunca se interpreta como día hábil.
type CalendarioDiasHabiles interface {
	EsDiaHabil(context.Context, time.Time) (bool, error)
}

// EstadoIntentosContacto es la lectura del control de intentos de un
// llamamiento que se entrega a la ficha del candidato.
type EstadoIntentosContacto struct {
	Configurada    bool
	LlamamientoRef string
	Estado         dominiobolsa.EstadoIntentosTelefonicos
	Politica       dominiobolsa.PoliticaIntentosTelefonicos
	Reglas         []ReglaIntentosContacto
	// Completo es falso si el histórico leído no alcanzó el principio.
	Completo bool
	Ahora    time.Time
}
