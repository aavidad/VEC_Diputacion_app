package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrResultadoMarcajeIndeterminado = errors.New("cronos resultado indeterminado; recuperar con la misma clave")
	ErrAuditoriaMarcajeNoDisponible  = errors.New("cronos auditoria de resultado no disponible")
)

// ResultadoEjecucionMarcaje describe lo observado por el runtime, no una
// denegación del PDP. No transporta material V3, errores SQL ni datos personales.
// DecisionRef y ContextoRef conservan la correlación con la petición autorizada.
// El destino debe ser la autoridad central durable, segregada del negocio.
type ResultadoEjecucionMarcaje struct {
	DecisionRef string
	ContextoRef string
	ActorRef    string
	PerfilRef   string
	Accion      string
	RecursoRef  string
	Resultado   string
	Causa       string
	ObservadaEn time.Time
}

// RegistroResultadoEjecucionMarcaje es obligatorio en la composición real.
// Sólo debe confirmar después del COMMIT de auditoría, en una transacción
// independiente de la ya terminada transacción de negocio. Un error nunca
// permite afirmar que el fallo se registró. No admite adaptadores en memoria.
type RegistroResultadoEjecucionMarcaje interface {
	RegistrarResultadoEjecucionMarcaje(context.Context, ResultadoEjecucionMarcaje) error
}
