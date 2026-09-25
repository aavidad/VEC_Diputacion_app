package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// EmisorIncidenciasTecnicas es el contrato común para declarar incidencias
// técnicas del catálogo cerrado de domain/incidencia_tecnica.go.
//
// Garantías exigibles a toda implementación:
//   - Emitir nunca bloquea al llamante ni realiza E/S en su goroutine; si no
//     puede aceptar la incidencia la descarta y la contabiliza.
//   - La solicitud se sanea contra el catálogo antes de retenerse.
//   - El fallo o la lentitud del destino nunca se propagan al llamante.
type EmisorIncidenciasTecnicas interface {
	Emitir(domain.SolicitudIncidenciaTecnica)
}

// MetricasEmisionIncidencias son contadores acumulados desde la creación del
// emisor. No llevan etiquetas por persona, expediente ni correlación.
type MetricasEmisionIncidencias struct {
	// Aceptadas entraron en la cola de emisión.
	Aceptadas uint64
	// Descartadas no entraron: cola llena o emisor cerrado.
	Descartadas uint64
	// Saneadas se aceptaron tras sustituir código, componente o etapa.
	Saneadas uint64
	// Escritas se entregaron completas al destino.
	Escritas uint64
	// FallosEscritura son escrituras rechazadas por el destino.
	FallosEscritura uint64
	// PendientesEnCola en el instante de la consulta.
	PendientesEnCola uint64
}

// ConsultaMetricasEmisionIncidencias expone los contadores internos.
type ConsultaMetricasEmisionIncidencias interface {
	MetricasEmision() MetricasEmisionIncidencias
}

// CierreEmisionIncidencias vacía lo pendiente y detiene el emisor. Si el
// contexto vence antes, devuelve su error sin bloquear más al llamante.
type CierreEmisionIncidencias interface {
	Cerrar(context.Context) error
}
