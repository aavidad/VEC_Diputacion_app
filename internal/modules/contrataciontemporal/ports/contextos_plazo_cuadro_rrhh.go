package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ContextoPlazoCuadroRRHH procede de la última publicación al corte,
// ya restringida al ámbito nominal. No transporta datos de persona.
type ContextoPlazoCuadroRRHH struct {
	ExpedienteRef     string
	VersionExpediente uint64
	FaseClave         domain.ClaveFase
	FaseDesde         time.Time
	Urgente           bool
}

const MaximoContextosPlazoCuadroRRHH = 100_000
