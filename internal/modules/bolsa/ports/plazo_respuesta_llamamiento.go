package ports

import (
	"errors"
	"time"
)

// EsquemaPlazoRespuestaLlamamiento identifica el contrato de la consulta de
// solo lectura que propone al asistente B7 el plazo de respuesta.
const EsquemaPlazoRespuestaLlamamiento = "vec.bolsa.llamamiento.plazo_respuesta.v1"

// ErrPlazoRespuestaNoDisponible indica que hay catálogo de reglas pero la
// regla o su cálculo no están disponibles. Nunca se sustituye por un plazo
// supuesto: el asistente conserva el texto libre.
var ErrPlazoRespuestaNoDisponible = errors.New("bolsa: plazo de respuesta no disponible")

// ReglaPlazoRespuesta es la regla del catálogo, tal como la publica. Texto es
// la descripción de la entrada; Referencia es catalogo:version:entrada.
type ReglaPlazoRespuesta struct {
	Etiqueta   string
	Texto      string
	Referencia string
	// Origen es «reglamento» o «ejemplo»; Articulo solo existe en el primero.
	Origen   string
	Articulo string
	// Ejemplo es cierto si la regla, o parte de ella, no procede del
	// Reglamento y debe rotularse como regla de ejemplo.
	Ejemplo bool
}

// PlazoRespuestaLlamamiento es la propuesta calculada para RRHH. Sin catálogo
// configurado, Configurada es falso y el resto queda vacío.
type PlazoRespuestaLlamamiento struct {
	Configurada bool
	Regla       ReglaPlazoRespuesta
	CalculadoEn time.Time
	// UltimoDia es la fecha civil (AAAA-MM-DD) del último día del plazo.
	UltimoDia string
	// VenceEn es el último instante del plazo; VenceAntesDe, el primero en
	// que ya ha vencido.
	VenceEn      time.Time
	VenceAntesDe time.Time
}
